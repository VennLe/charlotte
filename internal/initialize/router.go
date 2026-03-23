package initialize

import (
	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"

	"github.com/VennLe/charlotte/internal/dao"
	"github.com/VennLe/charlotte/internal/handler"
	"github.com/VennLe/charlotte/internal/middleware"
	"github.com/VennLe/charlotte/internal/router"
	"github.com/VennLe/charlotte/internal/service"
)

// InitRouter 初始化路由（依赖注入模式）
func InitRouter() *gin.Engine {
	// 初始化数据访问层
	userDAO := dao.NewUserDAO(DB)
	permissionDAO := dao.NewUnifiedPermissionDAO(DB)

	// 初始化服务层
	userService := service.NewUserService(DB)
	permissionService := service.NewSimplifiedPermissionService(userDAO, permissionDAO)

	// 初始化健康检查器，处理 Kafka 可能为 nil 的情况
	var kafkaProducer sarama.SyncProducer
	if KafkaProducer != nil {
		kafkaProducer = *KafkaProducer
	}
	healthChecker := service.NewHealthChecker(DB, Redis, kafkaProducer)

	// 初始化服务层
	fileService := service.NewFileService()
	importExportService := service.NewImportExportService(fileService)

	// 初始化处理器
	userHandler := handler.NewUserHandler(userService)
	healthHandler := handler.NewHealthHandler(healthChecker)
	importExportHandler := handler.NewImportExportHandler(importExportService, fileService)

	// 初始化权限中间件
	permissionMiddleware := middleware.NewSimplifiedPermissionMiddleware(permissionService)

	// 组装依赖
	deps := &router.Dependencies{
		UserHandler:          userHandler,
		HealthHandler:        healthHandler,
		ImportExportHandler:  importExportHandler,
		RedisClient:          Redis, // 如果Redis初始化失败，这里会是nil
		PermissionMiddleware:  permissionMiddleware,
	}

	return router.NewRouter(deps)
}
