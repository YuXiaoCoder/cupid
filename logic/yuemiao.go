package logic

import (
	"cupid/pkg/configs"
	"cupid/pkg/utils"
	"cupid/pkg/xhttp"
	"cupid/resource"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/golang-module/carbon"

	"github.com/bitly/go-simplejson"
	"go.uber.org/zap"
)

// 约苗
type YMEngine struct{}

// 获取约苗的引擎
func GetYMEngine() *YMEngine {
	return new(YMEngine)
}

// 探测哪些城市有秒杀信息
func (engine *YMEngine) Sniff() (results []map[string]string, err error) {
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

	// 多协程采集指标
	channels := make(chan []map[string]string, len(cityCodes))

	// 限流器
	limiter := rate.NewLimiter(rate.Every(500*time.Millisecond), 5)

	// 协程管理器
	var wg sync.WaitGroup
	for province, cities := range cityCodes {
		for {
			if limiter.Allow() {
				// 增加信号量，信号量 = 协程数量 = 集群数量
				wg.Add(1)

				item := cities.([]interface{})
				go engine.hasSeckill(&wg, channels, province, item)
				break
			}
		}
	}

	// 等待所有协程结束
	wg.Wait()

	// 关闭通道
	close(channels)

	// 遍历通道
	results = make([]map[string]string, 0)
	for item := range channels {
		results = append(results, item...)
	}

	return results, nil
}

// 秒杀
func (engine *YMEngine) SecKill() error {
	// 探测哪些城市有秒杀信息
	zap.L().Info("正在嗅探约苗当前哪些城市有秒杀信息")
	vaccines, err := engine.Sniff()
	if err != nil {
		zap.L().Error("无法获取约苗当前哪些城市有秒杀信息", zap.Error(err))
		return err
	}

	// 匹配待秒杀的疫苗
	var vaccine map[string]string
	for _, v := range vaccines {
		if v["seckill"] == configs.AllConfig.YM.SeckillID {
			vaccine = v
			break
		}
	}

	// 未匹配到指定的疫苗
	if vaccine == nil {
		zap.L().Error("未匹配到指定的疫苗", zap.String("seckill_id", configs.AllConfig.YM.SeckillID))
		return fmt.Errorf("未匹配到指定的疫苗")
	}

	// 解析秒杀时间
	seckillStartTime := carbon.ParseByLayout(vaccine["start_time"], carbon.DateTimeFormat)

	headers := map[string]string{
		"User-Agent": resource.UserAgent,
	}

	for {
		data, err := xhttp.Do(resource.YMTimestampURL, http.MethodGet, headers, nil, nil)
		if err != nil {
			zap.L().Error("failed to do request", zap.Error(err))
			return err
		}

		dataJSON, err := simplejson.NewJson(data)
		if err != nil {
			zap.L().Error("failed to unmarshal data", zap.Error(err))
			return err
		}

		serviceTimestamp := carbon.Now().TimestampWithMillisecond()
		if dataJSON.Get("code").MustString() == resource.YMResponseOKCode {
			serviceTimestamp = dataJSON.Get("data").MustInt64()
		}

		var seckillIntervalTime int64 = 400
		diffInSeconds := serviceTimestamp/1000 - seckillStartTime.Timestamp()
		diffInMilliseconds := serviceTimestamp - seckillStartTime.TimestampWithMillisecond()
		if diffInMilliseconds >= 0 {
			return fmt.Errorf("秒杀时间已过，欢迎下次使用")
		} else if diffInMilliseconds >= -seckillIntervalTime {
			zap.L().Info("开始订购疫苗", zap.String("誓言", "两情若是久长时，又岂在朝朝暮暮"))
			break
		} else if diffInMilliseconds >= -3*seckillIntervalTime {
			zap.L().Info(fmt.Sprintf("将在离秒杀时间剩余%d毫秒时开始发起请求", seckillIntervalTime), zap.String("休息时间", fmt.Sprintf("%d毫秒", 50)), zap.String("剩余时间", fmt.Sprintf("%d毫秒", utils.Abs(diffInMilliseconds))))
			time.Sleep(50 * time.Millisecond)
		} else if diffInMilliseconds >= -3*1000 {
			zap.L().Info(fmt.Sprintf("将在离秒杀时间剩余%d毫秒时开始发起请求", seckillIntervalTime), zap.String("休息时间", fmt.Sprintf("%d毫秒", 100)), zap.String("剩余时间", fmt.Sprintf("%d毫秒", utils.Abs(diffInMilliseconds))))
			time.Sleep(100 * time.Millisecond)
		} else if diffInMilliseconds >= -10*1000 {
			zap.L().Info(fmt.Sprintf("将在离秒杀时间剩余%d毫秒时开始发起请求", seckillIntervalTime), zap.String("休息时间", fmt.Sprintf("%d秒", 1)), zap.String("剩余时间", fmt.Sprintf("%d秒", utils.Abs(diffInSeconds))))
			time.Sleep(1 * time.Second)
		} else if diffInMilliseconds >= -600*1000 {
			zap.L().Info(fmt.Sprintf("将在离秒杀时间剩余%d毫秒时开始发起请求", seckillIntervalTime), zap.String("休息时间", fmt.Sprintf("%d秒", 5)), zap.String("剩余时间", fmt.Sprintf("%d秒", utils.Abs(diffInSeconds))))
			time.Sleep(5 * time.Second)
		} else if diffInMilliseconds >= -3600*1000 {
			zap.L().Info(fmt.Sprintf("将在离秒杀时间剩余%d毫秒时开始发起请求", seckillIntervalTime), zap.String("休息时间", fmt.Sprintf("%d分", 5)), zap.String("剩余时间", fmt.Sprintf("%d秒", utils.Abs(diffInSeconds))))
			time.Sleep(5 * time.Minute)
		} else {
			zap.L().Info("为防止请求Token过期，请在秒杀活动开始前1小时及时更新", zap.String("休息时间", fmt.Sprintf("%d分", 30)), zap.String("剩余时间", fmt.Sprintf("%d秒", utils.Abs(diffInSeconds))))
			time.Sleep(30 * time.Minute)
		}
	}

	// 限流器
	limiter := rate.NewLimiter(rate.Every(250*time.Millisecond), 4)
	for {
		if limiter.Allow() {
			go engine.subscribeVaccine()
		}

		if carbon.Now().DiffInSeconds(seckillStartTime) < -10 {
			zap.L().Info("秒杀活动已结束，小助手自动退出")
			break
		}
	}

	return nil
}

