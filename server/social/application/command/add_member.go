
// Package command social服务application写侧，编排alliance聚合根的成员变更。
// handler模式与world/application/command对齐（注入仓储+outbox+logger）。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/social/domain/alliance"
)

// Outbox 领域事件Outbox接口，保证事件不丢不重（规范3在domain层声明，此处引用）。
type Outbox interface {
	// Append 追加领域事件到Outbox表
	Append(ctx context.Context, event any) error
}

// AddMemberCommand 添加联盟成员命令DTO。
type AddMemberCommand struct {
	AllianceID int64 // 联盟ID
	PlayerID   int64 // 玩家ID
	Now        int64 // 当前时间戳毫秒，用于加入冷却校验
}

// AddMemberHandler 添加联盟成员命令处理器。
type AddMemberHandler struct {
	allianceRepo alliance.AllianceRepository // Alliance聚合根仓储接口
	outbox       Outbox                       // 领域事件Outbox接口
	logger       *zap.Logger                  // 结构化日志器（规范7）
}

// NewAddMemberHandler 创建添加成员命令处理器实例。
func NewAddMemberHandler(
	allianceRepo alliance.AllianceRepository,
	outbox Outbox,
	logger *zap.Logger,
) *AddMemberHandler {
	return &AddMemberHandler{
		allianceRepo: allianceRepo,
		outbox:       outbox,
		logger:       logger,
	}
}

// Handle 处理添加联盟成员命令。
// 编排：加载Alliance聚合根 → 调用AddMember → 保存聚合根 → 写Outbox投递MemberChangedEvent。
func (h *AddMemberHandler) Handle(ctx context.Context, cmd AddMemberCommand) error {
	// 1. 加载Alliance聚合根
	a, err := h.allianceRepo.LoadAlliance(ctx, cmd.AllianceID)
	if err != nil {
		return fmt.Errorf("添加成员失败，加载联盟失败，allianceID=%d: %w", cmd.AllianceID, err)
	}

	// 2. 调用聚合根方法添加成员
	event, err := a.AddMember(ctx, cmd.PlayerID, cmd.Now)
	if err != nil {
		return fmt.Errorf("添加成员失败，allianceID=%d，playerID=%d: %w",
			cmd.AllianceID, cmd.PlayerID, err)
	}

	// 3. 保存聚合根
	if err := h.allianceRepo.SaveAlliance(ctx, a); err != nil {
		return fmt.Errorf("添加成员失败，保存联盟失败，allianceID=%d: %w", cmd.AllianceID, err)
	}

	// 4. 写Outbox投递领域事件
	if err := h.outbox.Append(ctx, event); err != nil {
		return fmt.Errorf("添加成员失败，写Outbox失败，allianceID=%d: %w", cmd.AllianceID, err)
	}

	h.logger.Info("添加联盟成员成功",
		zap.Int64("alliance_id", cmd.AllianceID),
		zap.Int64("player_id", cmd.PlayerID),
	)

	return nil
}