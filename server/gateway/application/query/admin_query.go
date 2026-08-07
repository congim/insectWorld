// Package query Gateway服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"go.uber.org/zap"
)

// AdminQueryHandler 管理面查询handler。
type AdminQueryHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewAdminQueryHandler 创建管理面查询handler实例。
func NewAdminQueryHandler(logger *zap.Logger) *AdminQueryHandler {
	return &AdminQueryHandler{logger: logger}
}
