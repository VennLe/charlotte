package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	log   *zap.Logger
	sugar *zap.SugaredLogger
)

type Config struct {
	Level      string `mapstructure:"level" json:"level" yaml:"level"`
	Format     string `mapstructure:"format" json:"format" yaml:"format"`
	OutputPath string `mapstructure:"output_path" json:"output_path" yaml:"output_path"`
	MaxSize    int    `mapstructure:"max_size" json:"max_size" yaml:"max_size"`          // MB
	MaxBackups int    `mapstructure:"max_backups" json:"max_backups" yaml:"max_backups"` // 保留旧文件个数
	MaxAge     int    `mapstructure:"max_age" json:"max_age" yaml:"max_age"`             // 保留天数
	Compress   bool   `mapstructure:"compress" json:"compress" yaml:"compress"`          // 是否压缩
	Console    bool   `mapstructure:"console" json:"console" yaml:"console"`             // 是否输出到控制台
}

func Init(cfg *Config) *zap.Logger {
	// 解析日志级别
	level := getLogLevel(cfg.Level)

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// JSON 或 Console 格式
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 日志切割
	writeSyncer := getLogWriter(cfg)

	// 核心配置
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 添加调用者信息
	log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	sugar = log.Sugar()

	return log
}

func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func getLogWriter(cfg *Config) zapcore.WriteSyncer {
	// 确保目录存在
	if err := os.MkdirAll(cfg.OutputPath, 0755); err != nil {
		panic(err)
	}

	//  lumberjack 实现日志切割
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.OutputPath, "app.log"),
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// 同时输出到文件和控制台
	if cfg.Console {
		return zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(lumberJackLogger),
			zapcore.AddSync(os.Stdout),
		)
	}
	return zapcore.AddSync(lumberJackLogger)
}

// 全局方法
func Debug(msg string, fields ...zap.Field)       { log.Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)        { log.Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)        { log.Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field)       { log.Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field)       { log.Fatal(msg, fields...) }
func Debugf(template string, args ...interface{}) { sugar.Debugf(template, args...) }
func Infof(template string, args ...interface{})  { sugar.Infof(template, args...) }
func Warnf(template string, args ...interface{})  { sugar.Warnf(template, args...) }
func Errorf(template string, args ...interface{}) { sugar.Errorf(template, args...) }
func Fatalf(template string, args ...interface{}) { sugar.Fatalf(template, args...) }

func Sync() error            { return log.Sync() }
func GetLogger() *zap.Logger { return log }
