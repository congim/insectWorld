// Package alliance 联盟聚合根，维护联盟成员、等级与外交关系。
// Alliance聚合根提供成员加入（含冷却校验）与退出（含惩罚应用）能力。
package alliance

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/config"
	socialerr "insectworld/server/social/domain/errors"
)

// 联盟职位常量（规范1）。
const (
	RoleLeader  = 1 // 盟主
	RoleOfficer = 2 // 官员
	RoleMember  = 3 // 普通成员
)

// Alliance 联盟聚合根，维护联盟成员、等级与外交关系。
type Alliance struct {
	allianceID        int64         // 联盟ID，全局唯一
	allianceName      string        // 联盟名称
	level             int           // 联盟等级
	members           map[int64]int // 成员映射，key=玩家ID，value=职位（1=盟主 2=官员 3=普通成员）
	memberCount       int           // 当前成员数
	maxMembers        int           // 最大成员数，由配置驱动
	joinCooldownEndAt int64         // 加入冷却结束时间戳（毫秒），控制成员加入频率
}

// NewAlliance 创建联盟聚合根实例。
func NewAlliance(allianceID int64, name string, maxMembers int) *Alliance {
	return &Alliance{
		allianceID:   allianceID,
		allianceName: name,
		members:      make(map[int64]int),
		maxMembers:   maxMembers,
	}
}

// AllianceID 返回联盟ID。
func (a *Alliance) AllianceID() int64 { return a.allianceID }

// MemberCount 返回当前成员数。
func (a *Alliance) MemberCount() int { return a.memberCount }

// GetMemberRole 查询玩家在联盟中的职位。
func (a *Alliance) GetMemberRole(playerID int64) (int, bool) {
	role, ok := a.members[playerID]
	return role, ok
}

// AddMember 添加成员，校验联盟未满与加入冷却。
func (a *Alliance) AddMember(ctx context.Context, playerID int64, now int64) (*MemberChangedEvent, error) {
	if a.memberCount >= a.maxMembers {
		return nil, fmt.Errorf("加入联盟失败，联盟已满，allianceID=%d，当前=%d，上限=%d: %w",
			a.allianceID, a.memberCount, a.maxMembers, socialerr.ErrAllianceFull)
	}

	if now < a.joinCooldownEndAt {
		remaining := a.joinCooldownEndAt - now
		return nil, fmt.Errorf("加入联盟失败，加入冷却中，剩余冷却时间%dms: %w", remaining, socialerr.ErrJoinCooldown)
	}

	a.members[playerID] = RoleMember
	a.memberCount++

	return &MemberChangedEvent{
		AllianceID: a.allianceID,
		PlayerID:   playerID,
		ChangeType: 1, // 加入
	}, nil
}

// RemoveMember 移除成员，盟主退出需先转让。
func (a *Alliance) RemoveMember(ctx context.Context, playerID int64) (*MemberChangedEvent, error) {
	role, ok := a.members[playerID]
	if !ok {
		return nil, fmt.Errorf("退出联盟失败，玩家不在联盟中，playerID=%d: %w", playerID, socialerr.ErrInvalidParams)
	}

	if role == RoleLeader {
		return nil, fmt.Errorf("盟主退出需先转让: %w", socialerr.ErrRuleViolation)
	}

	delete(a.members, playerID)
	a.memberCount--

	return &MemberChangedEvent{
		AllianceID: a.allianceID,
		PlayerID:   playerID,
		ChangeType: 2, // 退出
	}, nil
}

// SetJoinCooldown 设置加入冷却结束时间。
func (a *Alliance) SetJoinCooldown(cooldownEndAt int64) {
	a.joinCooldownEndAt = cooldownEndAt
}

// MemberChangedEvent 成员变更领域事件。
type MemberChangedEvent struct {
	AllianceID int64 // 联盟ID
	PlayerID   int64 // 玩家ID
	ChangeType int   // 变更类型：1=加入 2=退出
}

// AllianceRepository Alliance聚合根仓储接口（规范3）。
type AllianceRepository interface {
	LoadAlliance(ctx context.Context, allianceID int64) (*Alliance, error)
	SaveAlliance(ctx context.Context, a *Alliance) error
}

