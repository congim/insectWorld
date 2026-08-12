// Package command Combat服务application层命令，编排domain层聚合根与技能释放。
// 本文件定义ExecuteRoundCommand战斗轮次执行编排，对应design.md 2.2.2.2节；
// E3-S4起接入公式引擎计算伤害，种子派生绑定快照config_version（ADR-004 3.5）。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/combat/domain/combat"
	combaterr "insectworld/server/combat/domain/errors"
	"insectworld/server/combat/domain/skill"
	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/formula"
)

// ExecuteRoundCommand 战斗轮次执行命令DTO。
type ExecuteRoundCommand struct {
	CombatID int64 // 战斗ID（规范8用int64）
}

// FormulaEvaluator 伤害公式求值接口，轮次执行编排依赖最小接口（规范9接口小化），
// 真实实现为formula.FormulaEngine，测试用mock替换。
type FormulaEvaluator interface {
	// EvalByID 按公式ID求值，公式未注册返回错误。
	EvalByID(formulaID string, vars map[string]float64, rand formula.RandSource) (float64, error)
}

// ExecuteRoundHandler 战斗轮次执行命令处理器，编排战斗轮次执行全流程。
type ExecuteRoundHandler struct {
	combatRepo    combat.CombatRepository // Combat聚合根仓储接口
	configQuery   config.ConfigQueryAPI   // 配置查询接口
	formulaEngine FormulaEvaluator        // 伤害公式引擎（真实为formula.FormulaEngine）
	skillService  *skill.SkillService     // 技能释放domain service
	outbox        Outbox                  // 领域事件Outbox接口
	logger        *zap.Logger             // 结构化日志器（规范7）
}

// Outbox 领域事件Outbox接口。
type Outbox interface {
	Append(ctx context.Context, event any) error
}

// NewExecuteRoundHandler 创建战斗轮次执行命令处理器实例。
func NewExecuteRoundHandler(
	combatRepo combat.CombatRepository,
	configQuery config.ConfigQueryAPI,
	formulaEngine FormulaEvaluator,
	skillService *skill.SkillService,
	outbox Outbox,
	logger *zap.Logger,
) *ExecuteRoundHandler {
	return &ExecuteRoundHandler{
		combatRepo:    combatRepo,
		configQuery:   configQuery,
		formulaEngine: formulaEngine,
		skillService:  skillService,
		outbox:        outbox,
		logger:        logger,
	}
}

// Handle 处理战斗轮次执行命令。
// 编排：长时战斗每轮刷新属性→应用阵型效果（轮次开始）→执行轮次→技能释放→
// 应用阵型效果（轮次结束）→轮数超限平局→保存+Outbox。
func (h *ExecuteRoundHandler) Handle(ctx context.Context, cmd ExecuteRoundCommand) error {
	// 1. 加载Combat聚合根
	c, err := h.combatRepo.LoadCombat(ctx, cmd.CombatID)
	if err != nil {
		return fmt.Errorf("轮次执行失败，加载战斗失败，combatID=%d: %w", cmd.CombatID, err)
	}
	if c == nil {
		return fmt.Errorf("轮次执行失败，战斗不存在，combatID=%d: %w", cmd.CombatID, combaterr.ErrCombatNotFound)
	}

	// 2. 长时战斗每轮开始前刷新属性快照
	if c.IsLongCombat() {
		h.refreshAttributes(ctx, c)
	}

	// 3. 应用阵型效果（轮次开始）
	h.applyFormationEffects(ctx, c, combat.FormationApplyRoundStart)

	// 4. 执行轮次
	roundEvent, err := c.ExecuteRound()
	if err != nil {
		return fmt.Errorf("轮次执行失败，combatID=%d: %w", cmd.CombatID, err)
	}

	// 4.1 计算本轮基础伤害（按快照版本公式引擎，种子派生绑定快照config_version）
	if err := h.calculateDamage(ctx, c, roundEvent); err != nil {
		return fmt.Errorf("轮次执行失败，伤害计算失败，combatID=%d: %w", cmd.CombatID, err)
	}

	// 5. 技能释放（触发条件判定+效果执行+冷却）
	h.executeSkills(ctx, c)

	// 6. 应用阵型效果（轮次结束）
	h.applyFormationEffects(ctx, c, combat.FormationApplyRoundEnd)

	// 7. 轮数超限强制平局
	if c.CheckMaxRounds() {
		endEvent, err := c.End(combat.ResultDraw)
		if err != nil {
			return fmt.Errorf("轮次执行失败，强制平局失败，combatID=%d: %w", cmd.CombatID, err)
		}
		if err := h.outbox.Append(ctx, endEvent); err != nil {
			return fmt.Errorf("轮次执行失败，写Outbox失败: %w", err)
		}
	}

	// 8. 保存聚合根
	if err := h.combatRepo.SaveCombat(ctx, c); err != nil {
		return fmt.Errorf("轮次执行失败，保存战斗失败: %w", err)
	}

	// 9. 写Outbox投递轮次完成事件
	if err := h.outbox.Append(ctx, roundEvent); err != nil {
		return fmt.Errorf("轮次执行失败，写Outbox失败: %w", err)
	}

	h.logger.Info("战斗轮次执行成功",
		zap.Int64("combat_id", cmd.CombatID),
		zap.Int("current_round", c.CurrentRound()),
		zap.Int("max_rounds", c.MaxRounds()),
	)

	return nil
}

