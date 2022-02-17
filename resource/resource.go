package resource

// 公共
const (
	// 用户代理
	UserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 11_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E302"

	// 城市编码文件
	CityCodeFile = "./city.json"
)

// 约苗
const (
	// 城市地址
	YMCityURL = "https://wx.healthych.com/base/region/childRegions.do"
	// 是否有秒杀信息
	YMHasSeckillURL = "https://miaomiao.scmttec.com/seckill/seckill/list.do"
	// 当前时间戳（毫秒）
	YMTimestampURL = "https://miaomiao.scmttec.com/seckill/seckill/now2.do"
	// 订购地址
	YMSubscribeURL = "https://miaomiao.scmttec.com/seckill/seckill/subscribe.do"

	// 正确的响应状态码
	YMResponseOKCode = "0000"
)

// 知苗易约
const (
	// 医院列表
	ZMYYRootURL = "https://cloud.cn2030.com/sc/wx/HandlerSubscribe.ashx"

	// 固定请求头
	ZMYYReferer = "https://servicewechat.com/wx2c7f0f3c30d99445/72/page-frame.html"
)

// 直辖市
var Municipality = map[string]bool{
	"北京市": true,
	"天津市": true,
	"上海市": true,
	"重庆市": true,
}

// 特别行政区
var SpecialAdministrativeRegion = map[string]bool{
	"香港": false,
}
