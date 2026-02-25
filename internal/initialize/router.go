package initialize

import (
	"github.com/gin-gonic/gin"

	"github.com/VennLe/charlotte/internal/handler"
	"github.com/VennLe/charlotte/internal/router"
	"github.com/VennLe/charlotte/internal/service"
)

// InitRouter 初始化路由（依赖注入模式）
func InitRouter() *gin.Engine {
	// 初始化服务层
	userService := service.NewUserService(DB)
	healthChecker := service.NewHealthChecker(DB, Redis, KafkaProducer)

	// 初始化处理器
	userHandler := handler.NewUserHandler(userService)
	healthHandler := handler.NewHealthHandler(healthChecker)

	// 组装依赖
	deps := &router.Dependencies{
		UserHandler:   userHandler,
		HealthHandler: healthHandler,
		RedisClient:   Redis,
	}

	return router.NewRouter(deps)
}