// refreshAttributes 刷新参战方属性快照（长时战斗每轮开始前）。
func (h *ExecuteRoundHandler) refreshAttributes(ctx context.Context, c *combat.Combat) {
	// TODO 后续通过gRPC查询Social拉取最新属性快照
	h.logger.Debug("刷新属性快照",
		zap.Int64("combat_id", c.CombatID()),
	)
}

// applyFormationEffects 应用阵型效果到战斗。
func (h *ExecuteRoundHandler) applyFormationEffects(ctx context.Context, c *combat.Combat, timing int) {
	// TODO 后续从config查询阵型效果配置并应用
	_ = timing
}

// executeSkills 执行技能释放编排。
func (h *ExecuteRoundHandler) executeSkills(ctx context.Context, c *combat.Combat) {
	// TODO 后续从config查询战斗类型的技能列表，逐技能判定触发条件并执行
}

// calculateDamage 按快照版本公式计算本轮基础伤害并写入轮次事件（E3-S4）。
// 公式ID取快照冻结值（开战绑定，ADR-004 3.1），种子取hash(combatID, config_version, round)，
// config_version恒等于快照值，热更/回滚不改变，保证重放种子不漂移（ADR-001 3.3 / ADR-004 3.5）。
func (h *ExecuteRoundHandler) calculateDamage(ctx context.Context, c *combat.Combat, roundEvent *combat.RoundCompletedEvent) error {
	snap := c.Snapshot()
	if snap.FormulaID == "" {
		// 未绑定公式的战斗（旧数据兼容）不计算伤害
		return nil
	}
	vars := h.buildDamageVars(snap)
	seed := combat.DeriveRoundSeed(c.CombatID(), snap.ConfigVersion, roundEvent.Round)
	rand := formula.NewPcg32(seed)
	val, err := h.formulaEngine.EvalByID(snap.FormulaID, vars, rand)
	if err != nil {
		return fmt.Errorf("伤害公式求值失败，formulaID=%s: %w", snap.FormulaID, err)
	}
	// 离散量伤害取整落库（AGENTS.md规范8）
	roundEvent.Damage = int64(val)
	return nil
}

// buildDamageVars 构造伤害公式求值变量表。
// 本期取攻击方首个实体属性作为基础值（P3骨头阶段扩展到全编队逐实体结算）；
// counter/terrain_mod暂为1（克制矩阵与地形修正后续接入，配置数值可为小数规范8例外）。
func (h *ExecuteRoundHandler) buildDamageVars(snap combat.CombatSnapshot) map[string]float64 {
	vars := map[string]float64{
		"atk":         0,
		"def":         0,
		"counter":     1,
		"terrain_mod": 1,
	}
	for _, prop := range snap.AttackerProps {
		vars["atk"] = float64(prop.Atk)
		vars["def"] = float64(prop.Def)
		break
	}
	return vars
}
