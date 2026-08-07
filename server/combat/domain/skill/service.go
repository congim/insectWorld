// Package skill 技能释放domain service，负责技能触发条件判定与效果执行。
// SkillService为无状态domain service，技能释放是战斗聚合根内编排逻辑（规范4）。
package skill

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	combaterr "insectworld/server/combat/domain/errors"
	"insectworld/server/shared/pkg/config"
)

// 技能触发阶段常量（规范1）。
const (
	TriggerPhaseRoundStart = 1 // 轮次开始
	TriggerPhaseRoundMid   = 2 // 轮次中
	TriggerPhaseRoundEnd   = 3 // 轮次结束
)

// 技能效果类型常量（规范1）。
const (
	EffectTypeDamage = 1 // 伤害
	EffectTypeHeal   = 2 // 治疗
	EffectTypeBuff   = 3 // 增益
	EffectTypeDebuff = 4 // 减益
)

// SkillTrigger 技能触发记录，记录技能在战斗中的触发信息。
type SkillTrigger struct {
	SkillID      int64 // 技能ID（规范8用int64）
	TriggerRound int   // 触发轮次
	EffectType   int   // 效果类型：1=伤害 2=治疗 3=增益 4=减益
	EffectValue  int64 // 效果数值（规范8用int64）
}

// SkillCooldownRecord 技能冷却记录。
type SkillCooldownRecord struct {
	SkillID          int64 // 技能ID
	CooldownEndRound int   // 冷却结束轮次
}

// SkillService 技能释放domain service，负责技能触发条件判定与效果执行。
type SkillService struct {
	configQuery config.ConfigQueryAPI         // 配置查询接口，查询combat.combat_skills
	logger      *zap.Logger                   // 结构化日志器（规范7）
	cooldowns   map[int64]SkillCooldownRecord // 技能冷却记录，key=技能ID
}

// NewSkillService 创建技能释放domain service实例。
func NewSkillService(configQuery config.ConfigQueryAPI, logger *zap.Logger) *SkillService {
	return &SkillService{
		configQuery: configQuery,
		logger:      logger,
		cooldowns:   make(map[int64]SkillCooldownRecord),
	}
}

// CheckTrigger 校验技能触发条件是否满足。
// skillID为技能ID，currentRound为当前轮次，hpRatio为当前血量比例（0-100）。
func (s *SkillService) CheckTrigger(ctx context.Context, skillID int64, currentRound int, hpRatio int) bool {
	skill := s.configQuery.GetCombatSkill(ctx, fmt.Sprintf("%d", skillID))
	if skill == nil {
		return false
	}

	// 冷却校验
	if cd, ok := s.cooldowns[skillID]; ok {
		if currentRound < cd.CooldownEndRound {
			return false
		}
	}

	// TODO 后续接入完整触发条件判定（hp_below_30%等）
	_ = hpRatio

	return true
}

// Execute 执行技能效果，返回技能触发记录。
// skillID为技能ID，combatID为战斗ID，currentRound为当前轮次。
func (s *SkillService) Execute(ctx context.Context, skillID, combatID int64, currentRound int) (*SkillTrigger, error) {
	skill := s.configQuery.GetCombatSkill(ctx, fmt.Sprintf("%d", skillID))
	if skill == nil {
		return nil, fmt.Errorf("技能执行失败，技能不存在，skillID=%d: %w", skillID, combaterr.ErrInvalidParams)
	}

	// 冷却校验
	if cd, ok := s.cooldowns[skillID]; ok {
		if currentRound < cd.CooldownEndRound {
			return nil, fmt.Errorf("技能冷却中，skillID=%d，剩余冷却轮次=%d: %w", skillID, cd.CooldownEndRound-currentRound, combaterr.ErrSkillCooldown)
		}
	}

	// 记算效果数值（TODO 后续接入完整公式计算）
	effectValue := skill.EffectValue

	// 记算冷却
	s.cooldowns[skillID] = SkillCooldownRecord{
		SkillID:          skillID,
		CooldownEndRound: currentRound + skill.CooldownRounds,
	}

	trigger := &SkillTrigger{
		SkillID:      skillID,
		TriggerRound: currentRound,
		EffectType:   skill.EffectType,
		EffectValue:  effectValue,
	}

	s.logger.Info("技能释放成功",
		zap.Int64("combat_id", combatID),
		zap.Int64("skill_id", skillID),
		zap.Int("trigger_round", currentRound),
		zap.Int("effect_type", skill.EffectType),
		zap.Int64("effect_value", effectValue),
	)

	return trigger, nil
}

// IsOnCooldown 判断技能是否在冷却中。
func (s *SkillService) IsOnCooldown(skillID int64, currentRound int) bool {
	cd, ok := s.cooldowns[skillID]
	if !ok {
		return false
	}
	return currentRound < cd.CooldownEndRound
}
