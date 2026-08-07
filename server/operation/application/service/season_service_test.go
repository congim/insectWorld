// Package service Operation服务application层服务，编排赛季重置/继承/奖励发放。
// 本文件定义SeasonResetCoordinator、SeasonInheritService与RewardDistributor的单元测试。
package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/operation/domain/season"
	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/config/mock"
)

// mockSeasonRepository Season仓储mock实现。
type mockSeasonRepository struct {
	mu      sync.Mutex
	season  *season.Season
	loadErr error
	saveErr error
}

// LoadSeason 加载赛季的mock实现。
func (r *mockSeasonRepository) LoadSeason(ctx context.Context, seasonID int64) (*season.Season, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r.season, nil
}

// SaveSeason 保存赛季的mock实现。
func (r *mockSeasonRepository) SaveSeason(ctx context.Context, s *season.Season) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.season = s
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

// TestSeasonResetCoordinator_Reset 测试赛季重置成功。
func TestSeasonResetCoordinator_Reset(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.SeasonResetRules = &config.ResetRulesConfig{
		ResetScope: []string{"player", "alliance"},
		KeepData:   []string{"player_profile"},
	}
	logger := zap.NewNop()

	s := season.NewSeason(1, 1000)
	repo := &mockSeasonRepository{season: s}
	outbox := &mockOutbox{}

	coord := NewSeasonResetCoordinator(repo, cfg, outbox, logger)

	err := coord.Reset(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, season.PhaseEnded, repo.season.Phase())
	assert.Len(t, outbox.events, 1)
}

// TestSeasonResetCoordinator_Reset_NoConfig 测试重置规则配置为空时失败。
func TestSeasonResetCoordinator_Reset_NoConfig(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()

	s := season.NewSeason(1, 1000)
	repo := &mockSeasonRepository{season: s}
	outbox := &mockOutbox{}

	coord := NewSeasonResetCoordinator(repo, cfg, outbox, logger)

	err := coord.Reset(context.Background(), 1)
	assert.Error(t, err)
}

// TestSeasonResetCoordinator_Reset_LoadError 测试加载赛季失败。
func TestSeasonResetCoordinator_Reset_LoadError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.SeasonResetRules = &config.ResetRulesConfig{
		ResetScope: []string{"player"},
	}
	logger := zap.NewNop()

	repo := &mockSeasonRepository{loadErr: assert.AnError}
	outbox := &mockOutbox{}

	coord := NewSeasonResetCoordinator(repo, cfg, outbox, logger)

	err := coord.Reset(context.Background(), 1)
	assert.Error(t, err)
}

// TestSeasonResetCoordinator_Reset_SaveError 测试保存赛季失败。
func TestSeasonResetCoordinator_Reset_SaveError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.SeasonResetRules = &config.ResetRulesConfig{
		ResetScope: []string{"player"},
	}
	logger := zap.NewNop()

	s := season.NewSeason(1, 1000)
	repo := &mockSeasonRepository{season: s, saveErr: assert.AnError}
	outbox := &mockOutbox{}

	coord := NewSeasonResetCoordinator(repo, cfg, outbox, logger)

	err := coord.Reset(context.Background(), 1)
	assert.Error(t, err)
}

// TestSeasonResetCoordinator_Reset_OutboxError 测试写Outbox失败。
func TestSeasonResetCoordinator_Reset_OutboxError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.SeasonResetRules = &config.ResetRulesConfig{
		ResetScope: []string{"player"},
	}
	logger := zap.NewNop()

	s := season.NewSeason(1, 1000)
	repo := &mockSeasonRepository{season: s}
	outbox := &mockOutbox{err: assert.AnError}

	coord := NewSeasonResetCoordinator(repo, cfg, outbox, logger)

	err := coord.Reset(context.Background(), 1)
	assert.Error(t, err)
}

// TestSeasonInheritService_Inherit 测试赛季继承成功。
func TestSeasonInheritService_Inherit(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.SeasonInheritRules = []config.InheritRuleConfig{
		{RuleID: "1", DataType: "player_level", InheritRatio: 0.5, MaxInherit: 100},
	}
	logger := zap.NewNop()

	svc := NewSeasonInheritService(cfg, logger)

	err := svc.Inherit(context.Background(), 1, 2)
	require.NoError(t, err)
}

// TestSeasonInheritService_Inherit_NoConfig 测试继承规则配置为空时返回nil不报错。
func TestSeasonInheritService_Inherit_NoConfig(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()

	svc := NewSeasonInheritService(cfg, logger)

	err := svc.Inherit(context.Background(), 1, 2)
	require.NoError(t, err)
}

// TestRewardDistributor_Distribute 测试奖励发放成功。
func TestRewardDistributor_Distribute(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.SeasonRewards = []config.RewardConfig{
		{RewardID: "1", MinRank: 1, MaxRank: 10, Resources: map[string]int64{"gold": 1000}},
	}
	logger := zap.NewNop()
	outbox := &mockOutbox{}

	svc := NewRewardDistributor(cfg, outbox, logger)

	err := svc.Distribute(context.Background(), 1, []int64{101, 102, 103})
	require.NoError(t, err)
}

// TestRewardDistributor_Distribute_NoConfig 测试奖励配置为空时返回nil不报错。
func TestRewardDistributor_Distribute_NoConfig(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	outbox := &mockOutbox{}

	svc := NewRewardDistributor(cfg, outbox, logger)

	err := svc.Distribute(context.Background(), 1, []int64{101, 102})
	require.NoError(t, err)
}

// TestRewardDistributor_Distribute_EmptyRankings 测试空排行榜时发放成功。
func TestRewardDistributor_Distribute_EmptyRankings(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.SeasonRewards = []config.RewardConfig{
		{RewardID: "1", MinRank: 1, MaxRank: 10},
	}
	logger := zap.NewNop()
	outbox := &mockOutbox{}

	svc := NewRewardDistributor(cfg, outbox, logger)

	err := svc.Distribute(context.Background(), 1, []int64{})
	require.NoError(t, err)
}
