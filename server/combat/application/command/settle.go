// Package command Combat服务application层命令，编排domain层聚合根与技能释放。
// 本文件定义SettleCommand战斗结算命令，结算前校验快照配置仍存，缺失走熔断协议（ADR-004 3.2）。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/combat/domain/combat"
	combaterr "insectworld/server/combat/domain/errors"
	"insectworld/server/shared/pkg/config"
)

// SettleCommand 战斗结算命令DTO。
type SettleCommand struct {
	CombatID int64 // 战斗ID（规范8用int64）
}

// SettleHandler 战斗结算命令处理器，编排结算校验与熔断决策（ADR-004 3.2）。
// 结算流程：加载战斗→ValidateSnapshotConfig→无缺失正常结算 / 有缺失ResolveCircuitBreak→
// 结束战斗→发布combat.ended / combat.circuit_broken事件。
type SettleHandler struct {
	combatRepo  combat.CombatRepository   // Combat聚合根仓储接口
	configQuery config.ConfigQueryAPI     // 配置查询接口
	outbox      Outbox                    // 领域事件Outbox接口
	logger      *zap.Logger               // 结构化日志器（规范7）
	policy      combat.CircuitBreakPolicy // 熔断策略，由combat.json配置注入（本期默认强制平局）
}

// NewSettleHandler 创建战斗结算命令处理器实例。
// 熔断策略默认强制平局（经济安全性优先），运营可通过SetCircuitBreakPolicy注入兜底配置策略。
func NewSettleHandler(
	combatRepo combat.CombatRepository,
	configQuery config.ConfigQueryAPI,
	outbox Outbox,
	logger *zap.Logger,
) *SettleHandler {
	return &SettleHandler{
		combatRepo:  combatRepo,
		configQuery: configQuery,
		outbox:      outbox,
		logger:      logger,
		policy:      combat.DefaultCircuitBreakPolicy(),
	}
}

// SetCircuitBreakPolicy 注入熔断策略（运营配置combat.json时调用）。
func (h *SettleHandler) SetCircuitBreakPolicy(policy combat.CircuitBreakPolicy) {
	h.policy = policy
}

// Handle 处理战斗结算命令。
func (h *SettleHandler) Handle(ctx context.Context, cmd SettleCommand) error {
	// 1. 加载Combat聚合根
	c, err := h.combatRepo.LoadCombat(ctx, cmd.CombatID)
	if err != nil {
		return fmt.Errorf("战斗结算失败，加载战斗失败，combatID=%d: %w", cmd.CombatID, err)
	}
	if c == nil {
		return fmt.Errorf("战斗结算失败，战斗不存在，combatID=%d: %w", cmd.CombatID, combaterr.ErrCombatNotFound)
	}

	// 2. 校验快照引用的配置项在快照版本仍存在（ADR-004 3.2）
	snap := c.Snapshot()
	missing, err := combat.ValidateSnapshotConfig(ctx, &snap, h.configQuery)
	if err != nil {
		return fmt.Errorf("战斗结算失败，快照配置校验失败，combatID=%d: %w", cmd.CombatID, err)
	}

	var endEvent *combat.CombatEndedEvent
	if len(missing) > 0 {
		// 3a. 熔断路径：按策略决策结算结果（默认强制平局，不发放战利品）
		result, err := combat.ResolveCircuitBreak(ctx, h.policy, missing, h.configQuery, snap.ConfigVersion)
		if err != nil {
			return fmt.Errorf("战斗结算失败，熔断决策失败，combatID=%d: %w", cmd.CombatID, err)
		}
		endEvent, err = c.End(result)
		if err != nil {
			return fmt.Errorf("战斗结算失败，熔断结束失败，combatID=%d: %w", cmd.CombatID, err)
		}
		// 发布熔断事件供运营告警（combat.circuit_broken）
		circuitEvent := &combat.CircuitBrokenEvent{
			CombatID:       c.CombatID(),
			MissingConfigs: missing,
			Strategy:       h.policy.Strategy,
			Result:         result,
		}
		if err := h.outbox.Append(ctx, circuitEvent); err != nil {
			return fmt.Errorf("战斗结算失败，写熔断事件Outbox失败: %w", err)
		}
		h.logger.Warn("战斗结算熔断",
			zap.Int64("combat_id", cmd.CombatID),
			zap.Int("strategy", h.policy.Strategy),
			zap.Int("result", result),
			zap.Int("missing_count", len(missing)),
		)
	} else {
		// 3b. 正常结算路径：按快照版本计算战果（本期简化为攻击方胜，P3骨头阶段接入完整战果计算）
		endEvent, err = c.End(combat.ResultAttackerWin)
		if err != nil {
			return fmt.Errorf("战斗结算失败，结束失败，combatID=%d: %w", cmd.CombatID, err)
		}
	}

	// 4. 保存聚合根
	if err := h.combatRepo.SaveCombat(ctx, c); err != nil {
		return fmt.Errorf("战斗结算失败，保存战斗失败: %w", err)
	}

	// 5. 写Outbox投递战斗结束事件
	if err := h.outbox.Append(ctx, endEvent); err != nil {
		return fmt.Errorf("战斗结算失败，写Outbox失败: %w", err)
	}

	h.logger.Info("战斗结算成功",
		zap.Int64("combat_id", cmd.CombatID),
		zap.Int("result", endEvent.Result),
		zap.Int("total_rounds", endEvent.TotalRounds),
	)
	return nil
}
