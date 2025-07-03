package main

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pweixin/wx_cb_server/xml_callback"

	"log"
	"net/http"
	"net/url"
	"strings"
)

var token = ""
var receiverId = ""
var encodingAeskey = ""

func getString(str, endstr string, start int, msg *string) int {
	end := strings.Index(str, endstr)
	*msg = str[start:end]
	return end + len(endstr)
}

func VerifyURL(w http.ResponseWriter, r *http.Request) {
	//httpstr := `&{GET /?msg_signature=825075c093249d5a60967fe4a613cae93146636b&timestamp=1597998748&nonce=1597483820&echostr=neLB8CftccHiz19tluVb%2BUBnUVMT3xpUMZU8qvDdD17eH8XfEsbPYC%2FkJyPsZOOc6GdsCeu8jSIa2noSJ%2Fez2w%3D%3D HTTP/1.1 1 1 map[Cache-Control:[no-cache] Accept:[*/*] Pragma:[no-cache] User-Agent:[Mozilla/4.0]] 0x86c180 0 [] false 100.108.211.112:8893 map[] map[] <nil> map[] 100.108.79.233:59663 /?msg_signature=825075c093249d5a60967fe4a613cae93146636b&timestamp=1597998748&nonce=1597483820&echostr=neLB8CftccHiz19tluVb%2BUBnUVMT3xpUMZU8qvDdD17eH8XfEsbPYC%2FkJyPsZOOc6GdsCeu8jSIa2noSJ%2Fez2w%3D%3D <nil>}`
	plogger.Debug(r, r.Body)
	httpstr := r.URL.RawQuery
	start := strings.Index(httpstr, "msg_signature=")
	start += len("msg_signature=")

	var msg_signature string
	next := getString(httpstr, "&timestamp=", start, &msg_signature)

	var timestamp string
	next = getString(httpstr, "&nonce=", next, &timestamp)

	var nonce string
	next = getString(httpstr, "&echostr=", next, &nonce)

	echostr := httpstr[next:]

	echostr, _ = url.QueryUnescape(echostr)
	plogger.Debug(msg_signature, timestamp, nonce, echostr, next)

	wxcpt := xml_callback.NewWXBizMsgCrypt(token, encodingAeskey, receiverId, xml_callback.XmlType)
	echoStr, cryptErr := wxcpt.VerifyURL(msg_signature, timestamp, nonce, echostr)
	if nil != cryptErr {
		plogger.Error("verifyUrl fail", cryptErr)
	}
	plogger.Debug("verifyUrl success echoStr", string(echoStr))
	fmt.Fprint(w, string(echoStr))

}

type SheetChangeRecord struct {
	ToUsername   string `xml:"ToUserName"`
	FromUsername string `xml:"FromUserName"`
	MsgType      string `xml:"MsgType"`

	// 表格修改通知有
	Event      string `xml:"Event"`
	ChangeType string `xml:"ChangeType"`
	CreateTime uint32 `xml:"CreateTime"`
	DocId      string `xml:"DocId"`
	SheetId    string `xml:"SheetId"`
	RecordId   string `xml:"RecordId"`

	// 未必有
	Content string `xml:"Content"`
	Msgid   string `xml:"MsgId"`
	Agentid uint32 `xml:"AgentId"`
}

func MsgHandler(w http.ResponseWriter, r *http.Request) {
	httpstr := r.URL.RawQuery
	start := strings.Index(httpstr, "msg_signature=")
	start += len("msg_signature=")

	var msg_signature string
	next := getString(httpstr, "&timestamp=", start, &msg_signature)

	var timestamp string
	next = getString(httpstr, "&nonce=", next, &timestamp)

	nonce := httpstr[next:]
	// plogger.Debugf("msgSign[%v] t[%v] nonce[%v]", msg_signature, timestamp, nonce)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		plogger.Error("ReadAll fail", err)
		return
	}

	// plogger.Debugf("body: %v", string(body))

	wxcpt := xml_callback.NewWXBizMsgCrypt(token, encodingAeskey, receiverId, xml_callback.XmlType)

	msg, cryptErr := wxcpt.DecryptMsg(msg_signature, timestamp, nonce, body)
	if cryptErr != nil {
		plogger.Error("DecryptMsg fail", cryptErr)
		return
	}

	// plogger.Debugf("decrypt msg: %v", string(msg))

	var req SheetChangeRecord
	err = xml.Unmarshal(msg, &req)
	if nil != err {
		plogger.Error("Unmarshal fail", err)
		return
	}

	if req.MsgType == "event" &&
		req.Event == "smart_sheet_change" &&
		req.ChangeType == "update_record" {

		plogger.Debugf("Received smart_sheet_change event: %v", req)

	} else {
		plogger.Debugf("Received unknown event: %v", req)
	}
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	httpstr := r.URL.RawQuery
	echo := strings.Index(httpstr, "echostr")
	if echo != -1 {
		VerifyURL(w, r)
	} else {
		MsgHandler(w, r)
	}
}

func main() {
	pconfig.MustInitConfig("")
	token = pconfig.GetStringM("WX.cbToken")
	encodingAeskey = pconfig.GetStringM("WX.cbEncodingAESKey")
	receiverId = pconfig.GetStringM("WX.cbReceiverId")

	plogger.InitServiceLogger(false)

	http.HandleFunc("/", CallbackHandler) //      设置访问路由

	plogger.Debug("Starting server on :8893")
	log.Fatal(http.ListenAndServe(":8893", nil))
}
