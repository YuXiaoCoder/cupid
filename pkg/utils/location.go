package utils

import (
	"cupid/pkg/xhttp"
	"encoding/json"
	"net/http"

	"github.com/bitly/go-simplejson"
)

// 通过百度API获取指定城市的经纬度
func GetLocation(city string) map[string]interface{} {
	location := map[string]interface{}{
		"lat": json.Number("30.578994"),
		"lng": json.Number("104.072747"),
	}

	url := "http://api.map.baidu.com/geocoder"

	queries := map[string]string{
		"city":    city,
		"address": city,
		"output":  "json",
		"key":     "nqQhyG3tAvrD8RmEpGUHhq6kFkTTSGfk",
	}

	retry := 10
	for i := 0; i < retry; i++ {
		data, err := xhttp.Do(url, http.MethodGet, nil, queries, nil)
		if err != nil {
			continue
		}

		dataJSON, err := simplejson.NewJson(data)
		if err != nil {
			continue
		}

		if dataJSON.Get("status").MustString() != "OK" {
			continue
		}

		if value, err := dataJSON.Map(); err != nil || value == nil {
			continue
		}

		location = dataJSON.GetPath("result", "location").MustMap()

		break
	}

	return location
}
