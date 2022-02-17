package utils

import (
	"bytes"
	"encoding/json"
	"os"
)

// 将数据写入文件
func WriteJSONToFile(filename string, data interface{}) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入数据
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", string(bytes.Repeat([]byte(" "), 2)))
	if err = encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

// 从文件中读取数据
func ReadJSONFromFile(filename string) (map[string]interface{}, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取数据
	data := make(map[string]interface{})
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// 判断文件是否存在
func FileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
