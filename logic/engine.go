package logic

type Engine interface {
	// 探测哪些城市有秒杀信息
	Sniff() ([]map[string]string, error)

	// 秒杀疫苗
	SecKill() error
}
