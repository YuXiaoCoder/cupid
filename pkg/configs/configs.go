package configs

import (
	"io/fs"
	"log"
	"os"

	"go.uber.org/zap"

	"github.com/fsnotify/fsnotify"

	"github.com/spf13/viper"
)

// 全局配置的对象
var AllConfig ServerConfig

// 全局配置的结构体
type ServerConfig struct {
	Basic  BasicConfig  `mapstructure:"basic"`  // 基础配置
	Logger LoggerConfig `mapstructure:"logger"` // 日志配置
	Sniff  SniffConfig  `mapstructure:"sniff"`  // 嗅探
	YM     YMConfig     `mapstructure:"ym"`     // 约苗
	ZMYY   ZMYYConfig   `mapstructure:"zmyy"`   // 知苗易约
}

// 基础配置
type BasicConfig struct {
	Debug bool `mapstructure:"debug"` // 调试模式
}

// 日志配置
type LoggerConfig struct {
	Level         string `mapstructure:"level"`          // 日志级别
	Directory     string `mapstructure:"directory"`      // 日志目录
	RotationTime  int    `mapstructure:"rotation_time"`  // 日志轮换时间间隔，单位为小时
	RotationCount uint   `mapstructure:"rotation_count"` // 日志轮换文件保留个数
}

// 嗅探配置
type SniffConfig struct {
	Regions []string `mapstructure:"regions"` // 区域，同时支持省粒度和市粒度
}

// 约苗配置
type YMConfig struct {
	Token         string `mapstructure:"token"`           // Token
	SeckillID     string `mapstructure:"seckill_id"`      // 秒杀编号
	LinkmanID     string `mapstructure:"linkman_id"`      // 接种人编号
	LinkmanIDCard string `mapstructure:"linkman_id_card"` // 接种人身份证号
}

// 知苗易约配置
type ZMYYConfig struct {
	Cookie string `mapstructure:"cookie"` // Cookie
}

func ParseConfigFile(configFile string) error {
	// 指定配置文件路径
	viper.SetConfigFile(configFile)
	// 指定配置文件格式
	viper.SetConfigType("yaml")

	// 解析配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("the configuration file does not exist, err: %s\n", err.Error())
		} else if _, flag := err.(*fs.PathError); flag {
			log.Fatalf("the configuration dir does not exist, err: %s\n", err.Error())
		} else {
			log.Fatalf("could not load configuration file, err: %s\n", err.Error())
		}
		return err
	}

	// 反序列化
	if err := viper.Unmarshal(&AllConfig); err != nil {
		log.Fatalf("Unable to unmarshal config, err: %s\n", err.Error())
		return err
	}

	// 动态加载配置文件：监视配置文件
	viper.WatchConfig()
	// 配置文件发生更改
	viper.OnConfigChange(func(event fsnotify.Event) {
		if event.Op == fsnotify.Write {
			// 反序列化
			if err := viper.Unmarshal(&AllConfig); err != nil {
				zap.L().Error("unable to unmarshal config", zap.Error(err))
				os.Exit(1)
			}
			zap.L().Info("the configuration file has changed")
		}
	})

	return nil
}
