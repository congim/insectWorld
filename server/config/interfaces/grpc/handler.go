// Package grpc Config服务接口层gRPC handler，实现ConfigServiceServer接口。
package grpc

import (
	"context"

	"go.uber.org/zap"

	configpb "insectworld/server/shared/proto/config"
)

// ConfigHandler Config服务gRPC handler，实现ConfigServiceServer接口。
type ConfigHandler struct {
	configpb.UnimplementedConfigServiceServer             // 嵌入未实现基类，保证前向兼容
	logger                                    *zap.Logger // 结构化日志器（规范7）
}

// NewConfigHandler 创建Config服务gRPC handler实例。
func NewConfigHandler(logger *zap.Logger) *ConfigHandler {
	return &ConfigHandler{logger: logger}
}

// SubmitConfigPack 提交配置包。
func (h *ConfigHandler) SubmitConfigPack(ctx context.Context, req *configpb.SubmitConfigRequest) (*configpb.SubmitConfigResponse, error) {
	h.logger.Info("提交配置包")
	return &configpb.SubmitConfigResponse{}, nil
}

// RollbackConfig 回滚配置。
func (h *ConfigHandler) RollbackConfig(ctx context.Context, req *configpb.RollbackRequest) (*configpb.RollbackResponse, error) {
	h.logger.Info("回滚配置")
	return &configpb.RollbackResponse{}, nil
}

// GetConfigVersionHistory 查询配置版本历史。
func (h *ConfigHandler) GetConfigVersionHistory(ctx context.Context, req *configpb.VersionHistoryRequest) (*configpb.VersionHistoryResponse, error) {
	h.logger.Info("查询配置版本历史")
	return &configpb.VersionHistoryResponse{}, nil
}
