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
	UserHandler         *handler.UserHandler
	HealthHandler       *handler.HealthHandler
	ImportExportHandler *handler.ImportExportHandler
	RedisClient         *redis.Client
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

			// 导入导出功能
			importExport := authorized.Group("/import-export")
			{
				// 获取支持的数据类型
				importExport.GET("/supported-types", deps.ImportExportHandler.GetSupportedDataTypes)
				
				// 数据导入
				importExport.POST("/import", deps.ImportExportHandler.ImportData)
				
				// 数据导出
				importExport.POST("/export", deps.ImportExportHandler.ExportData)
				
				// 获取导入模板
				importExport.GET("/template", deps.ImportExportHandler.GetImportTemplate)
			}

			// 文件管理功能
			files := authorized.Group("/files")
			{
				// 文件上传
				files.POST("/upload", deps.ImportExportHandler.UploadFile)
				
				// 文件列表
				files.GET("", deps.ImportExportHandler.ListFiles)
				
				// 文件信息
				files.GET("/:file_id/info", deps.ImportExportHandler.GetFileInfo)
				
				// 文件删除
				files.DELETE("/:file_id", deps.ImportExportHandler.DeleteFile)
			}
		}

		// 文件下载 (公开)
		r.GET("/files/download/:file_id", deps.ImportExportHandler.DownloadFile)
	}

	return r
}
