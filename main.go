package main

import (
	"cupid/logic"
	"cupid/pkg/configs"
	"cupid/pkg/logger"
	"cupid/pkg/utils"
	"cupid/resource"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/go-agumon/table"
	"github.com/golang-module/carbon"

	"go.uber.org/zap"

	"github.com/urfave/cli/v2"
)

var (
	app *cli.App
)

// 初始化函数
func init() {
	app = cli.NewApp()
	// APP 的名称
	app.Name = "Cupid"
	// APP 的作者
	app.Authors = []*cli.Author{
		{Name: "wangyuxiao", Email: "xiao.950901@gmail.com"},
	}
	// APP 的版权
	app.Copyright = "©2021-2021 Meituan Corporation,All Rights Reserved"
	// APP 的版本
	app.Version = "0.0.1"
}

func main() {
	// 将时间戳设置成种子数
	rand.Seed(time.Now().UnixNano())

	// 注册服务
	app.Commands = []*cli.Command{
		{
			Name:  "sniff",
			Usage: "探测哪些城市有秒杀信息",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "conf",
					Aliases:  []string{`c`},
					Usage:    "指定配置文件",
					Value:    "",
					Required: true,
				},
			},
			Action: func(c *cli.Context) error {
				if err := SniffService(c.String("conf")); err != nil {
					return cli.Exit(err.Error(), 1)
				}
				return nil
			},
		},
		{
			Name:  "seckill",
			Usage: "订购疫苗",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "conf",
					Aliases:  []string{`c`},
					Usage:    "指定配置文件",
					Value:    "",
					Required: true,
				},
			},
			Action: func(c *cli.Context) error {
				if err := SeckillService(c.String("conf")); err != nil {
					return cli.Exit(err.Error(), 1)
				}
				return nil
			},
		},
	}

	// 运行服务
	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("Service failed to start, err: [%s]\n", err.Error())
		os.Exit(1)
	}
}

// 探测哪些城市有秒杀信息
func SniffService(configFile string) (err error) {
	// 解析配置文件
	if err = configs.ParseConfigFile(configFile); err != nil {
		return err
	}

	// 初始化日志对象
	if err = logger.Init("sniff"); err != nil {
		return err
	}
	// 延迟注册：将缓存区的日志追加到日志文件中
	defer logger.Sync()

	// 创建表格
	seckillTable, _ := table.Create("渠道", "城市", "医院", "疫苗", "秒杀时间", "秒杀编号")

	// 生成城市编码文件
	var allCityCodes map[string]interface{}
	if !utils.FileExist(resource.CityCodeFile) {
		allCityCodes, err = logic.GetYMEngine().FetchCityCode()
		if err != nil {
			zap.L().Error("unable to get city code from api", zap.Error(err))
			return err
		}

		if err = utils.WriteJSONToFile(resource.CityCodeFile, allCityCodes); err != nil {
			zap.L().Error("unable to write city code to file", zap.Error(err))
			return err
		}
	}

	// 嗅探秒杀信息 - 约苗
	ymResult, err := logic.GetYMEngine().Sniff()
	if err != nil {
		zap.L().Error("无法获取约苗当前哪些城市有秒杀信息", zap.Error(err))
		return err
	}

	for _, v := range ymResult {
		// 移除已过期的秒杀信息
		if carbon.ParseByFormat(v["start_time"], carbon.DateTimeFormat).ToTimestamp() < carbon.Now().ToTimestamp() {
			continue
		}

		seckillRow := map[string]string{
			"渠道":   v["source"],
			"城市":   v["city"],
			"医院":   v["hospital"],
			"疫苗":   v["vaccine"],
			"秒杀时间": v["start_time"],
			"秒杀编号": v["seckill"],
		}
		err = seckillTable.AddRow(seckillRow)
		if err != nil {
			zap.L().Error("add row to table failed", zap.Error(err))
			continue
		}
	}

	// 嗅探秒杀信息 - 知苗易约
	zmyyResult, err := logic.GetZMYYEngine().Sniff()
	if err != nil {
		zap.L().Error("无法获取知苗易约当前哪些城市有秒杀信息", zap.Error(err))
		return err
	}

	for _, v := range zmyyResult {
		// 移除无法预约的秒杀信息
		if v["start_time"] == "暂无" {
			continue
		} else {
			dateSlice := strings.Split("12-03 17:05 至 12-03 17:10", " 至 ")
			if len(dateSlice) == 2 {
				v["start_time"] = fmt.Sprintf("%v-%s:00", carbon.Now().Year(), dateSlice[0])

				// 移除已过期的秒杀信息
				if carbon.ParseByFormat(v["start_time"], carbon.DateTimeFormat).ToTimestamp() < carbon.Now().ToTimestamp() {
					continue
				}
			}
		}

		seckillRow := map[string]string{
			"渠道":   v["source"],
			"城市":   v["city"],
			"医院":   v["hospital"],
			"疫苗":   v["vaccine"],
			"秒杀时间": v["start_time"],
			"秒杀编号": v["seckill"],
		}
		err = seckillTable.AddRow(seckillRow)
		if err != nil {
			zap.L().Error("add row to table failed", zap.Error(err))
			continue
		}
	}

	seckillTable.Print()

	return nil
}

// 秒杀疫苗
func SeckillService(configFile string) (err error) {
	// 解析配置文件
	if err = configs.ParseConfigFile(configFile); err != nil {
		return err
	}

	// 初始化日志对象
	if err = logger.Init("seckill"); err != nil {
		return err
	}
	// 延迟注册：将缓存区的日志追加到日志文件中
	defer logger.Sync()

	// 秒杀疫苗 - 约苗
	if err = logic.GetYMEngine().SecKill(); err != nil {
		zap.L().Error("很抱歉，疫苗订购失败", zap.Error(err))
		return err
	}

	return nil
}
