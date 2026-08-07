// Package command Combat服务application层命令，编排domain层聚合根与技能释放。
// 本文件定义ExecuteRoundHandler的单元测试。
package command

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/combat/domain/combat"
	"insectworld/server/combat/domain/skill"

	"insectworld/server/shared/pkg/config/mock"
)

// mockCombatRepository Combat仓储mock实现。
type mockCombatRepository struct {
	mu      sync.Mutex
	combat  *combat.Combat
	loadErr error
	saveErr error
}

// LoadCombat 加载战斗聚合根的mock实现。
func (r *mockCombatRepository) LoadCombat(ctx context.Context, combatID int64) (*combat.Combat, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r.combat, nil
}

// SaveCombat 保存战斗聚合根的mock实现。
func (r *mockCombatRepository) SaveCombat(ctx context.Context, c *combat.Combat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.combat = c
	if r.saveErr != nil {
		return r.saveErr
	}
	return nil
}

// mockOutbox Outbox mock实现。
type mockOutbox struct {
	events []any
	err    error
}

// Append 写Outbox的mock实现。
func (o *mockOutbox) Append(ctx context.Context, event any) error {
	if o.err != nil {
		return o.err
	}
	o.events = append(o.events, event)
	return nil
}

// TestExecuteRoundHandler_Success 测试战斗轮次执行成功。
func TestExecuteRoundHandler_Success(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := combat.NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.combat.CurrentRound())
	assert.Len(t, outbox.events, 1)
}

// TestExecuteRoundHandler_CombatNotFound 测试战斗不存在时执行失败。
func TestExecuteRoundHandler_CombatNotFound(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	repo := &mockCombatRepository{combat: nil}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 999})
	assert.Error(t, err)
}

// TestExecuteRoundHandler_LoadError 测试加载战斗失败。
func TestExecuteRoundHandler_LoadError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	repo := &mockCombatRepository{loadErr: assert.AnError}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	assert.Error(t, err)
}

// TestExecuteRoundHandler_MaxRoundsForceDraw 测试轮数超限强制平局。
func TestExecuteRoundHandler_MaxRoundsForceDraw(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	// maxRounds=1，执行1轮后超限强制平局
	c := combat.NewCombat(1, 1, 1, []int64{101}, []int64{201}, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	require.NoError(t, err)
	assert.Equal(t, combat.StatusEnded, repo.combat.Status())
	// 超限平局事件 + 轮次完成事件 = 2个事件
	assert.Len(t, outbox.events, 2)
}

// TestExecuteRoundHandler_SaveError 测试保存战斗失败。
func TestExecuteRoundHandler_SaveError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := combat.NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)
	repo := &mockCombatRepository{combat: c, saveErr: assert.AnError}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	assert.Error(t, err)
}

// TestExecuteRoundHandler_OutboxError 测试写Outbox失败。
func TestExecuteRoundHandler_OutboxError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := combat.NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{err: assert.AnError}

	handler := NewExecuteRoundHandler(repo, cfg, skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	assert.Error(t, err)
}

// TestExecuteRoundHandler_MultipleRounds 测试多轮次执行。
func TestExecuteRoundHandler_MultipleRounds(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := combat.NewCombat(1, 1, 5, []int64{101}, []int64{201}, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, skillSvc, outbox, logger)

	// 执行3轮
	for i := 0; i < 3; i++ {
		err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
		require.NoError(t, err)
	}

	assert.Equal(t, 3, repo.combat.CurrentRound())
	assert.True(t, repo.combat.IsInProgress())
	// 3轮，每轮1个轮次完成事件
	assert.Len(t, outbox.events, 3)
}
