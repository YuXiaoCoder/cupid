package utils

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"time"
)

// 获取ZFTSL
func GetZFTSL() string {
	m := md5.New()
	rawData := []byte("zfsw_" + strconv.FormatInt(time.Now().Unix()/10, 10))
	m.Write(rawData)
	return hex.EncodeToString(m.Sum(nil))
}