// 获取城市的编码
func (engine *YMEngine) FetchCityCode() (map[string]interface{}, error) {
	cityCodes := make(map[string]interface{})

	headers := map[string]string{
		"User-Agent": resource.UserAgent,
	}

	// 省份
	data, err := xhttp.Do(resource.YMCityURL, http.MethodGet, headers, nil, nil)
	if err != nil {
		zap.L().Error("failed to do request", zap.Error(err))
		return nil, err
	}

	provinces, err := simplejson.NewJson(data)
	if err != nil {
		zap.L().Error("failed to unmarshal data", zap.String("provinces", string(data)), zap.Error(err))
		return nil, err
	}

	if provinces.Get("code").MustString() != resource.YMResponseOKCode || !provinces.Get("ok").MustBool() {
		zap.L().Error("unable to get provinces", zap.String("provinces", string(data)), zap.Error(err))
		return nil, err
	}

	for _, v := range provinces.Get("data").MustArray() {
		// 省
		province := v.(map[string]interface{})

		// 排除直辖市和特别行政区
		if value, ok := resource.Municipality[province["name"].(string)]; ok {
			if !value {
				continue
			}

			if _, exist := cityCodes["直辖市"]; !exist {
				cityCodes["直辖市"] = make([]interface{}, 0)
			}

			// 获取经纬度
			location := utils.GetLocation(province["name"].(string))

			cityCodes["直辖市"] = append(cityCodes["直辖市"].([]interface{}), map[string]interface{}{
				"name":     province["name"].(string),
				"value":    fmt.Sprintf("%v", province["value"]),
				"location": location,
			})
			continue
		} else if value, ok = resource.SpecialAdministrativeRegion[province["name"].(string)]; ok {
			if !value {
				continue
			}

			if _, exist := cityCodes["特别行政区"]; !exist {
				cityCodes["特别行政区"] = make([]interface{}, 0)
			}

			// 获取经纬度
			location := make(map[string]interface{})
			if province["name"].(string) == "香港" {
				location = map[string]interface{}{
					"lat": 22.320048,
					"lng": 114.173355,
				}
			}

			cityCodes["特别行政区"] = append(cityCodes["特别行政区"].([]interface{}), map[string]interface{}{
				"name":     province["name"].(string),
				"value":    fmt.Sprintf("%v01", province["value"]),
				"location": location,
			})
			continue
		}

		// 市
		queries := map[string]string{
			"parentCode": province["value"].(string),
		}
		data, err = xhttp.Do(resource.YMCityURL, http.MethodGet, headers, queries, nil)
		if err != nil {
			zap.L().Error("failed to do request", zap.Error(err))
			return nil, err
		}

		cities, err := simplejson.NewJson(data)
		if err != nil {
			zap.L().Error("failed to unmarshal city", zap.String("city", string(data)), zap.Error(err))
			return nil, err
		}

		for _, vv := range cities.Get("data").MustArray() {
			city := vv.(map[string]interface{})

			if _, exist := cityCodes[province["name"].(string)]; !exist {
				cityCodes[province["name"].(string)] = make([]interface{}, 0)
			}

			// 获取经纬度
			location := utils.GetLocation(city["name"].(string))
			if location == nil {
				// 若获取失败，则获取省份的经纬度
				location = utils.GetLocation(province["name"].(string))
			}

			cityCodes[province["name"].(string)] = append(cityCodes[province["name"].(string)].([]interface{}), map[string]interface{}{
				"name":     city["name"].(string),
				"value":    city["value"].(string),
				"location": location,
			})
		}
	}

	return cityCodes, nil
}

