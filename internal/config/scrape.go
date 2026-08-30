package config

// ScrapeConfig 刮削子系统配置
type ScrapeConfig struct {
	WorkerCount      int `mapstructure:"worker_count"`       // Worker 并发数
	TimeoutSeconds   int `mapstructure:"timeout_seconds"`    // 单任务执行超时（秒），python 脚本可能长时间运行
	PollIntervalMs   int `mapstructure:"poll_interval_ms"`   // 空闲轮询间隔（毫秒）
	OutputLimitBytes int `mapstructure:"output_limit_bytes"` // 脚本输出大小上限（字节）
	ReclaimSeconds   int `mapstructure:"reclaim_seconds"`    // 僵死任务回收阈值（秒）
}
