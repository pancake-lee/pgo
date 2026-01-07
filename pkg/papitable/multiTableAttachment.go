package papitable

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/iris/v12/x/errors"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type uploadAttachmentResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token    string `json:"token"`
		Name     string `json:"name"`
		Size     int64  `json:"size"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		MimeType string `json:"mimeType"`
		Preview  string `json:"preview"`
		Url      string `json:"url"`
	} `json:"data"`
}

// UploadAttachment 上传单个文件到 datasheet，返回服务器返回的数据（包含 token）
func (doc *MultiTableDoc) UploadAttachment(filePath string) (*uploadAttachmentResponse, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	defer f.Close()

	// file stat not needed because we stream via multipart writer

	url := fmt.Sprintf("%s/fusion/v1/datasheets/%s/attachments", g_baseUrl, doc.DatasheetId)

	// 使用 io.Pipe 流式写入 multipart body，避免一次性将文件读入内存
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	// 写入 multipart 的 goroutine
	go func() {
		defer pw.Close()
		// create form file field named "file"
		fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		// copy file content
		if _, err := io.Copy(fw, f); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		// close multipart writer to flush ending boundary
		if err := mw.Close(); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequest(http.MethodPost, url, pr)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	for k, v := range getTokenHeader() {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	body, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	var respData uploadAttachmentResponse
	err = json.Unmarshal(body, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	if !respData.Success {
		return &respData, plogger.LogErr(fmt.Errorf("upload attachment failed: code=%d, response=%s", respData.Code, respData.Message))
	}

	return &respData, nil
}

// Niubility! finally works!
// 因为官方文档提供的上传附件，只能multipart/form-data类型
// 但是这样上传后图片无法展示缩略图，那就无法使用相册视图了
// 所以我尝试扒apitable网页的上传逻辑，使用 presigned URL 上传，成功了！
// 这代码虽然有点臭，配合AI一点点调试出来的，暂时也没有封装的必要
// 相关源码位置：(git tag v1.13.0-beta.1)
// packages/room-server/src/fusion/vos/attachment.vo.ts
// packages/room-server/src/fusion/fusion.api.controller.ts
// backend-server/application/src/main/java/com/apitable/asset/controller/AssetCallbackController.java
func (doc *MultiTableDoc) UploadAttachmentWithPresignedUrl(
	fileName, contentType string, fileSize int64, fileReader io.Reader,
) (*uploadAttachmentResponse, error) {
	if fileSize == 0 {
		return nil, plogger.LogErr(errors.New("can not upload an empty file"))
	}

	// 1. 请求 presignedUrl
	reqUrl := fmt.Sprintf("%s/fusion/v1/datasheets/%s/attachments/presignedUrl?count=1", g_baseUrl, doc.DatasheetId)
	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	for k, v := range getTokenHeader() {
		req.Header.Set(k, v)
	}

	body, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	// plogger.Debugf("presigned URL response : %v", string(body))

	// --------------------------------------------------
	// 结构化解析 presigned 响应
	type assetVo struct {
		Token               string `json:"token"`
		UploadUrl           string `json:"uploadUrl"`
		UploadRequestMethod string `json:"uploadRequestMethod"`
	}

	// 先尝试解析为 { success, code, message, data: { results: [assetVo] } }
	var wrapResp struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Results []assetVo `json:"results"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapResp); err != nil {
		return nil, plogger.LogErr(err)
	}

	if len(wrapResp.Data.Results) == 0 {
		return nil, plogger.LogErr(fmt.Errorf("no presigned upload URL returned"))
	}

	uploadInfo := wrapResp.Data.Results[0]
	if uploadInfo.UploadRequestMethod != http.MethodPut {
		return nil, plogger.LogErr(fmt.Errorf("unsupported upload method: %s", uploadInfo.UploadRequestMethod))
	}

	// --------------------------------------------------
	// 2. 上传到 presigned URL
	// 因为minio来自于apitable的docker集群
	// 所以对于外部调用，我们自己要重新替换baseUrl
	parsedUpload, err := url.Parse(uploadInfo.UploadUrl)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	baseParsed, err := url.Parse(g_baseUrl)
	if err == nil {
		parsedUpload.Scheme = baseParsed.Scheme
		parsedUpload.Host = baseParsed.Host
	}
	targetURL := parsedUpload.String()

	// --------------------------------------------------
	putReq, err := http.NewRequest(http.MethodPut, targetURL, fileReader)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	putReq.Header.Set("Content-Type", contentType)
	putReq.ContentLength = fileSize
	// plogger.Debugf("req : %v", putReq)

	respBody, err := putil.HttpDo(putReq)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	if strings.Contains(string(respBody), "Error") {
		// s3接口可能返回200但是body是一份错误信息
		// 更准确的判断可以用Header:Etag判断
		plogger.Errorf("presigned put response body=%s", string(respBody))
		return nil, plogger.LogErr(fmt.Errorf("%s", string(respBody)))
	}

	// --------------------------------------------------
	// 还需要调用一个接口，apitable才完成附件的处理
	// POST /api/v1/asset/upload/callback
	// body: {"resourceKeys":["<token>"], "type":2}
	callbackUrl := fmt.Sprintf("%s/api/v1/asset/upload/callback", g_baseUrl)
	cbBody := map[string]any{
		"resourceKeys": []string{uploadInfo.Token},
		"type":         2,
	}
	cbReq, err := putil.NewHttpRequestJson(http.MethodPost, callbackUrl, getTokenHeader(), nil, cbBody)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	cbRespBytes, err := putil.HttpDo(cbReq)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	// 解析回调响应，结构为 array in data
	var cbResp struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    []struct {
			Token    string `json:"token"`
			Preview  any    `json:"preview"`
			MimeType string `json:"mimeType"`
			Size     int64  `json:"size"`
			Bucket   string `json:"bucket"`
			Name     string `json:"name"`
			Height   int    `json:"height"`
			Width    int    `json:"width"`
		} `json:"data"`
	}
	if err := json.Unmarshal(cbRespBytes, &cbResp); err != nil {
		return nil, plogger.LogErr(err)
	}
	if !cbResp.Success || len(cbResp.Data) == 0 {
		return nil, plogger.LogErr(fmt.Errorf("asset upload callback failed: code=%d, message=%s", cbResp.Code, cbResp.Message))
	}
	final := cbResp.Data[0]
	plogger.Debugf("upload[%v] success, token[%v] type[%v]",
		final.Name, final.Token, final.MimeType)

	// --------------------------------------------------
	var upResp uploadAttachmentResponse
	upResp.Success = true
	upResp.Code = 200
	upResp.Message = "SUCCESS"
	upResp.Data.Name = fileName
	upResp.Data.Token = final.Token
	upResp.Data.MimeType = final.MimeType
	upResp.Data.Size = final.Size
	upResp.Data.Width = final.Width
	upResp.Data.Height = final.Height
	upResp.Data.Preview = ""
	return &upResp, nil
}
