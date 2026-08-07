// Package command Gateway服务application层命令，编排管理面操作。
package command

import (
	"go.uber.org/zap"
)

// AdminCommandHandler 管理面命令handler，编排配置热更/赛季管理等运营操作。
type AdminCommandHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewAdminCommandHandler 创建管理面命令handler实例。
func NewAdminCommandHandler(logger *zap.Logger) *AdminCommandHandler {
	return &AdminCommandHandler{logger: logger}
}
