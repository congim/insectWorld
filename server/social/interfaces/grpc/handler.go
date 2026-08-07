// Package grpc Social服务接口层gRPC handler，实现SocialServiceServer接口。
package grpc

import (
	"context"

	"go.uber.org/zap"

	socialpb "insectworld/server/shared/proto/social"
)

// SocialHandler Social服务gRPC handler，实现SocialServiceServer接口。
type SocialHandler struct {
	socialpb.UnimplementedSocialServiceServer             // 嵌入未实现基类，保证前向兼容
	logger                                    *zap.Logger // 结构化日志器（规范7）
}

// NewSocialHandler 创建Social服务gRPC handler实例。
func NewSocialHandler(logger *zap.Logger) *SocialHandler {
	return &SocialHandler{logger: logger}
}

// GetPlayerInfo 查询玩家信息。
func (h *SocialHandler) GetPlayerInfo(ctx context.Context, req *socialpb.GetPlayerInfoRequest) (*socialpb.PlayerInfoResponse, error) {
	h.logger.Info("查询玩家信息")
	return &socialpb.PlayerInfoResponse{}, nil
}

// GetDiplomacyStatus 查询外交状态。
func (h *SocialHandler) GetDiplomacyStatus(ctx context.Context, req *socialpb.GetDiplomacyRequest) (*socialpb.DiplomacyResponse, error) {
	h.logger.Info("查询外交状态")
	return &socialpb.DiplomacyResponse{}, nil
}

// CheckAlliancePermission 校验联盟权限。
func (h *SocialHandler) CheckAlliancePermission(ctx context.Context, req *socialpb.CheckPermissionRequest) (*socialpb.PermissionResponse, error) {
	h.logger.Info("校验联盟权限")
	return &socialpb.PermissionResponse{}, nil
}

// GetPlayerAllianceTerritory 查询玩家联盟领地。
func (h *SocialHandler) GetPlayerAllianceTerritory(ctx context.Context, req *socialpb.GetTerritoryRequest) (*socialpb.TerritoryResponse, error) {
	h.logger.Info("查询玩家联盟领地")
	return &socialpb.TerritoryResponse{}, nil
}
