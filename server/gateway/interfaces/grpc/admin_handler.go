// Package grpc Gateway服务interfaces层gRPC handler，实现运营管理操作协议。
// AdminHandler实现AdminService gRPC接口，所有管理操作经过鉴权+审计日志记录。
package grpc

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
	adminpb "insectworld/server/shared/proto/admin"
	commonpb "insectworld/server/shared/proto/common"
)

// 管理操作类型常量（规范1），用于审计日志标识操作类型。
const (
	AdminOpBanPlayer   = 6 // 封禁玩家
	AdminOpUnbanPlayer = 7 // 解封玩家
)

// AdminHandler 运营管理操作gRPC handler，实现AdminServiceServer接口。
// 所有管理操作经过运营管理员鉴权+审计日志记录（规范7审计日志独立存储）。
// 非Gateway职责方法（配置热更/赛季管理/事件触发/公告）由UnimplementedAdminServiceServer嵌入兜底。
type AdminHandler struct {
	adminpb.UnimplementedAdminServiceServer             // 嵌入未实现的服务器，非Gateway职责方法返回Unimplemented错误
	authChecker                             AuthChecker // 鉴权接口，校验运营管理员Token
	auditLogger                             AuditLogger // 审计日志接口，独立存储审计记录
	playerAdmin                             PlayerAdmin // 玩家管理接口
	logger                                  *zap.Logger // 结构化日志器（规范7）
}

// AuthChecker 鉴权接口，校验运营管理员Token。
type AuthChecker interface {
	CheckAdminToken(ctx context.Context, token string) (adminID int64, err error)
}

// AuditLogger 审计日志接口，独立存储审计记录（规范7审计日志独立）。
type AuditLogger interface {
	LogAudit(ctx context.Context, adminID int64, opType int, targetID int64, content string, result bool)
}

// PlayerAdmin 玩家管理接口。
type PlayerAdmin interface {
	BanPlayer(ctx context.Context, playerID int64, durationMs int64, reason string) error
	UnbanPlayer(ctx context.Context, playerID int64) error
}

// NewAdminHandler 创建运营管理操作handler实例。
//
// 仅接收Gateway职责内的依赖（鉴权/审计/玩家管理），非Gateway职责方法由UnimplementedAdminServiceServer兜底。
func NewAdminHandler(
	authChecker AuthChecker,
	auditLogger AuditLogger,
	playerAdmin PlayerAdmin,
	logger *zap.Logger,
) *AdminHandler {
	return &AdminHandler{
		authChecker: authChecker,
		auditLogger: auditLogger,
		playerAdmin: playerAdmin,
		logger:      logger,
	}
}

// BanPlayer 封禁玩家。
func (h *AdminHandler) BanPlayer(ctx context.Context, req *adminpb.BanPlayerRequest) (*adminpb.BanPlayerResponse, error) {
	adminID, err := h.authenticate(ctx, req.GetAdminToken(), AdminOpBanPlayer, req.GetPlayerId())
	if err != nil {
		return nil, err
	}

	if err := h.playerAdmin.BanPlayer(ctx, req.GetPlayerId(), req.GetDurationMs(), req.GetReason()); err != nil {
		h.auditLogger.LogAudit(ctx, adminID, AdminOpBanPlayer, req.GetPlayerId(), req.GetReason(), false)
		return nil, fmt.Errorf("封禁玩家失败: %w", err)
	}

	h.auditLogger.LogAudit(ctx, adminID, AdminOpBanPlayer, req.GetPlayerId(), fmt.Sprintf("封禁%dms，原因=%s", req.GetDurationMs(), req.GetReason()), true)
	h.logger.Info("玩家封禁成功", zap.Int64("admin_id", adminID), zap.Int64("player_id", req.GetPlayerId()), zap.Int64("duration_ms", req.GetDurationMs()))
	return &adminpb.BanPlayerResponse{Header: h.successHeader(), Banned: true}, nil
}

// UnbanPlayer 解封玩家。
func (h *AdminHandler) UnbanPlayer(ctx context.Context, req *adminpb.UnbanPlayerRequest) (*adminpb.UnbanPlayerResponse, error) {
	adminID, err := h.authenticate(ctx, req.GetAdminToken(), AdminOpUnbanPlayer, req.GetPlayerId())
	if err != nil {
		return nil, err
	}

	if err := h.playerAdmin.UnbanPlayer(ctx, req.GetPlayerId()); err != nil {
		h.auditLogger.LogAudit(ctx, adminID, AdminOpUnbanPlayer, req.GetPlayerId(), "解封", false)
		return nil, fmt.Errorf("解封玩家失败: %w", err)
	}

	h.auditLogger.LogAudit(ctx, adminID, AdminOpUnbanPlayer, req.GetPlayerId(), "解封", true)
	return &adminpb.UnbanPlayerResponse{Header: h.successHeader(), Unbanned: true}, nil
}

// authenticate 鉴权，校验运营管理员Token。
func (h *AdminHandler) authenticate(ctx context.Context, token string, opType int, targetID int64) (int64, error) {
	adminID, err := h.authChecker.CheckAdminToken(ctx, token)
	if err != nil {
		h.logger.Warn("管理操作鉴权失败",
			zap.Int("op_type", opType),
			zap.Int64("target_id", targetID),
			zap.Error(err),
		)
		return 0, fmt.Errorf("鉴权失败: %w", gatewayerr.ErrTokenInvalid)
	}
	return adminID, nil
}

// successHeader 构造成功响应头。
func (h *AdminHandler) successHeader() *commonpb.ResponseHeader {
	return &commonpb.ResponseHeader{
		ErrorCode: 0,
		ErrorMsg:  "成功",
	}
}
