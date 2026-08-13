// Package command 编排玩家创建、建筑建造与单位训练写侧用例。
package command

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/game/domain/building"
	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/game/domain/player"
	"insectworld/server/game/domain/training"
	gamecatalog "insectworld/server/game/infrastructure/catalog"
	"insectworld/server/game/infrastructure/memory"
	"insectworld/server/shared/pkg/gamepack"
)

const testEngineVersion = "0.1.0"

type growthFixture struct {
	service   *Service                // 待测Growth应用服务
	resources *memory.ResourceAccount // 用于验证跨上下文资源副作用的内存适配器
	roster    *memory.UnitRoster      // 用于验证训练完成后的单位入账
}

type failingPlayerRepository struct{}

func (f failingPlayerRepository) FindByPlayerID(_ context.Context, _ int64) (*player.Profile, error) {
	return nil, gameerr.ErrPlayerNotFound
}

func (f failingPlayerRepository) FindByCommandID(_ context.Context, _ string) (*player.Profile, error) {
	return nil, gameerr.ErrPlayerNotFound
}

func (f failingPlayerRepository) SaveIfAbsent(_ context.Context, _ *player.Profile) (*player.Profile, bool, error) {
	return nil, false, fmt.Errorf("模拟持久化失败")
}

// TestGrowthRetryAndFailureBranches 验证纵切关键重试、权限和状态失败语义。
func TestGrowthRetryAndFailureBranches(t *testing.T) {
	t.Parallel()
	fixture := newGrowthFixture(t)
	ctx := context.Background()
	createCommand := CreatePlayerCommand{CommandID: "create-branches", PlayerID: 11, Nickname: "分支玩家", NowMs: 1000}
	firstProfile, err := fixture.service.CreatePlayer(ctx, createCommand)
	require.NoError(t, err)
	retriedProfile, err := fixture.service.CreatePlayer(ctx, createCommand)
	require.NoError(t, err)
	assert.Equal(t, firstProfile.PlayerID(), retriedProfile.PlayerID())

	buildCommand := ConstructBuildingCommand{CommandID: "build-branches", PlayerID: 11, BuildingTypeID: "brood-chamber", NowMs: 2000}
	firstBuilding, err := fixture.service.ConstructBuilding(ctx, buildCommand)
	require.NoError(t, err)
	retriedBuilding, err := fixture.service.ConstructBuilding(ctx, buildCommand)
	require.NoError(t, err)
	assert.Equal(t, firstBuilding.ID(), retriedBuilding.ID())
	_, err = fixture.service.ConstructBuilding(ctx, ConstructBuildingCommand{CommandID: buildCommand.CommandID, PlayerID: 12, BuildingTypeID: buildCommand.BuildingTypeID, NowMs: buildCommand.NowMs})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrStateConflict))

	_, err = fixture.service.StartTraining(ctx, StartTrainingCommand{CommandID: "train-too-early", PlayerID: 11, BuildingID: firstBuilding.ID(), UnitTypeID: "worker-ant", Count: 1, NowMs: 3000})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrBuildingNotReady))
	_, err = fixture.service.CompleteBuilding(ctx, 12, firstBuilding.ID(), 4000, "wrong-owner")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrStateConflict))
	_, err = fixture.service.CompleteBuilding(ctx, 11, firstBuilding.ID(), 4000, "complete-branches")
	require.NoError(t, err)

	trainCommand := StartTrainingCommand{CommandID: "train-branches", PlayerID: 11, BuildingID: firstBuilding.ID(), UnitTypeID: "worker-ant", Count: 1, NowMs: 5000}
	firstTask, err := fixture.service.StartTraining(ctx, trainCommand)
	require.NoError(t, err)
	retriedTask, err := fixture.service.StartTraining(ctx, trainCommand)
	require.NoError(t, err)
	assert.Equal(t, firstTask.ID(), retriedTask.ID())
	changedCommand := trainCommand
	changedCommand.Count = 2
	_, err = fixture.service.StartTraining(ctx, changedCommand)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrStateConflict))
	_, err = fixture.service.CompleteTraining(ctx, 11, firstTask.ID(), 5999, "train-early")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrTaskNotReady))
	_, err = fixture.service.CompleteTraining(ctx, 12, firstTask.ID(), 6000, "train-wrong-owner")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrStateConflict))
}

// TestGrowthRejectsInvalidCommands 验证所有写入口拒绝缺失幂等键或非法标识。
func TestGrowthRejectsInvalidCommands(t *testing.T) {
	t.Parallel()
	fixture := newGrowthFixture(t)
	ctx := context.Background()
	_, err := fixture.service.CreatePlayer(ctx, CreatePlayerCommand{})
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
	_, err = fixture.service.ConstructBuilding(ctx, ConstructBuildingCommand{})
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
	_, err = fixture.service.StartTraining(ctx, StartTrainingCommand{})
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
	_, err = fixture.service.CompleteBuilding(ctx, 0, 0, 0, "")
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
	_, err = fixture.service.CompleteTraining(ctx, 0, 0, 0, "")
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
	serviceWithDefaultLogger := NewService(nil, nil, nil, nil, nil, nil, nil, nil)
	assert.NotNil(t, serviceWithDefaultLogger.logger)
}

