package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// envPrefix 环境变量前缀：DATACENTER_xxx
const envPrefix = "DATACENTER"

// Config 应用总配置
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Log       LogConfig       `mapstructure:"log"`
	Data      DataConfig      `mapstructure:"data"`
	Scrape    ScrapeConfig    `mapstructure:"scrape"`
	Bootstrap BootstrapConfig `mapstructure:"bootstrap"`
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug / release
}

// DatabaseConfig MongoDB 配置
type DatabaseConfig struct {
	URI     string `mapstructure:"uri"`
	Name    string `mapstructure:"name"`
	Timeout int    `mapstructure:"timeout"` // 连接超时（秒）
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
	Issuer      string `mapstructure:"issuer"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug / info / warn / error
	Output string `mapstructure:"output"` // stdout 或文件路径
}

// DataConfig NFS 数据目录配置
type DataConfig struct {
	RootDir string `mapstructure:"root_dir"` // 数据根目录，防路径逃逸基准
}

// BootstrapConfig 首次启动种子数据配置
type BootstrapConfig struct {
	AdminUsername string `mapstructure:"admin_username"`
	AdminPassword string `mapstructure:"admin_password"`
}

// Load 从默认值 + 配置文件 + 环境变量加载配置。
// 优先级：环境变量 > 配置文件 > 默认值。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 默认值
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.uri", "mongodb://localhost:27017")
	v.SetDefault("database.name", "datacenter")
	v.SetDefault("database.timeout", 10)
	v.SetDefault("jwt.secret", "datacenter-dev-secret")
	v.SetDefault("jwt.expire_hours", 12) // 安全增强 P1-8：默认 12h（原 72h）
	v.SetDefault("jwt.issuer", "datacenter")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.output", "stdout")
	v.SetDefault("data.root_dir", "/nfs/data")
	v.SetDefault("scrape.worker_count", 8)
	v.SetDefault("scrape.timeout_seconds", 1800)
	v.SetDefault("scrape.poll_interval_ms", 2000)
	v.SetDefault("scrape.output_limit_bytes", 1048576)
	v.SetDefault("scrape.reclaim_seconds", 3600)

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	return &cfg, nil
}
