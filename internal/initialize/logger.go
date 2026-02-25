package initialize

import (
	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/pkg/logger"
)

func InitLogger() {
	logConfig := &config.Global.Log
	if logConfig.Level == "" {
		logConfig.Level = "info"
	}
	if logConfig.OutputPath == "" {
		logConfig.OutputPath = "./logs"
	}
	if logConfig.MaxSize == 0 {
		logConfig.MaxSize = 100
	}
	if logConfig.MaxBackups == 0 {
		logConfig.MaxBackups = 30
	}
	if logConfig.MaxAge == 0 {
		logConfig.MaxAge = 7
	}

	logger.Init(logConfig)
}
