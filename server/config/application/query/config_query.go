// Package query Config服务application层读侧查询，CQRS读模型查询handler。
//
// query handler注入domain层Repository接口（规范3 DDD），不直接依赖infrastructure实现。
package query

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/config/domain"
)

// ConfigVersionQueryHandler 配置版本查询handler，实现CQRS读侧查询。
type ConfigVersionQueryHandler struct {
	versionStorage domain.VersionStorage // 版本存储接口，读侧查询
	logger         *zap.Logger           // 结构化日志器（规范7）
}

// NewConfigVersionQueryHandler 创建配置版本查询handler实例。
// versionStorage为domain层接口，infrastructure实现由cmd/main.go注入。
func NewConfigVersionQueryHandler(versionStorage domain.VersionStorage, logger *zap.Logger) *ConfigVersionQueryHandler {
	return &ConfigVersionQueryHandler{
		versionStorage: versionStorage,
		logger:         logger,
	}
}

// VersionHistoryQuery 版本历史查询参数。
type VersionHistoryQuery struct {
	ConfigType int // 配置类型
	Limit      int // 返回数量上限
}

// Handle 查询配置版本历史，从读模型（version store）读取。
func (h *ConfigVersionQueryHandler) Handle(ctx context.Context, q VersionHistoryQuery) ([]domain.VersionInfo, error) {
	versions, err := h.versionStorage.FindVersions(ctx, q.ConfigType, q.Limit)
	if err != nil {
		h.logger.Error("查询配置版本历史失败",
			zap.Int("config_type", q.ConfigType),
			zap.Int("limit", q.Limit),
			zap.Error(err),
		)
		return nil, fmt.Errorf("查询配置版本历史失败: %w", err)
	}

	h.logger.Debug("查询配置版本历史完成",
		zap.Int("config_type", q.ConfigType),
		zap.Int("result_count", len(versions)),
	)
	return versions, nil
}
