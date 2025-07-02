package putil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func NewHttpRequestJson(method, rawURL string, header, querys map[string]string, body any) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return NewHttpRequest(method, rawURL, header, querys, string(jsonBody))
}

func NewHttpRequest(method, rawURL string, header, querys map[string]string, body string) (*http.Request, error) {
	// fmt.Println("body : ", body)
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	query := parsedURL.Query()
	for k, v := range querys {
		query.Set(k, v)
	}
	parsedURL.RawQuery = query.Encode()

	req, err := http.NewRequest(method, parsedURL.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}

	return req, nil
}

func HttpDo(req *http.Request) (bodyBytes []byte, err error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return bodyBytes, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New("status code : " + IntToStr(resp.StatusCode))
	}

	if resp != nil && resp.Body != nil {
		bodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return bodyBytes, err
		}
	}
	return bodyBytes, nil
}
