package xhttp

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// 执行请求
func Do(apiURL string, method string, headers map[string]string, params map[string]string, body map[string]interface{}) (data []byte, err error) {
	// Reader
	var ioReader bytes.Reader

	if body != nil {
		// 序列化请求体
		buffer, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		ioReader = *bytes.NewReader(buffer)
	}

	// 初始化请求
	request, err := http.NewRequest(method, apiURL, &ioReader)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	// 设置请求参数
	if params != nil {
		query := request.URL.Query()
		for k, v := range params {
			query.Add(k, v)
		}
		request.URL.RawQuery = query.Encode()
	}

	// 初始化客户端
	client := &http.Client{}

	// 发送请求
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}
