package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/VennLe/charlotte/internal/service"
	"github.com/VennLe/charlotte/pkg/utils"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	checker *service.HealthChecker
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(checker *service.HealthChecker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

// Check 健康检查接口
func (h *HealthHandler) Check(c *gin.Context) {
	status := h.checker.Check(c.Request.Context())
	utils.Success(c, status)
}
