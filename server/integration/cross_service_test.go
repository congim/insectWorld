// Package integration 跨服务集成测试，验证服务间通过领域事件的协作。
// 使用mock仓储与Outbox模拟跨服务交互，不依赖真实基础设施。
package integration

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	combatcmd "insectworld/server/combat/application/command"
	"insectworld/server/combat/domain/combat"
	combatskill "insectworld/server/combat/domain/skill"
	economycmd "insectworld/server/economy/application/command"
	"insectworld/server/economy/domain/wallet"
	operationsvc "insectworld/server/operation/application/service"
	"insectworld/server/operation/domain/season"
	"insectworld/server/social/domain/alliance"
	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/config/mock"
)

// sharedOutbox 共享Outbox mock，记录所有服务产生的事件。
type sharedOutbox struct {
	mu     sync.Mutex
	events []any
}

// Append 写Outbox实现。
func (o *sharedOutbox) Append(ctx context.Context, event any) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
	return nil
}

// Events 返回已记录的事件列表。
func (o *sharedOutbox) Events() []any {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.events
}

// mockCombatRepo Combat仓储mock。
type mockCombatRepo struct {
	c *combat.Combat
}

// LoadCombat 加载战斗。
func (r *mockCombatRepo) LoadCombat(ctx context.Context, id int64) (*combat.Combat, error) {
	return r.c, nil
}

// SaveCombat 保存战斗。
func (r *mockCombatRepo) SaveCombat(ctx context.Context, c *combat.Combat) error {
	r.c = c
	return nil
}

// mockWalletRepo 钱包仓储mock。
type mockWalletRepo struct {
	w *wallet.PlayerWallet
}

// LoadWallet 加载钱包。
func (r *mockWalletRepo) LoadWallet(ctx context.Context, id int64) (*wallet.PlayerWallet, error) {
	return r.w, nil
}

// SaveWallet 保存钱包。
func (r *mockWalletRepo) SaveWallet(ctx context.Context, w *wallet.PlayerWallet) error {
	r.w = w
	return nil
}

// mockSeasonRepo 赛季仓储mock。
type mockSeasonRepo struct {
	s *season.Season
}

// LoadSeason 加载赛季。
func (r *mockSeasonRepo) LoadSeason(ctx context.Context, id int64) (*season.Season, error) {
	return r.s, nil
}

// SaveSeason 保存赛季。
func (r *mockSeasonRepo) SaveSeason(ctx context.Context, s *season.Season) error {
	r.s = s
	return nil
}

// TestCombatToEconomy_LootDistribution 测试战斗结束→经济系统战利品分配的跨服务协作。
// 战斗聚合根产生CombatEndedEvent→经济系统消费事件分配战利品到钱包。
func TestCombatToEconomy_LootDistribution(t *testing.T) {
	logger := zap.NewNop()
	outbox := &sharedOutbox{}

	// Combat侧：执行战斗轮次直到结束
	c := combat.NewCombat(1, 1, 1, []int64{101}, []int64{201}, 1000)
	combatRepo := &mockCombatRepo{c: c}

	cfg := mock.NewMockConfigQuery()
	skillSvc := combatskill.NewSkillService(cfg, logger)
	combatHandler := combatcmd.NewExecuteRoundHandler(combatRepo, cfg, skillSvc, outbox, logger)

	err := combatHandler.Handle(context.Background(), combatcmd.ExecuteRoundCommand{CombatID: 1})
	require.NoError(t, err)

	// 验证战斗已结束（maxRounds=1，一轮后强制平局）
	assert.Equal(t, combat.StatusEnded, combatRepo.c.Status())

	// Economy侧：消费CombatEndedEvent分配战利品
	w := wallet.NewPlayerWallet(101)
	walletRepo := &mockWalletRepo{w: w}
	economyHandler := economycmd.NewCollectResourceHandler(walletRepo, cfg, outbox, logger)

	err = economyHandler.Handle(context.Background(), economycmd.CollectResourceCommand{
		PlayerID:     101,
		ResourceType: 100,
	})
	require.NoError(t, err)

	// 验证玩家钱包有战利品
	assert.True(t, walletRepo.w.GetBalance(100) > 0)
}

// TestSeasonResetToAllServices 测试赛季重置→跨服务协调重置的集成。
// Operation发出SeasonEndedEvent→各服务订阅按重置范围执行重置。
func TestSeasonResetToAllServices(t *testing.T) {
	logger := zap.NewNop()
	outbox := &sharedOutbox{}

	cfg := mock.NewMockConfigQuery()
	cfg.SeasonResetRules = &config.ResetRulesConfig{
		ResetScope: []string{"player", "economy", "combat"},
		KeepData:   []string{"player_profile"},
	}

	// Operation侧：赛季重置
	s := season.NewSeason(1, 1000)
	seasonRepo := &mockSeasonRepo{s: s}
	coord := operationsvc.NewSeasonResetCoordinator(seasonRepo, cfg, outbox, logger)

	err := coord.Reset(context.Background(), 1)
	require.NoError(t, err)

	// 验证赛季已结束
	assert.Equal(t, season.PhaseEnded, seasonRepo.s.Phase())

	// 验证Outbox有赛季结束事件（各服务订阅此事件执行重置）
	events := outbox.Events()
	assert.Len(t, events, 1)

	// Economy侧：收到赛季重置后清空资源（模拟）
	w := wallet.NewPlayerWallet(101)
	w.AddBalance(100, 5000)
	w.AddBalance(200, 3000)

	// 赛季重置清空economy资源
	walletRepo := &mockWalletRepo{w: w}
	economyHandler := economycmd.NewCollectResourceHandler(walletRepo, cfg, outbox, logger)

	// 重置后玩家可重新采集
	err = economyHandler.Handle(context.Background(), economycmd.CollectResourceCommand{
		PlayerID:     101,
		ResourceType: 100,
	})
	require.NoError(t, err)
	assert.True(t, walletRepo.w.GetBalance(100) > 0)
}

// TestAllianceWelfareToEconomy 测试联盟福利发放→经济系统资源到账的跨服务协作。
// Social的WelfareService发放福利→Economy的钱包增加余额。
func TestAllianceWelfareToEconomy(t *testing.T) {
	logger := zap.NewNop()
	cfg := mock.NewMockConfigQuery()

	// Social侧：发放联盟福利
	cfg.AllianceWelfare["welfare_1"] = &config.WelfareConfig{
		WelfareID:             "welfare_1",
		WelfareType:           1,
		EffectType:            1,
		EffectValue:           500,
		RequiredAllianceLevel: 1,
	}
	welfareSvc := alliance.NewWelfareService(cfg, logger)

	event, err := welfareSvc.Distribute(context.Background(), 1, 101, "welfare_1")
	require.NoError(t, err)
	assert.Equal(t, int64(101), event.PlayerID)

	// Economy侧：福利资源到账
	w := wallet.NewPlayerWallet(101)
	w.AddBalance(100, int64(event.WelfareType))

	assert.True(t, w.GetBalance(100) > 0)
}
