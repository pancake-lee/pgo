package login

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/antihax/optional"
	"github.com/pancake-lee/pgo/cmd/pgo/swagger"
)

const manifestVersion = 1

type User struct {
	ID       int32  `json:"id"`
	UserName string `json:"userName"`
	Token    string `json:"token"`
}

type Manifest struct {
	Version   int        `json:"version"`
	BatchID   string     `json:"batchID"`
	BaseURL   string     `json:"baseURL"`
	CreatedAt time.Time  `json:"createdAt"`
	CleanedAt *time.Time `json:"cleanedAt,omitempty"`
	Users     []User     `json:"users"`
}

type Client struct {
	baseURL   string
	apiClient *swagger.APIClient
}

type runResult struct {
	user User
	err  error
}

type vegetaTarget struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Body   []byte              `json:"body"`
	Header map[string][]string `json:"header"`
}

func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid base URL %q", baseURL)
	}
	if timeout <= 0 {
		return nil, errors.New("timeout must be positive")
	}
	configuration := swagger.NewConfiguration()
	configuration.BasePath = baseURL
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Batch preparation and cleanup are control traffic, not measured load. Avoid reusing
	// a connection that an older HTTP server may close without advertising Connection: close.
	transport.DisableKeepAlives = true
	configuration.HTTPClient = &http.Client{Timeout: timeout, Transport: transport}
	return &Client{baseURL: baseURL, apiClient: swagger.NewAPIClient(configuration)}, nil
}

func Prepare(ctx context.Context, client *Client, path, batchID string, count, concurrency int) (*Manifest, error) {
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}
	if concurrency <= 0 {
		return nil, errors.New("concurrency must be positive")
	}
	if batchID == "" {
		var err error
		batchID, err = newBatchID()
		if err != nil {
			return nil, err
		}
	}
	if !isValidBatchID(batchID) {
		return nil, errors.New("batch must contain only letters, digits, underscore, or hyphen")
	}
	manifest := &Manifest{
		Version:   manifestVersion,
		BatchID:   batchID,
		BaseURL:   client.baseURL,
		CreatedAt: time.Now().UTC(),
		Users:     make([]User, 0, count),
	}
	if err := WriteManifest(path, manifest); err != nil {
		return nil, err
	}

	jobList := make([]User, count)
	for index := range jobList {
		jobList[index].UserName = fmt.Sprintf("load_%s_%06d", batchID, index+1)
		if len(jobList[index].UserName) > 32 {
			return manifest, fmt.Errorf("generated user name exceeds 32 characters: %s", jobList[index].UserName)
		}
	}
	var persistenceErr error
	resultList := runConcurrent(ctx, jobList, concurrency, func(ctx context.Context, user User) (User, error) {
		return client.login(ctx, user.UserName)
	}, func(result runResult) {
		if result.err != nil || persistenceErr != nil {
			return
		}
		manifest.Users = append(manifest.Users, result.user)
		persistenceErr = WriteManifest(path, manifest)
	})

	var errorList []error
	for _, result := range resultList {
		if result.err != nil {
			errorList = append(errorList, result.err)
		}
	}
	if persistenceErr != nil {
		return manifest, persistenceErr
	}
	sort.Slice(manifest.Users, func(i, j int) bool { return manifest.Users[i].UserName < manifest.Users[j].UserName })
	if err := WriteManifest(path, manifest); err != nil {
		return manifest, err
	}
	return manifest, errors.Join(errorList...)
}

func Verify(ctx context.Context, client *Client, manifest *Manifest, concurrency int) error {
	if err := validateManifest(manifest); err != nil {
		return err
	}
	if concurrency <= 0 {
		return errors.New("concurrency must be positive")
	}
	if err := client.verifyAuthRejection(ctx); err != nil {
		return err
	}
	resultList := runConcurrent(ctx, manifest.Users, concurrency, func(ctx context.Context, expected User) (User, error) {
		actual, err := client.login(ctx, expected.UserName)
		if err != nil {
			return User{}, err
		}
		if actual.ID != expected.ID || actual.UserName != expected.UserName {
			return User{}, fmt.Errorf("identity mismatch for %s", expected.UserName)
		}
		if err = client.verifyProtectedUser(ctx, actual); err != nil {
			return User{}, err
		}
		return actual, nil
	}, nil)
	return joinResultErrors(resultList)
}

func Cleanup(ctx context.Context, client *Client, manifest *Manifest, concurrency int) error {
	if err := validateManifest(manifest); err != nil {
		return err
	}
	if concurrency <= 0 {
		return errors.New("concurrency must be positive")
	}
	resultList := runConcurrent(ctx, manifest.Users, concurrency, func(ctx context.Context, user User) (User, error) {
		return user, client.delUser(ctx, user)
	}, nil)
	return joinResultErrors(resultList)
}

