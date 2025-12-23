package putil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

func GetUrlQueryString(req any) (querys map[string]string) {
	if req == nil {
		return nil
	}
	querys = make(map[string]string)

	v := reflect.ValueOf(req)
	t := reflect.TypeOf(req)

	// 如果是指针，获取指向的值
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return querys
		}
		v = v.Elem()
		t = t.Elem()
	}

	// 只处理结构体
	if v.Kind() != reflect.Struct {
		return querys
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 跳过未导出的字段
		if !field.CanInterface() {
			continue
		}

		// 获取字段名，优先使用 json tag
		fieldName := fieldType.Name
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
			if jsonTag == "-" {
				continue
			}
			// 解析 json tag，取第一部分作为字段名
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// 处理不同类型的字段
		switch field.Kind() {
		case reflect.String:
			if str := field.String(); str != "" {
				querys[fieldName] = str
			}
		case reflect.Int, reflect.Int32, reflect.Int64:
			if val := field.Int(); val != 0 {
				querys[fieldName] = strconv.FormatInt(val, 10)
			}
		case reflect.Bool:
			querys[fieldName] = strconv.FormatBool(field.Bool())
		case reflect.Float32, reflect.Float64:
			if val := field.Float(); val != 0 {
				querys[fieldName] = strconv.FormatFloat(val, 'f', -1, 64)
			}
		case reflect.Struct:
			// 处理结构体类型，将结构体转换为JSON字符串
			if !field.IsZero() {
				if jsonBytes, err := json.Marshal(field.Interface()); err == nil {
					querys[fieldName] = string(jsonBytes)
				}
			}
		case reflect.Slice, reflect.Array:
			// 处理数组类型
			if field.Len() > 0 {
				var values []string
				for j := 0; j < field.Len(); j++ {
					elem := field.Index(j)
					switch elem.Kind() {
					case reflect.String:
						if str := elem.String(); str != "" {
							values = append(values, str)
						}
					case reflect.Int, reflect.Int32, reflect.Int64:
						values = append(values, strconv.FormatInt(elem.Int(), 10))
					case reflect.Bool:
						values = append(values, strconv.FormatBool(elem.Bool()))
					case reflect.Float32, reflect.Float64:
						values = append(values, strconv.FormatFloat(elem.Float(), 'f', -1, 64))
					case reflect.Struct:
						// 处理结构体数组，将每个结构体转换为JSON字符串
						if jsonBytes, err := json.Marshal(elem.Interface()); err == nil {
							values = append(values, string(jsonBytes))
						}
					}
				}
				if len(values) > 0 {
					querys[fieldName] = strings.Join(values, ",")
				}
			}
		}
	}

	return querys
}

func NewHttpRequestJson(method, rawURL string, header, querys map[string]string, body any) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	if header == nil {
		header = make(map[string]string)
	}
	header["Content-Type"] = "application/json"
	return NewHttpRequest(method, rawURL, header, querys, string(jsonBody))
}

func NewHttpRequest(method, rawURL string, header, querys map[string]string, body string) (*http.Request, error) {
	// fmt.Println("url  : ", rawURL)
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
