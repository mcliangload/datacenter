package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger zerolog.Logger
var HTTPLogger zerolog.Logger

func Init(level, httpLogFile string, maxSize, maxBackups, maxAge int) {
	httpLogWriter := &lumberjack.Logger{
		Filename:   httpLogFile,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	}

	HTTPLogger = zerolog.New(httpLogWriter).With().
		Timestamp().
		Caller().
		Logger()

	appLogWriter := &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	}

	multiWriter := zerolog.MultiLevelWriter(os.Stdout, appLogWriter)
	Logger = zerolog.New(multiWriter).With().
		Timestamp().
		Caller().
		Logger()

	zerolog.TimeFieldFormat = time.RFC3339

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

func Debug(msg string, args ...interface{}) {
	Logger.Debug().Msgf(msg, args...)
}

func Info(msg string, args ...interface{}) {
	Logger.Info().Msgf(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	Logger.Warn().Msgf(msg, args...)
}

func Error(msg string, args ...interface{}) {
	Logger.Error().Msgf(msg, args...)
}

func DebugJSON(data map[string]interface{}) {
	Logger.Debug().Fields(data).Msg("")
}

func InfoJSON(data map[string]interface{}) {
	Logger.Info().Fields(data).Msg("")
}

func WarnJSON(data map[string]interface{}) {
	Logger.Warn().Fields(data).Msg("")
}

func ErrorJSON(data map[string]interface{}) {
	Logger.Error().Fields(data).Msg("")
}
