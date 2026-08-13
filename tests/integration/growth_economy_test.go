// Package integration 跨服务集成测试，验证模块化单体内限界上下文协作。
package integration

import (
	"context"
	"errors"
	"maps"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	economyapp "insectworld/server/economy/application/resourceaccount"
	economyerr "insectworld/server/economy/domain/errors"
	economydomain "insectworld/server/economy/domain/resourceaccount"
	gamecommand "insectworld/server/game/application/command"
	gamecatalog "insectworld/server/game/infrastructure/catalog"
	"insectworld/server/game/infrastructure/memory"
	"insectworld/server/shared/pkg/gamepack"
)

type economyRepository struct {
	balances   map[int64]map[string]int64      // 玩家资源余额
	operations map[string]economydomain.Change // 已应用操作账本
}

func newEconomyRepository() *economyRepository {
	return &economyRepository{balances: make(map[int64]map[string]int64), operations: make(map[string]economydomain.Change)}
}

func (r *economyRepository) Apply(_ context.Context, change economydomain.Change) error {
	if existing, ok := r.operations[change.OperationID]; ok {
		if existing.PlayerID != change.PlayerID || !maps.Equal(existing.Amounts, change.Amounts) {
			return economyerr.ErrOperationConflict
		}
		return nil
	}
	current := r.balances[change.PlayerID]
	if current == nil {
		current = make(map[string]int64)
	}
	for resourceID, amount := range change.Amounts {
		if current[resourceID]+amount < 0 {
			return economyerr.ErrResourceInsufficient
		}
	}
	for resourceID, amount := range change.Amounts {
		current[resourceID] += amount
	}
	r.balances[change.PlayerID] = current
	change.Amounts = maps.Clone(change.Amounts)
	r.operations[change.OperationID] = change
	return nil
}

func (r *economyRepository) Reverse(_ context.Context, operationID string, _ int64) error {
	change, ok := r.operations[operationID]
	if !ok {
		return nil
	}
	for resourceID, amount := range change.Amounts {
		r.balances[change.PlayerID][resourceID] -= amount
	}
	delete(r.operations, operationID)
	return nil
}

func (r *economyRepository) Balances(_ context.Context, playerID int64) (map[string]int64, error) {
	return maps.Clone(r.balances[playerID]), nil
}

// TestGrowthUsesEconomyResourceAccount 验证Growth通过Economy应用API完成初始资源、建造和训练扣款。
func TestGrowthUsesEconomyResourceAccount(t *testing.T) {
	t.Parallel()
	pack, err := gamepack.LoadAndCompile(filepath.Join("..", "..", "gamepacks", "insect-world"), "0.1.0")
	require.NoError(t, err)
	reader, err := gamecatalog.NewGamePackReader(pack)
	require.NoError(t, err)
	economyRepository := newEconomyRepository()
	economyService := economyapp.NewService(economyRepository)
	growth := gamecommand.NewService(memory.NewPlayerRepository(), memory.NewBuildingRepository(), memory.NewTrainingRepository(), memory.NewUnitRoster(), reader, economyService, memory.NewIDGenerator(100), zap.NewNop())
	ctx := context.Background()

	_, err = growth.CreatePlayer(ctx, gamecommand.CreatePlayerCommand{CommandID: "create-economy", PlayerID: 101, Nickname: "经济玩家", NowMs: 1000})
	require.NoError(t, err)
	building, err := growth.ConstructBuilding(ctx, gamecommand.ConstructBuildingCommand{CommandID: "build-economy", PlayerID: 101, BuildingTypeID: "brood-chamber", NowMs: 2000})
	require.NoError(t, err)
	_, err = growth.CompleteBuilding(ctx, 101, building.ID(), 4000, "complete-building")
	require.NoError(t, err)
	_, err = growth.StartTraining(ctx, gamecommand.StartTrainingCommand{CommandID: "train-economy", PlayerID: 101, BuildingID: building.ID(), UnitTypeID: "worker-ant", Count: 2, NowMs: 5000})
	require.NoError(t, err)
	balances, err := economyService.Balances(ctx, 101)
	require.NoError(t, err)
	assert.Equal(t, int64(55), balances["food"])

	_, err = growth.StartTraining(ctx, gamecommand.StartTrainingCommand{CommandID: "train-insufficient", PlayerID: 101, BuildingID: building.ID(), UnitTypeID: "worker-ant", Count: 6, NowMs: 6000})
	require.Error(t, err)
	assert.True(t, errors.Is(err, economyerr.ErrResourceInsufficient))
}
