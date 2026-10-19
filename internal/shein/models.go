package shein

// AmazonMonitorData Amazon监控数据结构（从monitor_helper.go移动过来）
type AmazonMonitorData struct {
	ASIN          string  `json:"asin"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	InStock       bool    `json:"in_stock"`
	Stock         int     `json:"stock"`
	LastCheckTime int64   `json:"last_check_time"` // Unix时间戳（秒）
}
