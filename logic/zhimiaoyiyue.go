package logic

import (
	"cupid/pkg/configs"
	"cupid/pkg/utils"
	"cupid/pkg/xhttp"
	"cupid/resource"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"

	"go.uber.org/zap"
)

// 知苗易约
type ZMYYEngine struct{}

// 获取知苗易约的引擎
func GetZMYYEngine() *ZMYYEngine {
	return new(ZMYYEngine)
}

func (engine *ZMYYEngine) Sniff() (results []map[string]string, err error) {
	// 获取城市的编码
	var allCityCodes map[string]interface{}
	if !utils.FileExist(resource.CityCodeFile) {
		zap.L().Fatal("unable to get city code from file", zap.String("file", "city.json"))
	} else {
		allCityCodes, err = utils.ReadJSONFromFile(resource.CityCodeFile)
		if err != nil {
			zap.L().Error("unable to get city code from file", zap.String("file", resource.CityCodeFile), zap.Error(err))
			return nil, err
		}
	}

	// 待嗅探的区域
	var cityCodes map[string]interface{}
	if len(configs.AllConfig.Sniff.Regions) <= 0 {
		cityCodes = allCityCodes
	} else {
		cityCodes = make(map[string]interface{})

		for _, region := range configs.AllConfig.Sniff.Regions {
			regionSlice := strings.Split(region, "-")

			if len(regionSlice) == 1 {
				// 嗅探整个省或直辖市
				if _, ok := allCityCodes[regionSlice[0]]; ok {
					cityCodes[regionSlice[0]] = allCityCodes[regionSlice[0]]
					continue
				}
			} else if len(regionSlice) == 2 {
				// 嗅探指定市
				if _, ok := allCityCodes[regionSlice[0]]; ok {
					cityCodes[regionSlice[0]] = make([]interface{}, 0)
				}

				for _, v := range allCityCodes[regionSlice[0]].([]interface{}) {
					city := v.(map[string]interface{})
					if city["name"].(string) == regionSlice[1] {
						cityCodes[regionSlice[0]] = append(cityCodes[regionSlice[0]].([]interface{}), city)
						break
					}
				}

				// 若未匹配到指定的市，则为空
				if len(cityCodes[regionSlice[0]].([]interface{})) > 0 {
					continue
				} else {
					delete(cityCodes, regionSlice[0])
				}
			}
			zap.L().Error("未能正确匹配待嗅探的区域，请检查", zap.String("region", region))
		}
	}

	// 嗅探疫苗
	results = make([]map[string]string, 0)
	for province, cities := range cityCodes {
		item := cities.([]interface{})
		results = append(results, engine.hasSeckill(province, item)...)
	}

	return results, nil
}

func (engine *ZMYYEngine) SecKill() error {
	log.Println("ZMYYEngine's SecKill")
	return nil
}

// 判断是否有秒杀信息
func (engine *ZMYYEngine) hasSeckill(province string, cities []interface{}) []map[string]string {
	headers := map[string]string{
		"User-Agent": resource.UserAgent,
		"Referer":    resource.ZMYYReferer,
	}

	results := make([]map[string]string, 0)
	for _, v := range cities {
		city := v.(map[string]interface{})

		if flag, ok := resource.SpecialAdministrativeRegion[city["name"].(string)]; ok && flag {
			city["value"] = fmt.Sprintf("%v01", city["value"])
		} else {
			city["value"] = fmt.Sprintf("%v00", city["value"])
		}

		if configs.AllConfig.Basic.Debug {
			zap.L().Debug("当前探测的城市", zap.Any("province", province), zap.Any("city", city))
		}

		location := city["location"].(interface{}).(map[string]interface{})

		// 设置请求头
		headers["zftsl"] = utils.GetZFTSL()

		queries := map[string]string{
			"id":       "0",
			"product":  "1",
			"act":      "CustomerList",
			"city":     fmt.Sprintf("[\"%s\",\"%s\",\"%s\"]", province, city["name"].(string), ""),
			"cityCode": city["value"].(string),
			"lat":      strconv.FormatFloat(location["lat"].(float64), 'f', -1, 64),
			"lng":      strconv.FormatFloat(location["lng"].(float64), 'f', -1, 64),
		}

		// 获取指定地区的医院列表
		data, err := xhttp.Do(resource.ZMYYRootURL, http.MethodGet, headers, queries, nil)
		if err != nil {
			zap.L().Error("failed to do request", zap.Any("city", city["name"]), zap.Error(err))
			return results
		}

		dataJSON, err := simplejson.NewJson(data)
		if err != nil {
			zap.L().Error("failed to unmarshal data", zap.Any("city", city["name"]), zap.Error(err))
			return results
		}

		if dataJSON.Get("status").MustInt() != http.StatusOK {
			zap.L().Error("unable to get seckill info", zap.String("province", province), zap.Any("city", city["name"]), zap.Any("data", dataJSON.MustMap()), zap.Error(err))
			return results
		}

		// 医院列表
		hospitals := dataJSON.Get("list").MustArray()

		for _, vv := range hospitals {
			hospital := vv.(map[string]interface{})

			// 设置请求头
			headers["zftsl"] = utils.GetZFTSL()

			// 获取某医院内疫苗情况
			productQueries := map[string]string{
				"id":  hospital["id"].(json.Number).String(),
				"act": "CustomerProduct",
				"lat": strconv.FormatFloat(location["lat"].(float64), 'f', -1, 64),
				"lng": strconv.FormatFloat(location["lng"].(float64), 'f', -1, 64),
			}

			// 获取指定医院的所有疫苗
			productData, err := xhttp.Do(resource.ZMYYRootURL, http.MethodGet, headers, productQueries, nil)
			if err != nil {
				zap.L().Error("failed to do request", zap.Any("city", city["name"]), zap.Error(err))
				continue
			}

			productDataJSON, err := simplejson.NewJson(productData)
			if err != nil {
				zap.L().Error("failed to unmarshal data", zap.Any("city", city["name"]), zap.String("data", string(productData)), zap.Error(err))
				continue
			}

			if productDataJSON.Get("status").MustInt() != http.StatusOK {
				zap.L().Error("unable to get seckill info", zap.String("province", province), zap.Any("city", city["name"]), zap.Any("hospital", hospital), zap.Any("data", productDataJSON.MustMap()), zap.Error(err))
				continue
			}

			for _, vvv := range productDataJSON.Get("list").MustArray() {
				vaccine := vvv.(map[string]interface{})

				if strings.Contains(vaccine["text"].(string), "九价") {
					result := map[string]string{
						"city":       city["name"].(string),                // 城市
						"seckill":    vaccine["id"].(json.Number).String(), // 秒杀编号
						"vaccine":    vaccine["text"].(string),             // 疫苗名称
						"hospital":   hospital["cname"].(string),           // 医院名称
						"start_time": vaccine["date"].(string),             // 开始时间
						"source":     "知苗易约",                               // 渠道
					}
					results = append(results, result)
				} else if configs.AllConfig.Basic.Debug {
					zap.L().Debug("当前城市的秒杀信息", zap.Any("city", city["name"]), zap.String("vaccine", vaccine["text"].(string)))
					continue
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(1 * time.Second)
	}
	return results
}
