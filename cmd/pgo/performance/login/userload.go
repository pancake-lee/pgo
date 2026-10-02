package login

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/cmd/pgo/common"
	"github.com/pancake-lee/pgo/pkg/papp"
)

// User 保存压测用户的身份与鉴权信息。
type User struct {
	ID       int32  `json:"id"`
	UserName string `json:"userName"`
	Token    string `json:"token"`
}

// Prepare 并发创建内置数量的测试用户并保存批次清单。
func Prepare(
	ctx context.Context,
	client *common.Client,
	userCount int,
	path string,
) (*Manifest, error) {
	batchID, err := newBatchID()
	if err != nil {
		return nil, err
	}
	manifest := &Manifest{
		BatchID:   batchID,
		BaseURL:   client.BaseURL(),
		CreatedAt: time.Now().UTC(),
		Users:     make([]User, 0, userCount),
		path:      path,
	}

	jobList := make([]User, userCount)
	for index := range jobList {
		jobList[index].UserName = fmt.Sprintf("load_%s_%06d", batchID, index+1)
		if len(jobList[index].UserName) > 32 {
			return manifest, fmt.Errorf(
				"generated user name exceeds 32 characters: %s",
				jobList[index].UserName,
			)
		}
	}
	resultList, runErr := papp.RunConcurrent(
		ctx,
		jobList,
		func(ctx context.Context, user User) (User, error) {
			userInfo, token, loginErr := client.Login(ctx, user.UserName)
			createdUser := User{
				ID:       userInfo.ID,
				UserName: userInfo.UserName,
				Token:    token,
			}
			return createdUser, loginErr
		},
	)

	for _, result := range resultList {
		if result.Err == nil {
			manifest.Users = append(manifest.Users, result.Value)
		}
	}
	sort.Slice(manifest.Users, func(i, j int) bool {
		return manifest.Users[i].UserName <
			manifest.Users[j].UserName
	})
	err = manifest.write()
	if err != nil {
		return manifest, err
	}

	return manifest, errors.Join(runErr, joinResultErrors(resultList))
}

// Verify 重新登录批次用户并验证身份与受保护接口鉴权。
func Verify(ctx context.Context, client *common.Client, manifest *Manifest,
) error {
	err := manifest.validate()
	if err != nil {
		return err
	}

	resultList, runErr := papp.RunConcurrent(
		ctx,
		manifest.Users,
		func(ctx context.Context, expected User) (User, error) {
			userInfo, token, loginErr := client.Login(ctx, expected.UserName)
			if loginErr != nil {
				return User{}, loginErr
			}

			actual := User{
				ID:       userInfo.ID,
				UserName: userInfo.UserName,
				Token:    token,
			}
			identityMismatch := actual.ID != expected.ID ||
				actual.UserName != expected.UserName
			if identityMismatch {
				return User{}, fmt.Errorf(
					"identity mismatch for %s",
					expected.UserName,
				)
			}

			userList, _, getErr := client.GetUserList(ctx, actual.ID, actual.Token)
			if getErr != nil {
				return User{}, fmt.Errorf(
					"get user list for %s: %w",
					actual.UserName,
					getErr,
				)
			}

			userMismatch := len(userList) != 1 ||
				userList[0].ID != actual.ID ||
				userList[0].UserName != actual.UserName
			if userMismatch {
				return User{}, fmt.Errorf(
					"user list mismatch for %s",
					actual.UserName,
				)
			}

			return actual, nil
		},
	)
	return errors.Join(runErr, joinResultErrors(resultList))
}

// Cleanup 并发删除清单中的测试用户。
func Cleanup(ctx context.Context, client *common.Client, manifest *Manifest,
) error {
	err := manifest.validate()
	if err != nil {
		return err
	}

	resultList, runErr := papp.RunConcurrent(
		ctx,
		manifest.Users,
		func(ctx context.Context, user User) (User, error) {
			deleteErr := client.DelUserByIDList(ctx, user.ID, user.Token)
			return user, deleteErr
		},
	)
	err = errors.Join(runErr, joinResultErrors(resultList))
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	manifest.CleanedAt = &now
	return manifest.write()
}

// --------------------------------------------------
// Manifest 记录一个可验证和可清理的测试用户批次。
type Manifest struct {
	Version   int        `json:"version"`
	BatchID   string     `json:"batchID"`
	BaseURL   string     `json:"baseURL"`
	CreatedAt time.Time  `json:"createdAt"`
	CleanedAt *time.Time `json:"cleanedAt,omitempty"`
	Users     []User     `json:"users"`
	path      string
}

// validate 校验批次元数据及用户身份信息的完整性。
func (manifest *Manifest) validate() error {
	if manifest == nil || manifest.BatchID == "" || manifest.BaseURL == "" {
		return errors.New("invalid manifest metadata")
	}
	for _, user := range manifest.Users {
		if user.ID == 0 || user.UserName == "" || user.Token == "" {
			return fmt.Errorf("invalid user in batch %s", manifest.BatchID)
		}
	}
	return nil
}

// read 从文件读取并校验测试用户批次清单。
func (manifest *Manifest) read(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, manifest)
	if err != nil {
		return err
	}

	err = manifest.validate()
	if err != nil {
		return err
	}

	manifest.path = path
	return nil
}

// write 覆盖保存测试用户批次清单。
func (manifest *Manifest) write() error {
	if manifest.path == "" {
		return errors.New("manifest path is required")
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	directory := filepath.Dir(manifest.path)
	err = os.MkdirAll(directory, 0o700)
	if err != nil && directory != "." {
		return err
	}

	return os.WriteFile(manifest.path, append(content, '\n'), 0o600)
}

// vegetaTarget 描述一条 Vegeta JSON 格式的请求目标。
type vegetaTarget struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Body   []byte              `json:"body"`
	Header map[string][]string `json:"header"`
}

// WriteTargets 将测试用户转换为 Vegeta JSON 请求目标。
func (manifest *Manifest) WriteTargets(path string) error {
	err := manifest.validate()
	if err != nil {
		return err
	}

	directory := filepath.Dir(path)
	err = os.MkdirAll(directory, 0o700)
	if err != nil && directory != "." {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	for _, user := range manifest.Users {
		body, marshalErr := json.Marshal(map[string]string{
			"userName": user.UserName,
		})
		if marshalErr != nil {
			_ = file.Close()
			return marshalErr
		}
		target := vegetaTarget{
			Method: http.MethodPost,
			URL:    strings.TrimRight(manifest.BaseURL, "/") + "/user/token",
			Body:   body,
			Header: map[string][]string{"Content-Type": {"application/json"}},
		}
		err = encoder.Encode(target)
		if err != nil {
			_ = file.Close()
			return err
		}
	}
	return file.Close()
}

// --------------------------------------------------
// joinResultErrors 合并并发用户任务产生的全部错误。
func joinResultErrors(resultList []papp.RunResult[User]) error {
	errorList := make([]error, 0)
	for _, result := range resultList {
		if result.Err != nil {
			errorList = append(errorList, result.Err)
		}
	}
	return errors.Join(errorList...)
}

// newBatchID 生成适合用户名使用的随机批次标识。
func newBatchID() (string, error) {
	randomBytes := make([]byte, 2)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	timestamp := time.Now().UTC().Format("060102150405")
	return timestamp + hex.EncodeToString(randomBytes), nil
}
