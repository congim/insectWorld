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
	AdminOpReloadConfig         = 1 // 配置热更
	AdminOpRollbackConfig       = 2 // 配置回滚
	AdminOpForceTransitionPhase = 3 // 强制切换赛季阶段
	AdminOpForceResetSeason     = 4 // 强制重置赛季
	AdminOpManualTriggerEvent   = 5 // 手动触发事件
	AdminOpBanPlayer            = 6 // 封禁玩家
	AdminOpUnbanPlayer          = 7 // 解封玩家
	AdminOpPublishNotice        = 8 // 发布公告
)

// AdminHandler 运营管理操作gRPC handler，实现AdminServiceServer接口。
// 所有管理操作经过运营管理员鉴权+审计日志记录（规范7审计日志独立存储）。
type AdminHandler struct {
	adminpb.UnimplementedAdminServiceServer             // 嵌入未实现的服务器，保证向前兼容
	authChecker                             AuthChecker // 鉴权接口，校验运营管理员Token
	auditLogger                             AuditLogger // 审计日志接口，独立存储审计记录
	configAdmin                             ConfigAdmin // 配置管理接口
	seasonAdmin                             SeasonAdmin // 赛季管理接口
	eventAdmin                              EventAdmin  // 事件管理接口
	playerAdmin                             PlayerAdmin // 玩家管理接口
	noticeAdmin                             NoticeAdmin // 公告管理接口
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

// ConfigAdmin 配置管理接口。
type ConfigAdmin interface {
	ReloadConfig(ctx context.Context, configVersion int64) (int64, error)
	RollbackConfig(ctx context.Context, targetVersion int64) (int64, error)
}

// SeasonAdmin 赛季管理接口。
type SeasonAdmin interface {
	ForceTransitionPhase(ctx context.Context, seasonID int64, targetPhase int) (int, error)
	ForceResetSeason(ctx context.Context, seasonID int64) error
}

// EventAdmin 事件管理接口。
type EventAdmin interface {
	ManualTriggerEvent(ctx context.Context, eventID int64) error
}

// PlayerAdmin 玩家管理接口。
type PlayerAdmin interface {
	BanPlayer(ctx context.Context, playerID int64, durationMs int64, reason string) error
	UnbanPlayer(ctx context.Context, playerID int64) error
}

// NoticeAdmin 公告管理接口。
type NoticeAdmin interface {
	PublishNotice(ctx context.Context, noticeType int, content string, durationMs int64, targetIDs []int64) (int64, error)
}

// NewAdminHandler 创建运营管理操作handler实例。
func NewAdminHandler(
	authChecker AuthChecker,
	auditLogger AuditLogger,
	configAdmin ConfigAdmin,
	seasonAdmin SeasonAdmin,
	eventAdmin EventAdmin,
	playerAdmin PlayerAdmin,
	noticeAdmin NoticeAdmin,
	logger *zap.Logger,
) *AdminHandler {
	return &AdminHandler{
		authChecker: authChecker,
		auditLogger: auditLogger,
		configAdmin: configAdmin,
		seasonAdmin: seasonAdmin,
		eventAdmin:  eventAdmin,
		playerAdmin: playerAdmin,
		noticeAdmin: noticeAdmin,
		logger:      logger,
	}
}

// ReloadConfig 触发配置热更。
func (h *AdminHandler) ReloadConfig(ctx context.Context, req *adminpb.ReloadConfigRequest) (*adminpb.ReloadConfigResponse, error) {
	adminID, err := h.authenticate(ctx, req.GetAdminToken(), AdminOpReloadConfig, 0)
	if err != nil {
		return nil, err
	}

	appliedVersion, err := h.configAdmin.ReloadConfig(ctx, req.GetConfigVersion())
	if err != nil {
		h.auditLogger.LogAudit(ctx, adminID, AdminOpReloadConfig, 0, fmt.Sprintf("版本=%d", req.GetConfigVersion()), false)
		return nil, fmt.Errorf("配置热更失败: %w", err)
	}

	h.auditLogger.LogAudit(ctx, adminID, AdminOpReloadConfig, 0, fmt.Sprintf("版本=%d→%d", req.GetConfigVersion(), appliedVersion), true)
	h.logger.Info("配置热更成功", zap.Int64("admin_id", adminID), zap.Int64("applied_version", appliedVersion))

	return &adminpb.ReloadConfigResponse{
		Header:         h.successHeader(),
		AppliedVersion: appliedVersion,
	}, nil
}

// RollbackConfig 回滚配置。
func (h *AdminHandler) RollbackConfig(ctx context.Context, req *adminpb.RollbackConfigRequest) (*adminpb.RollbackConfigResponse, error) {
	adminID, err := h.authenticate(ctx, req.GetAdminToken(), AdminOpRollbackConfig, 0)
	if err != nil {
		return nil, err
	}

	currentVersion, err := h.configAdmin.RollbackConfig(ctx, req.GetTargetVersion())
	if err != nil {
		h.auditLogger.LogAudit(ctx, adminID, AdminOpRollbackConfig, 0, fmt.Sprintf("目标版本=%d", req.GetTargetVersion()), false)
		return nil, fmt.Errorf("配置回滚失败: %w", err)
	}

	h.auditLogger.LogAudit(ctx, adminID, AdminOpRollbackConfig, 0, fmt.Sprintf("回滚到版本=%d", currentVersion), true)
	return &adminpb.RollbackConfigResponse{Header: h.successHeader(), CurrentVersion: currentVersion}, nil
}

// ForceTransitionPhase 强制切换赛季阶段。
func (h *AdminHandler) ForceTransitionPhase(ctx context.Context, req *adminpb.ForceTransitionRequest) (*adminpb.ForceTransitionResponse, error) {
	adminID, err := h.authenticate(ctx, req.GetAdminToken(), AdminOpForceTransitionPhase, req.GetSeasonId())
	if err != nil {
		return nil, err
	}

	prevPhase, err := h.seasonAdmin.ForceTransitionPhase(ctx, req.GetSeasonId(), int(req.GetTargetPhase()))
	if err != nil {
		h.auditLogger.LogAudit(ctx, adminID, AdminOpForceTransitionPhase, req.GetSeasonId(), fmt.Sprintf("目标阶段=%d", req.GetTargetPhase()), false)
		return nil, fmt.Errorf("强制切换赛季阶段失败: %w", err)
	}

	h.auditLogger.LogAudit(ctx, adminID, AdminOpForceTransitionPhase, req.GetSeasonId(), fmt.Sprintf("阶段%d→%d", prevPhase, req.GetTargetPhase()), true)
	return &adminpb.ForceTransitionResponse{Header: h.successHeader(), PreviousPhase: int32(prevPhase)}, nil
}

// ForceResetSeason 强制重置赛季。
func (h *AdminHandler) ForceResetSeason(ctx context.Context, req *adminpb.ForceResetRequest) (*adminpb.ForceResetResponse, error) {
	adminID, err := h.authenticate(ctx, req.GetAdminToken(), AdminOpForceResetSeason, req.GetSeasonId())
	if err != nil {
		return nil, err
	}

	if err := h.seasonAdmin.ForceResetSeason(ctx, req.GetSeasonId()); err != nil {
		h.auditLogger.LogAudit(ctx, adminID, AdminOpForceResetSeason, req.GetSeasonId(), "重置", false)
		return nil, fmt.Errorf("强制重置赛季失败: %w", err)
	}

	h.auditLogger.LogAudit(ctx, adminID, AdminOpForceResetSeason, req.GetSeasonId(), "重置", true)
	return &adminpb.ForceResetResponse{Header: h.successHeader(), Reset_: true}, nil
}

// ManualTriggerEvent 手动触发游戏事件。
func (h *AdminHandler) ManualTriggerEvent(ctx context.Context, req *adminpb.ManualTriggerRequest) (*adminpb.ManualTriggerResponse, error) {
	adminID, err := h.authenticate(ctx, req.GetAdminToken(), AdminOpManualTriggerEvent, req.GetEventId())
	if err != nil {
		return nil, err
	}

	if err := h.eventAdmin.ManualTriggerEvent(ctx, req.GetEventId()); err != nil {
		h.auditLogger.LogAudit(ctx, adminID, AdminOpManualTriggerEvent, req.GetEventId(), "触发", false)
		return nil, fmt.Errorf("手动触发事件失败: %w", err)
	}

	h.auditLogger.LogAudit(ctx, adminID, AdminOpManualTriggerEvent, req.GetEventId(), "触发", true)
	return &adminpb.ManualTriggerResponse{Header: h.successHeader(), Triggered: true}, nil
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

// PublishNotice 发布公告。
func (h *AdminHandler) PublishNotice(ctx context.Context, req *adminpb.PublishNoticeRequest) (*adminpb.PublishNoticeResponse, error) {
	adminID, err := h.authenticate(ctx, req.GetAdminToken(), AdminOpPublishNotice, 0)
	if err != nil {
		return nil, err
	}

	noticeID, err := h.noticeAdmin.PublishNotice(ctx, int(req.GetNoticeType()), req.GetContent(), req.GetDurationMs(), req.GetTargetIds())
	if err != nil {
		h.auditLogger.LogAudit(ctx, adminID, AdminOpPublishNotice, 0, req.GetContent(), false)
		return nil, fmt.Errorf("发布公告失败: %w", err)
	}

	h.auditLogger.LogAudit(ctx, adminID, AdminOpPublishNotice, noticeID, fmt.Sprintf("类型=%d", req.GetNoticeType()), true)
	return &adminpb.PublishNoticeResponse{Header: h.successHeader(), NoticeId: noticeID}, nil
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