func ReadManifest(path string) (*Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err = json.Unmarshal(content, &manifest); err != nil {
		return nil, err
	}
	if err = validateManifest(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func WriteManifest(path string, manifest *Manifest) error {
	if path == "" {
		return errors.New("manifest path is required")
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil && filepath.Dir(path) != "." {
		return err
	}
	temporaryPath := path + ".tmp"
	if err = os.WriteFile(temporaryPath, content, 0o600); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func WriteTargets(path string, manifest *Manifest) error {
	if err := validateManifest(manifest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil && filepath.Dir(path) != "." {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	for _, user := range manifest.Users {
		body, marshalErr := json.Marshal(map[string]string{"userName": user.UserName})
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
		if err = encoder.Encode(target); err != nil {
			_ = file.Close()
			return err
		}
	}
	return file.Close()
}

func (client *Client) login(ctx context.Context, userName string) (User, error) {
	request := swagger.ApiLoginRequest{UserName: userName}
	response, _, err := client.apiClient.UserApi.UserLogin(ctx, request)
	if err != nil {
		return User{}, fmt.Errorf("login %s: %w", userName, err)
	}
	if response.User == nil || response.User.ID == 0 || response.User.UserName != userName || response.Token == "" {
		return User{}, fmt.Errorf("login %s returned incomplete identity or token", userName)
	}
	return User{ID: response.User.ID, UserName: response.User.UserName, Token: response.Token}, nil
}

func (client *Client) verifyProtectedUser(ctx context.Context, user User) error {
	authContext := context.WithValue(ctx, swagger.ContextAccessToken, user.Token)
	options := &swagger.UserCURDApiUserCURDGetUserListOpts{IDList: optional.NewInterface([]int32{user.ID})}
	response, _, err := client.apiClient.UserCURDApi.UserCURDGetUserList(authContext, options)
	if err != nil {
		return fmt.Errorf("verify protected access for %s: %w", user.UserName, err)
	}
	if len(response.UserList) != 1 || response.UserList[0].ID != user.ID || response.UserList[0].UserName != user.UserName {
		return fmt.Errorf("protected lookup mismatch for %s", user.UserName)
	}
	return nil
}

func (client *Client) verifyAuthRejection(ctx context.Context) error {
	for _, token := range []string{"", "invalid-token"} {
		requestContext := ctx
		if token != "" {
			requestContext = context.WithValue(ctx, swagger.ContextAccessToken, token)
		}
		options := &swagger.UserCURDApiUserCURDGetUserListOpts{IDList: optional.NewInterface([]int32{1})}
		_, response, err := client.apiClient.UserCURDApi.UserCURDGetUserList(requestContext, options)
		if err == nil || response == nil || response.StatusCode < http.StatusBadRequest {
			return fmt.Errorf("protected endpoint accepted rejected token case %q", token)
		}
	}
	return nil
}

func (client *Client) delUser(ctx context.Context, user User) error {
	authContext := context.WithValue(ctx, swagger.ContextAccessToken, user.Token)
	options := &swagger.UserCURDApiUserCURDDelUserByIDListOpts{IDList: optional.NewInterface([]int32{user.ID})}
	_, _, err := client.apiClient.UserCURDApi.UserCURDDelUserByIDList(authContext, options)
	return err
}

func runConcurrent(
	ctx context.Context,
	userList []User,
	concurrency int,
	run func(context.Context, User) (User, error),
	onResult func(runResult),
) []runResult {
	jobChannel := make(chan User)
	resultChannel := make(chan runResult)
	workerCount := min(concurrency, len(userList))
	var waitGroup sync.WaitGroup
	for range workerCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for user := range jobChannel {
				resultUser, err := run(ctx, user)
				resultChannel <- runResult{user: resultUser, err: err}
			}
		}()
	}
	go func() {
		defer close(jobChannel)
		for _, user := range userList {
			select {
			case jobChannel <- user:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		waitGroup.Wait()
		close(resultChannel)
	}()

	resultList := make([]runResult, 0, len(userList))
	for result := range resultChannel {
		resultList = append(resultList, result)
		if onResult != nil {
			onResult(result)
		}
	}
	if err := ctx.Err(); err != nil {
		resultList = append(resultList, runResult{err: err})
	}
	return resultList
}

func joinResultErrors(resultList []runResult) error {
	errorList := make([]error, 0)
	for _, result := range resultList {
		if result.err != nil {
			errorList = append(errorList, result.err)
		}
	}
	return errors.Join(errorList...)
}

func validateManifest(manifest *Manifest) error {
	if manifest == nil || manifest.Version != manifestVersion || manifest.BatchID == "" || manifest.BaseURL == "" {
		return errors.New("invalid manifest metadata")
	}
	for _, user := range manifest.Users {
		if user.ID == 0 || user.UserName == "" || user.Token == "" {
			return fmt.Errorf("invalid user in batch %s", manifest.BatchID)
		}
	}
	return nil
}

func newBatchID() (string, error) {
	randomBytes := make([]byte, 2)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return time.Now().UTC().Format("060102150405") + hex.EncodeToString(randomBytes), nil
}

func isValidBatchID(batchID string) bool {
	if batchID == "" || len(batchID) > 18 {
		return false
	}
	for _, character := range batchID {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}
