package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/internal/handler"
	"github.com/VennLe/charlotte/internal/middleware"
)

// Dependencies 路由依赖
type Dependencies struct {
	UserHandler   *handler.UserHandler
	HealthHandler *handler.HealthHandler
	RedisClient   *redis.Client
}

// NewRouter 创建路由
func NewRouter(deps *Dependencies) *gin.Engine {
	if config.Global.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// 全局中间件
	r.Use(middleware.ZapLogger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	
	// 使用新的限流中间件
	if deps.RedisClient != nil {
		r.Use(middleware.DefaultRateLimiter(deps.RedisClient))
	}

	// 健康检查 (公开)
	r.GET("/health", deps.HealthHandler.Check)
	r.GET("/ready", deps.HealthHandler.Check)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 认证相关 (公开)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", deps.UserHandler.Register)
			auth.POST("/login", deps.UserHandler.Login)
		}

		// 需要 JWT 认证
		authorized := v1.Group("")
		authorized.Use(middleware.JWTAuth())
		{
			// 用户管理
			users := authorized.Group("/users")
			{
				users.GET("", deps.UserHandler.GetUsers)
				users.GET("/:id", deps.UserHandler.GetUser)
				users.POST("", deps.UserHandler.CreateUser)
				users.PUT("/:id", deps.UserHandler.UpdateUser)
				users.DELETE("/:id", deps.UserHandler.DeleteUser)
			}

			// 当前用户
			authorized.GET("/profile", deps.UserHandler.GetProfile)
			authorized.PUT("/password", deps.UserHandler.ChangePassword)
		}
	}

	return r
}