// 判断是否有秒杀信息
func (engine *YMEngine) hasSeckill(wg *sync.WaitGroup, channels chan<- []map[string]string, province string, cities []interface{}) {
	// 协程管理信号量减一
	defer wg.Done()

	headers := map[string]string{
		"User-Agent": resource.UserAgent,
	}

	result := make([]map[string]string, 0)
	for _, v := range cities {
		city := v.(map[string]interface{})

		if flag, ok := resource.Municipality[city["name"].(string)]; ok && flag {
			city["value"] = fmt.Sprintf("%v01", city["value"])
		}

		if configs.AllConfig.Basic.Debug {
			zap.L().Debug("当前探测的城市", zap.Any("province", province), zap.Any("city", city))
		}

		queries := map[string]string{
			"regionCode": city["value"].(string),
			"offset":     "0",
			"limit":      "10",
		}

		data, err := xhttp.Do(resource.YMHasSeckillURL, http.MethodGet, headers, queries, nil)
		if err != nil {
			zap.L().Error("failed to do request", zap.Any("city", city), zap.Error(err))
			return
		}

		dataJSON, err := simplejson.NewJson(data)
		if err != nil {
			zap.L().Error("failed to unmarshal data", zap.Any("city", city), zap.String("data", string(data)), zap.Error(err))
			return
		}

		if dataJSON.Get("code").MustString() != resource.YMResponseOKCode || !dataJSON.Get("ok").MustBool() {
			zap.L().Error("unable to get seckill info", zap.String("province", province), zap.Any("city", city), zap.Any("data", dataJSON.MustMap()), zap.Error(err))
			return
		}

		for _, vv := range dataJSON.Get("data").MustArray() {
			item := vv.(map[string]interface{})

			// 移除已过期的秒杀信息
			if carbon.ParseByFormat(item["startTime"].(string), carbon.DateTimeFormat).ToTimestamp() < carbon.Now().ToTimestamp() {
				continue
			}

			if vaccineName, ok := item["vaccineName"]; ok && strings.Contains(vaccineName.(string), "九价") {
				vaccine := map[string]string{
					"city":       city["name"].(string),             // 城市
					"seckill":    item["id"].(json.Number).String(), // 秒杀编号
					"vaccine":    vaccineName.(string),              // 疫苗名称
					"hospital":   item["name"].(string),             // 医院名称
					"start_time": item["startTime"].(string),        // 开始时间
					"source":     "约苗",
				}
				result = append(result, vaccine)
			} else if configs.AllConfig.Basic.Debug {
				zap.L().Debug("当前城市的秒杀信息", zap.Any("city", city), zap.String("vaccine", vaccineName.(string)))
				continue
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	channels <- result
}

// 订购疫苗
func (engine *YMEngine) subscribeVaccine() {
	headers := map[string]string{
		"User-Agent": resource.UserAgent,
		"tk":         configs.AllConfig.YM.Token,
	}

	query := map[string]string{
		"seckillId":    configs.AllConfig.YM.SeckillID,
		"linkmanId":    configs.AllConfig.YM.LinkmanID,
		"idCardNo":     configs.AllConfig.YM.LinkmanIDCard,
		"vaccineIndex": "1",
	}

	data, err := xhttp.Do(resource.YMSubscribeURL, http.MethodGet, headers, query, nil)
	if err != nil {
		zap.L().Error("failed to do request", zap.Error(err))
		return
	}

	dataJSON, err := simplejson.NewJson(data)
	if err != nil {
		zap.L().Error("failed to unmarshal data", zap.Error(err))
		return
	}

	zap.L().Info("发送请求成功", zap.Any("data", dataJSON.MustMap()))

	if dataJSON.Get("code").MustString() != resource.YMResponseOKCode {
		zap.L().Error("订购失败", zap.String("message", dataJSON.Get("msg").MustString()))
		if dataJSON.Get("msg").MustString() == "操作过于频繁,请稍后再试!" {
			time.Sleep(125 * time.Millisecond)
		} else if dataJSON.Get("msg").MustString() == "用户登录超时,请重新登入!" {
			time.Sleep(2 * time.Second)
		} else if dataJSON.Get("msg").MustString() == "很抱歉 没抢到" {
			time.Sleep(2 * time.Second)
		}
	}
}