// TestGrowthRejectsMissingDefinitions 验证玩家、建筑和单位定义缺失时返回稳定错误。
func TestGrowthRejectsMissingDefinitions(t *testing.T) {
	t.Parallel()
	fixture := newGrowthFixture(t)
	ctx := context.Background()
	_, err := fixture.service.CreatePlayer(ctx, CreatePlayerCommand{CommandID: "missing-faction", PlayerID: 20, Nickname: "玩家", FactionID: "unknown", NowMs: 1000})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrDefinitionNotFound))
	_, err = fixture.service.ConstructBuilding(ctx, ConstructBuildingCommand{CommandID: "missing-player", PlayerID: 20, BuildingTypeID: "brood-chamber", NowMs: 1000})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrPlayerNotFound))
	_, err = fixture.service.CompleteBuilding(ctx, 20, 9999, 1000, "missing-building-instance")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrBuildingNotFound))
	_, err = fixture.service.CompleteTraining(ctx, 20, 9999, 1000, "missing-training-instance")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrTrainingNotFound))
	_, err = fixture.service.CreatePlayer(ctx, CreatePlayerCommand{CommandID: "explicit-faction", PlayerID: 20, Nickname: "玩家", FactionID: "ant-colony", NowMs: 1000})
	require.NoError(t, err)
	_, err = fixture.service.ConstructBuilding(ctx, ConstructBuildingCommand{CommandID: "missing-building", PlayerID: 20, BuildingTypeID: "unknown", NowMs: 2000})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrDefinitionNotFound))
	constructed, err := fixture.service.ConstructBuilding(ctx, ConstructBuildingCommand{CommandID: "known-building", PlayerID: 20, BuildingTypeID: "brood-chamber", NowMs: 2000})
	require.NoError(t, err)
	_, err = fixture.service.CompleteBuilding(ctx, 20, constructed.ID(), 4000, "complete-known")
	require.NoError(t, err)
	_, err = fixture.service.StartTraining(ctx, StartTrainingCommand{CommandID: "missing-unit", PlayerID: 20, BuildingID: constructed.ID(), UnitTypeID: "unknown", Count: 1, NowMs: 5000})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrDefinitionNotFound))
}

// TestGrowthArithmeticBoundaries 验证配置成本和时间计算拒绝整数溢出。
func TestGrowthArithmeticBoundaries(t *testing.T) {
	t.Parallel()
	_, err := safeMultiply(math.MaxInt64, 2)
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
	_, err = safeAdd(math.MaxInt64, 1)
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
	_, err = scaleAndNegate(map[string]int64{"resource": math.MaxInt64}, 2)
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
}

// TestCreatePlayerReversesResourcesWhenPersistenceFails 验证档案保存失败时初始资源会被补偿撤销。
func TestCreatePlayerReversesResourcesWhenPersistenceFails(t *testing.T) {
	t.Parallel()
	baseFixture := newGrowthFixture(t)
	reader := baseFixture.service.catalog
	resources := memory.NewResourceAccount()
	service := NewService(failingPlayerRepository{}, memory.NewBuildingRepository(), memory.NewTrainingRepository(), memory.NewUnitRoster(), reader, resources, memory.NewIDGenerator(1), zap.NewNop())
	_, err := service.CreatePlayer(context.Background(), CreatePlayerCommand{CommandID: "create-save-failure", PlayerID: 30, Nickname: "失败玩家", NowMs: 1000})
	require.Error(t, err)
	balances, balanceErr := resources.Balances(context.Background(), 30)
	require.NoError(t, balanceErr)
	assert.Zero(t, balances["food"])
}

func newGrowthFixture(t *testing.T) growthFixture {
	t.Helper()
	packRoot := filepath.Join("..", "..", "..", "..", "gamepacks", "insect-world")
	pack, err := gamepack.LoadAndCompile(packRoot, testEngineVersion)
	require.NoError(t, err)
	reader, err := gamecatalog.NewGamePackReader(pack)
	require.NoError(t, err)
	resources := memory.NewResourceAccount()
	roster := memory.NewUnitRoster()
	service := NewService(memory.NewPlayerRepository(), memory.NewBuildingRepository(), memory.NewTrainingRepository(), roster, reader, resources, memory.NewIDGenerator(1000), zap.NewNop())
	return growthFixture{service: service, resources: resources, roster: roster}
}

