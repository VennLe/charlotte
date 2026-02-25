package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/internal/initialize"
	"github.com/VennLe/charlotte/pkg/logger"
)

func init() {
	// 添加所有子命令
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configEnvCmd)
	rootCmd.AddCommand(versionCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 API 服务器",
	Long:  "初始化所有组件并启动 HTTP 服务",
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
	Aliases: []string{"run", "server"},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "执行数据库迁移",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initialize.Migrate(); err != nil {
			fmt.Printf("❌ 迁移失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ 数据库迁移完成")
	},
}

var configShowCmd = &cobra.Command{
	Use:   "config show",
	Short: "显示当前配置",
	Run: func(cmd *cobra.Command, args []string) {
		config.Show()
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "config validate",
	Short: "验证配置完整性",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.Validate(); err != nil {
			fmt.Printf("❌ 配置验证失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ 配置验证通过")
		fmt.Println(config.GetConfigSummary())
	},
}

var configEnvCmd = &cobra.Command{
	Use:   "config env",
	Short: "显示环境变量映射",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("环境变量映射关系:")
		fmt.Println("CHARLOTTE_SERVER_NAME     -> server.name")
		fmt.Println("CHARLOTTE_SERVER_PORT     -> server.port")
		fmt.Println("CHARLOTTE_DATABASE_HOST   -> database.host")
		fmt.Println("CHARLOTTE_DATABASE_USER   -> database.user")
		fmt.Println("CHARLOTTE_DATABASE_PASSWORD -> database.password")
		fmt.Println("CHARLOTTE_REDIS_HOST      -> redis.host")
		fmt.Println("CHARLOTTE_REDIS_PASSWORD  -> redis.password")
		fmt.Println("CHARLOTTE_JWT_SECRET      -> jwt.secret")
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("export CHARLOTTE_DATABASE_HOST=postgres-primary")
		fmt.Println("export CHARLOTTE_DATABASE_PASSWORD=your_password")
		fmt.Println("export CHARLOTTE_JWT_SECRET=your_secret_key")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		PrintVersion()
	},
}

func runServer() {
	// 1. 初始化日志
	initialize.InitLogger()
	defer logger.Sync()

	logger.Info("启动 Charlotte API",
		zap.String("version", Version),
		zap.String("build_time", BuildTime))

	// 2. 验证配置
	if err := config.Validate(); err != nil {
		logger.Fatal("配置验证失败", zap.Error(err))
	}
	logger.Info(config.GetConfigSummary())

	// 3. 初始化组件
	if err := initialize.InitRedis(); err != nil {
		logger.Fatal("Redis 初始化失败", zap.Error(err))
	}

	if err := initialize.InitKafka(); err != nil {
		logger.Fatal("Kafka 初始化失败", zap.Error(err))
	}

	// 数据库必须成功连接，否则无法运行
	if err := initialize.InitGorm(); err != nil {
		logger.Fatal("数据库初始化失败", zap.Error(err))
	}

	// 4. 初始化路由
	router := initialize.InitRouter()

	// 5. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		port := config.Global.Server.Port
		if port == "" {
			port = "8080"
		}

		logger.Info("HTTP 服务启动", zap.String("port", port))
		if err := router.Run(":" + port); err != nil {
			logger.Fatal("服务启动失败", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("正在关闭服务...")

	// 关闭 Kafka 连接
	initialize.CloseKafka()

	logger.Info("服务已停止")
}
