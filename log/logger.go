package log

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogSugar 全局Sugar日志实例，支持printf风格格式化
var Log *zap.SugaredLogger

// Config 日志配置
type Config struct {
	EnableConsole bool   // 是否启用控制台输出
	EnableFile    bool   // 是否启用文件输出
	ConsoleLevel  string // 控制台日志级别
	FileLevel     string // 文件日志级别
	FilePath      string // 文件输出路径
	MaxSize       int    // 单个日志文件大小(MB)
	MaxBackups    int    // 最大备份数量
	MaxAge        int    // 最大保存天数
	Compress      bool   // 是否压缩
}

// NewLogger 创建新的日志实例
func InitLogger(cfg Config) error {
	// 解析控制台日志级别
	consoleLevel := zap.InfoLevel
	if err := consoleLevel.Set(cfg.ConsoleLevel); err != nil {
		return err
	}

	fileLevel := zap.DebugLevel
	if err := fileLevel.Set(cfg.FileLevel); err != nil {
		return err
	}

	// 编码器配置
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02T15:04:05.000"))
	}

	// 获取当前进程ID并自定义级别编码器
	pid := os.Getpid()
	encoderCfg.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(fmt.Sprintf(" %d\t%s", pid, level.String()))
	}

	// 创建核心数组
	var cores []zapcore.Core

	// 添加控制台输出
	if cfg.EnableConsole {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), consoleLevel)
		cores = append(cores, consoleCore)
	}

	// 添加文件输出
	if cfg.EnableFile {
		fileEncoder := zapcore.NewConsoleEncoder(encoderCfg)
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,   // 日志文件路径
			MaxSize:    cfg.MaxSize,    // 单个日志文件大小(MB)
			MaxBackups: cfg.MaxBackups, // 保留旧文件数
			MaxAge:     cfg.MaxAge,     // 保留天数
			Compress:   cfg.Compress,   // 是否压缩归档
		}
		fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(fileWriter), fileLevel)
		cores = append(cores, fileCore)
	}

	// 创建日志器
	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1))
	Log = logger.Sugar()

	return nil
}

// Debug logs the provided arguments at [DebugLevel].
// Spaces are added between arguments when neither is a string.
func Debug(args ...interface{}) {
	Log.Debug(args...)
}

// Info logs the provided arguments at [InfoLevel].
// Spaces are added between arguments when neither is a string.
func Info(args ...interface{}) {
	Log.Info(args...)
}

// Warn logs the provided arguments at [WarnLevel].
// Spaces are added between arguments when neither is a string.
func Warn(args ...interface{}) {
	Log.Warn(args...)
}

// Error logs the provided arguments at [ErrorLevel].
// Spaces are added between arguments when neither is a string.
func Error(args ...interface{}) {
	Log.Error(args...)
}

// Panic constructs a message with the provided arguments and panics.
// Spaces are added between arguments when neither is a string.
func Panic(args ...interface{}) {
	Log.Panic(args...)
}

// Fatal constructs a message with the provided arguments and calls os.Exit.
// Spaces are added between arguments when neither is a string.
func Fatal(args ...interface{}) {
	Log.Fatal(args...)
}

// Debugf formats the message according to the format specifier
// and logs it at [DebugLevel].
func Debugf(template string, args ...interface{}) {
	Log.Debugf(template, args...)
}

// Infof formats the message according to the format specifier
// and logs it at [InfoLevel].
func Infof(template string, args ...interface{}) {
	Log.Infof(template, args...)
}

// Warnf formats the message according to the format specifier
// and logs it at [WarnLevel].
func Warnf(template string, args ...interface{}) {
	Log.Warnf(template, args...)
}

// Errorf formats the message according to the format specifier
// and logs it at [ErrorLevel].
func Errorf(template string, args ...interface{}) {
	Log.Errorf(template, args...)
}

// Panicf formats the message according to the format specifier
// and panics.
func Panicf(template string, args ...interface{}) {
	Log.Panicf(template, args...)
}

// Fatalf formats the message according to the format specifier
// and calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	Log.Fatalf(template, args...)
}
