package middleware

import (
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/pkg/logger"
	"github.com/VennLe/charlotte/pkg/utils"
)

// Recovery 自定义错误恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否为客户端断开连接
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest := c.Request.Method + " " + c.Request.URL.Path
				logger.Error("HTTP 处理 panic",
					zap.Any("error", err),
					zap.String("request", httpRequest),
					zap.String("stack", string(debug.Stack())))

				if brokenPipe {
					c.Error(err.(error))
					c.Abort()
					return
				}

				utils.Error(c, http.StatusInternalServerError, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}
