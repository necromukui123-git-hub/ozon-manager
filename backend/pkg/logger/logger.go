package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

// Init 初始化全局日志记录器
func Init() {
	// 判断 logs 目录是否存在，不存在则创建
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", 0766)
	}

	writeSyncer := getLogWriter()
	encoder := getEncoder()
	
	// 同时输出到控制台和文件
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
	)

	// 添加调用者信息和堆栈跟踪(针对 Error 及以上级别)
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig) // 可以换成 NewJSONEncoder 以 JSON 格式输出
}

func getLogWriter() zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "logs/ozon-manager.log",
		MaxSize:    10,    // megabytes (每个日志文件最大 10MB)
		MaxBackups: 30,    // 最多保留 30 个备份
		MaxAge:     30,    // days (最多保留 30 天)
		Compress:   false, // 是否压缩(gzip)
	}
	return zapcore.AddSync(lumberJackLogger)
}

// Sync 刷新任何缓冲的日志条目
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
