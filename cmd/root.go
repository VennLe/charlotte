package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/pkg/logger"
)

var (
	Version   = "dev"
	BuildTime = "unknown"

	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "charlotte",
		Short: "Charlotte API 服务",
		Long: `基于 Gin + GORM + Redis + Kafka + Nacos 的高可用 API 服务
支持配置热更新、分布式追踪、结构化日志`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "配置文件路径 (默认: ./configs/config.yaml)")

	// 设置 GOMAXPROCS
	maxprocs.Set(maxprocs.Logger(func(format string, v ...interface{}) {
		logger.Debugf(format, v...)
	}))
}

func initConfig() {
	config.LoadConfig(cfgFile)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
