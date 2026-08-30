package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	global *zap.Logger
	atom   = zap.NewAtomicLevel()
)

// Init 初始化全局日志器。
// level: debug/info/warn/error；output: stdout 或日志文件路径。
func Init(level, output string) error {
	atom.SetLevel(parseLevel(level))

	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		writer(output),
		atom,
	)
	global = zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(global)
	return nil
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func writer(output string) zapcore.WriteSyncer {
	if output == "" || output == "stdout" {
		return zapcore.AddSync(os.Stdout)
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		// 打开日志文件失败时退回 stdout，避免服务无法启动
		return zapcore.AddSync(os.Stdout)
	}
	return zapcore.AddSync(f)
}

// L 返回全局日志器（未初始化时返回 Nop，避免空指针）
func L() *zap.Logger {
	if global == nil {
		return zap.NewNop()
	}
	return global
}

// Sync 刷新缓冲日志
func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}
