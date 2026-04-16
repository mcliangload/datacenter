package logger

import (
	"os"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 全局日志实例
var Logger zerolog.Logger

// Init 初始化日志系统
func Init(level, filePath string, maxSize, maxBackups, maxAge int) {
	// 配置 lumberjack 日志轮转
	logWriter := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    maxSize,    // 日志文件最大大小（MB）
		MaxBackups: maxBackups, // 最多保留的日志文件数
		MaxAge:     maxAge,     // 日志文件最大保留天数
		Compress:   true,       // 压缩日志文件
	}

	// 配置 zerolog
	multiWriter := zerolog.MultiLevelWriter(os.Stdout, logWriter)
	Logger = zerolog.New(multiWriter).With().
		Timestamp().
		Caller().
		Logger()

	// 设置日志级别
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// Debug 记录调试级别日志
func Debug(msg string, args ...interface{}) {
	Logger.Debug().Msgf(msg, args...)
}

// Info 记录信息级别日志
func Info(msg string, args ...interface{}) {
	Logger.Info().Msgf(msg, args...)
}

// Warn 记录警告级别日志
func Warn(msg string, args ...interface{}) {
	Logger.Warn().Msgf(msg, args...)
}

// Error 记录错误级别日志
func Error(msg string, args ...interface{}) {
	Logger.Error().Msgf(msg, args...)
}

// DebugJSON 记录调试级别的JSON日志
func DebugJSON(data map[string]interface{}) {
	Logger.Debug().Fields(data).Msg("")
}

// InfoJSON 记录信息级别的JSON日志
func InfoJSON(data map[string]interface{}) {
	Logger.Info().Fields(data).Msg("")
}

// WarnJSON 记录警告级别的JSON日志
func WarnJSON(data map[string]interface{}) {
	Logger.Warn().Fields(data).Msg("")
}

// ErrorJSON 记录错误级别的JSON日志
func ErrorJSON(data map[string]interface{}) {
	Logger.Error().Fields(data).Msg("")
}