// TestGrowthVerticalSlice 验证游戏包驱动的创建玩家、建造和训练完整纵切。
func TestGrowthVerticalSlice(t *testing.T) {
	t.Parallel()
	fixture := newGrowthFixture(t)
	ctx := context.Background()

	profile, err := fixture.service.CreatePlayer(ctx, CreatePlayerCommand{CommandID: "create-1", PlayerID: 7, Nickname: "探索者", NowMs: 1000})
	require.NoError(t, err)
	assert.Equal(t, "ant-colony", profile.FactionID())
	balances, err := fixture.resources.Balances(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balances["food"])

	constructed, err := fixture.service.ConstructBuilding(ctx, ConstructBuildingCommand{CommandID: "build-1", PlayerID: 7, BuildingTypeID: "brood-chamber", NowMs: 2000})
	require.NoError(t, err)
	assert.Equal(t, building.StatusConstructing, constructed.Status())
	_, err = fixture.service.CompleteBuilding(ctx, 7, constructed.ID(), 3999, "complete-build-early")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrTaskNotReady))
	constructed, err = fixture.service.CompleteBuilding(ctx, 7, constructed.ID(), 4000, "complete-build-1")
	require.NoError(t, err)
	assert.Equal(t, building.StatusOperational, constructed.Status())

	task, err := fixture.service.StartTraining(ctx, StartTrainingCommand{CommandID: "train-1", PlayerID: 7, BuildingID: constructed.ID(), UnitTypeID: "worker-ant", Count: 2, NowMs: 5000})
	require.NoError(t, err)
	assert.Equal(t, int64(7000), task.CompleteAt())
	task, err = fixture.service.CompleteTraining(ctx, 7, task.ID(), 7000, "complete-train-1")
	require.NoError(t, err)
	assert.Equal(t, training.StatusComplete, task.Status())
	unitCount, err := fixture.roster.Count(ctx, 7, "worker-ant")
	require.NoError(t, err)
	assert.Equal(t, int64(2), unitCount)
	_, err = fixture.service.CompleteTraining(ctx, 7, task.ID(), 7001, "complete-train-retry")
	require.NoError(t, err)
	unitCount, err = fixture.roster.Count(ctx, 7, "worker-ant")
	require.NoError(t, err)
	assert.Equal(t, int64(2), unitCount)

	balances, err = fixture.resources.Balances(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, int64(55), balances["food"])
}

// TestConstructBuildingConcurrentRetryChargesOnce 验证并发重复建造命令只创建一个建筑并扣款一次。
func TestConstructBuildingConcurrentRetryChargesOnce(t *testing.T) {
	t.Parallel()
	fixture := newGrowthFixture(t)
	ctx := context.Background()
	_, err := fixture.service.CreatePlayer(ctx, CreatePlayerCommand{CommandID: "create-concurrent", PlayerID: 8, Nickname: "并发玩家", NowMs: 1000})
	require.NoError(t, err)

	const workers = 16
	var waitGroup sync.WaitGroup
	results := make(chan int64, workers)
	errorsFound := make(chan error, workers)
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			value, commandErr := fixture.service.ConstructBuilding(ctx, ConstructBuildingCommand{CommandID: "build-concurrent", PlayerID: 8, BuildingTypeID: "brood-chamber", NowMs: 2000})
			if commandErr != nil {
				errorsFound <- commandErr
				return
			}
			results <- value.ID()
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errorsFound)
	for commandErr := range errorsFound {
		require.NoError(t, commandErr)
	}
	var firstID int64
	for id := range results {
		if firstID == 0 {
			firstID = id
		}
		assert.Equal(t, firstID, id)
	}
	balances, err := fixture.resources.Balances(ctx, 8)
	require.NoError(t, err)
	assert.Equal(t, int64(75), balances["food"])
}

// TestIdempotencyKeyRejectsDifferentPayload 验证相同幂等键不能复用于不同业务载荷。
func TestIdempotencyKeyRejectsDifferentPayload(t *testing.T) {
	t.Parallel()
	fixture := newGrowthFixture(t)
	ctx := context.Background()
	_, err := fixture.service.CreatePlayer(ctx, CreatePlayerCommand{CommandID: "create-conflict", PlayerID: 9, Nickname: "原昵称", NowMs: 1000})
	require.NoError(t, err)
	_, err = fixture.service.CreatePlayer(ctx, CreatePlayerCommand{CommandID: "create-conflict", PlayerID: 9, Nickname: "新昵称", NowMs: 1001})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrStateConflict))
}

// TestStartTrainingRejectsInsufficientResources 验证资源不足时不创建训练任务且余额不变。
func TestStartTrainingRejectsInsufficientResources(t *testing.T) {
	t.Parallel()
	fixture := newGrowthFixture(t)
	ctx := context.Background()
	_, err := fixture.service.CreatePlayer(ctx, CreatePlayerCommand{CommandID: "create-poor", PlayerID: 10, Nickname: "资源玩家", NowMs: 1000})
	require.NoError(t, err)
	constructed, err := fixture.service.ConstructBuilding(ctx, ConstructBuildingCommand{CommandID: "build-poor", PlayerID: 10, BuildingTypeID: "brood-chamber", NowMs: 2000})
	require.NoError(t, err)
	_, err = fixture.service.CompleteBuilding(ctx, 10, constructed.ID(), 4000, "complete-poor")
	require.NoError(t, err)
	_, err = fixture.service.StartTraining(ctx, StartTrainingCommand{CommandID: "train-too-many", PlayerID: 10, BuildingID: constructed.ID(), UnitTypeID: "worker-ant", Count: 8, NowMs: 5000})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrResourceInsufficient))
	balances, balanceErr := fixture.resources.Balances(ctx, 10)
	require.NoError(t, balanceErr)
	assert.Equal(t, int64(75), balances["food"])
}