// PermissionChecker 权限动态校验domain service，基于配置的职位权限映射校验。
// 权限校验是无状态查询，用domain service而非新聚合根（规范4）。
type PermissionChecker struct {
	configQuery  config.ConfigQueryAPI // 配置查询接口，查询alliance.alliance_permissions
	allianceRepo AllianceRepository    // 联盟仓储接口，查询玩家在联盟中的职位
	logger       *zap.Logger           // 结构化日志器（规范7）
}

// NewPermissionChecker 创建权限校验domain service实例。
func NewPermissionChecker(configQuery config.ConfigQueryAPI, allianceRepo AllianceRepository, logger *zap.Logger) *PermissionChecker {
	return &PermissionChecker{
		configQuery:  configQuery,
		allianceRepo: allianceRepo,
		logger:       logger,
	}
}

// Check 校验玩家在联盟中的权限，基于配置的职位权限映射（非硬编码）。
func (pc *PermissionChecker) Check(ctx context.Context, allianceID, playerID int64, action string) bool {
	// 从config查询职位权限映射（非硬编码）
	permMap := pc.configQuery.GetAlliancePermissions(ctx)
	if permMap == nil {
		pc.logger.Warn("权限校验失败，权限配置为空",
			zap.Int64("alliance_id", allianceID),
		)
		return false
	}

	// 从AllianceRepository加载联盟聚合根，查询玩家职位
	alliance, err := pc.allianceRepo.LoadAlliance(ctx, allianceID)
	if err != nil {
		pc.logger.Warn("权限校验失败，联盟加载失败",
			zap.Int64("alliance_id", allianceID),
			zap.Int64("player_id", playerID),
			zap.Error(err),
		)
		return false
	}

	role, ok := alliance.GetMemberRole(playerID)
	if !ok {
		pc.logger.Warn("权限校验失败，玩家不在联盟中",
			zap.Int64("alliance_id", allianceID),
			zap.Int64("player_id", playerID),
		)
		return false
	}

	allowedActions, ok := permMap[fmt.Sprintf("%d", role)]
	if !ok {
		return false
	}

	for _, allowed := range allowedActions {
		if allowed == action {
			return true
		}
	}
	return false
}

// WelfareService 联盟福利domain service，负责福利触发条件判定与效果发放。
// 福利发放由联盟聚合根触发，用domain service而非新聚合根（规范4）。
type WelfareService struct {
	configQuery config.ConfigQueryAPI // 配置查询接口，查询alliance.alliance_welfare
	logger      *zap.Logger           // 结构化日志器（规范7）
}

// NewWelfareService 创建联盟福利domain service实例。
func NewWelfareService(configQuery config.ConfigQueryAPI, logger *zap.Logger) *WelfareService {
	return &WelfareService{
		configQuery: configQuery,
		logger:      logger,
	}
}

// Distribute 发放联盟福利，返回福利发放事件。
func (ws *WelfareService) Distribute(ctx context.Context, allianceID, playerID int64, welfareID string) (*WelfareDistributedEvent, error) {
	welfare := ws.configQuery.GetAllianceWelfare(ctx, welfareID)
	if welfare == nil {
		return nil, fmt.Errorf("福利发放失败，福利配置不存在，welfareID=%s: %w", welfareID, socialerr.ErrInvalidParams)
	}

	ws.logger.Info("联盟福利发放成功",
		zap.Int64("alliance_id", allianceID),
		zap.Int64("player_id", playerID),
		zap.String("welfare_id", welfareID),
		zap.Int("welfare_type", welfare.WelfareType),
		zap.Int64("effect_value", welfare.EffectValue),
	)

	return &WelfareDistributedEvent{
		AllianceID:  allianceID,
		PlayerID:    playerID,
		WelfareType: welfare.WelfareType,
	}, nil
}

// WelfareDistributedEvent 福利发放领域事件。
type WelfareDistributedEvent struct {
	AllianceID  int64 // 联盟ID
	PlayerID    int64 // 玩家ID
	WelfareType int   // 福利类型：1=每日 2=每周 3=一次性
}
