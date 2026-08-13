// Package command 编排玩家创建、建筑建造与单位训练写侧用例。
package command

import (
	"context"
	stderrors "errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"go.uber.org/zap"

	"insectworld/server/game/domain/building"
	"insectworld/server/game/domain/catalog"
	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/game/domain/identity"
	"insectworld/server/game/domain/player"
	"insectworld/server/game/domain/resource"
	"insectworld/server/game/domain/training"
)

// Service 编排Growth上下文首个可玩纵切的写命令。
type Service struct {
	players   player.Repository   // 玩家档案仓储
	buildings building.Repository // 玩家建筑仓储
	trainings training.Repository // 训练任务仓储
	roster    training.Roster     // 已训练单位名册
	catalog   catalog.Reader      // 绑定版本的游戏包目录
	resources resource.Account    // Economy资源账户防腐层
	ids       identity.Generator  // 聚合实例ID生成器
	logger    *zap.Logger         // 结构化业务日志器
}

// NewService 创建Growth写命令服务。
func NewService(players player.Repository, buildings building.Repository, trainings training.Repository, roster training.Roster, catalogReader catalog.Reader, resources resource.Account, ids identity.Generator, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{players: players, buildings: buildings, trainings: trainings, roster: roster, catalog: catalogReader, resources: resources, ids: ids, logger: logger}
}

// CreatePlayerCommand 是创建玩家并发放初始资源的幂等命令。
type CreatePlayerCommand struct {
	CommandID string // CommandID 是调用方生成的全局幂等键
	PlayerID  int64  // PlayerID 是注册链路分配的玩家ID
	Nickname  string // Nickname 是玩家昵称
	FactionID string // FactionID 是阵营稳定ID；为空时使用游戏包默认阵营
	NowMs     int64  // NowMs 是命令发生时间，Unix毫秒
}

// CreatePlayer 创建玩家档案并通过资源端口幂等发放初始资源。
func (s *Service) CreatePlayer(ctx context.Context, cmd CreatePlayerCommand) (*player.Profile, error) {
	if cmd.CommandID == "" {
		return nil, fmt.Errorf("创建玩家缺少幂等键: %w", gameerr.ErrInvalidCommand)
	}
	factionID := cmd.FactionID
	var faction catalog.FactionDefinition
	var err error
	if factionID == "" {
		faction, err = s.catalog.DefaultFaction(ctx)
	} else {
		faction, err = s.catalog.Faction(ctx, factionID)
	}
	if err != nil {
		return nil, fmt.Errorf("读取玩家阵营配置失败，factionID=%s: %w", factionID, err)
	}
	if existing, findErr := s.players.FindByCommandID(ctx, cmd.CommandID); findErr == nil {
		if existing.PlayerID() != cmd.PlayerID || existing.FactionID() != faction.ID || existing.Nickname() != strings.TrimSpace(cmd.Nickname) {
			return nil, fmt.Errorf("创建玩家幂等键载荷冲突，commandID=%s: %w", cmd.CommandID, gameerr.ErrStateConflict)
		}
		return existing, nil
	} else if !stderrors.Is(findErr, gameerr.ErrPlayerNotFound) {
		return nil, fmt.Errorf("按命令查询玩家失败，commandID=%s: %w", cmd.CommandID, findErr)
	}

	profile, err := player.NewProfile(cmd.PlayerID, faction.ID, cmd.Nickname, cmd.NowMs, cmd.CommandID)
	if err != nil {
		return nil, err
	}
	operationID := "create-player:" + cmd.CommandID
	if err := s.resources.Change(ctx, cmd.PlayerID, faction.StartingResources, operationID); err != nil {
		return nil, fmt.Errorf("发放玩家初始资源失败，playerID=%d: %w", cmd.PlayerID, err)
	}
	saved, _, err := s.players.SaveIfAbsent(ctx, profile)
	if err != nil {
		if reverseErr := s.resources.Reverse(ctx, operationID); reverseErr != nil {
			s.logger.Error("创建玩家回滚初始资源失败", zap.String("request_id", cmd.CommandID), zap.Int64("player_id", cmd.PlayerID), zap.Error(reverseErr))
		}
		return nil, fmt.Errorf("保存玩家档案失败，playerID=%d: %w", cmd.PlayerID, err)
	}
	s.logger.Info("创建玩家完成", zap.String("request_id", cmd.CommandID), zap.Int64("player_id", cmd.PlayerID), zap.String("faction_id", faction.ID), zap.String("result", "success"))
	return saved, nil
}

// ConstructBuildingCommand 是开始建造建筑的幂等命令。
type ConstructBuildingCommand struct {
	CommandID      string // CommandID 是调用方生成的全局幂等键
	PlayerID       int64  // PlayerID 是发起建造的玩家ID
	BuildingTypeID string // BuildingTypeID 是游戏包中的建筑稳定ID
	NowMs          int64  // NowMs 是建造开始时间，Unix毫秒
}

// ConstructBuilding 校验阵营与资源后创建建造任务。
func (s *Service) ConstructBuilding(ctx context.Context, cmd ConstructBuildingCommand) (*building.Building, error) {
	if cmd.CommandID == "" || cmd.PlayerID <= 0 || cmd.BuildingTypeID == "" || cmd.NowMs <= 0 {
		return nil, fmt.Errorf("建造命令参数非法，playerID=%d: %w", cmd.PlayerID, gameerr.ErrInvalidCommand)
	}
	if existing, err := s.buildings.FindByCommandID(ctx, cmd.CommandID); err == nil {
		if existing.PlayerID() != cmd.PlayerID || existing.TypeID() != cmd.BuildingTypeID {
			return nil, fmt.Errorf("建造幂等键载荷冲突，commandID=%s: %w", cmd.CommandID, gameerr.ErrStateConflict)
		}
		return existing, nil
	} else if !stderrors.Is(err, gameerr.ErrDefinitionNotFound) {
		return nil, fmt.Errorf("按命令查询建筑失败，commandID=%s: %w", cmd.CommandID, err)
	}
	profile, err := s.players.FindByPlayerID(ctx, cmd.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("加载建造玩家失败，playerID=%d: %w", cmd.PlayerID, err)
	}
	definition, err := s.catalog.Building(ctx, cmd.BuildingTypeID)
	if err != nil {
		return nil, err
	}
	if definition.FactionID != profile.FactionID() {
		return nil, fmt.Errorf("建筑阵营与玩家不匹配，buildingTypeID=%s: %w", cmd.BuildingTypeID, gameerr.ErrFactionMismatch)
	}
	completeAt, err := safeAdd(cmd.NowMs, definition.BuildTimeMs)
	if err != nil {
		return nil, err
	}
	aggregate, err := building.NewConstruction(s.ids.Next(), cmd.PlayerID, definition.ID, cmd.NowMs, completeAt, cmd.CommandID)
	if err != nil {
		return nil, err
	}
	operationID := "construct-building:" + cmd.CommandID
	if err := s.resources.Change(ctx, cmd.PlayerID, negate(definition.BuildCost), operationID); err != nil {
		return nil, fmt.Errorf("扣除建造资源失败，playerID=%d，buildingTypeID=%s: %w", cmd.PlayerID, definition.ID, err)
	}
	saved, _, err := s.buildings.SaveIfAbsent(ctx, aggregate)
	if err != nil {
		if reverseErr := s.resources.Reverse(ctx, operationID); reverseErr != nil {
			s.logger.Error("建造失败回滚资源异常", zap.String("request_id", cmd.CommandID), zap.Int64("player_id", cmd.PlayerID), zap.Error(reverseErr))
		}
		return nil, fmt.Errorf("保存建筑失败，buildingID=%d: %w", aggregate.ID(), err)
	}
	s.logger.Info("开始建造建筑", zap.String("request_id", cmd.CommandID), zap.Int64("player_id", cmd.PlayerID), zap.Int64("building_id", saved.ID()), zap.String("building_type_id", saved.TypeID()), zap.String("result", "success"))
	return saved, nil
}

// CompleteBuilding 完成已达到结束时间的建筑建造。
func (s *Service) CompleteBuilding(ctx context.Context, playerID int64, buildingID int64, nowMs int64, requestID string) (*building.Building, error) {
	if requestID == "" || playerID <= 0 || buildingID <= 0 || nowMs <= 0 {
		return nil, fmt.Errorf("完成建筑命令参数非法，buildingID=%d: %w", buildingID, gameerr.ErrInvalidCommand)
	}
	aggregate, err := s.buildings.FindByID(ctx, buildingID)
	if err != nil {
		return nil, fmt.Errorf("加载建筑失败，buildingID=%d: %w", buildingID, err)
	}
	if aggregate.PlayerID() != playerID {
		return nil, fmt.Errorf("建筑不属于当前玩家，buildingID=%d: %w", buildingID, gameerr.ErrStateConflict)
	}
	if err := aggregate.Complete(nowMs); err != nil {
		return nil, err
	}
	if err := s.buildings.Save(ctx, aggregate); err != nil {
		return nil, fmt.Errorf("保存建筑完成状态失败，buildingID=%d: %w", buildingID, err)
	}
	s.logger.Info("建筑建造完成", zap.String("request_id", requestID), zap.Int64("player_id", playerID), zap.Int64("building_id", buildingID), zap.String("result", "success"))
	return aggregate, nil
}

// StartTrainingCommand 是开始单位训练的幂等命令。
type StartTrainingCommand struct {
	CommandID  string // CommandID 是调用方生成的全局幂等键
	PlayerID   int64  // PlayerID 是发起训练的玩家ID
	BuildingID int64  // BuildingID 是执行训练的建筑实例ID
	UnitTypeID string // UnitTypeID 是游戏包中的单位稳定ID
	Count      int64  // Count 是训练数量，必须大于0
	NowMs      int64  // NowMs 是训练开始时间，Unix毫秒
}

// StartTraining 校验建筑能力、阵营和资源后创建训练任务。
func (s *Service) StartTraining(ctx context.Context, cmd StartTrainingCommand) (*training.Task, error) {
	if cmd.CommandID == "" || cmd.PlayerID <= 0 || cmd.BuildingID <= 0 || cmd.UnitTypeID == "" || cmd.Count <= 0 || cmd.NowMs <= 0 {
		return nil, fmt.Errorf("训练命令参数非法，playerID=%d: %w", cmd.PlayerID, gameerr.ErrInvalidCommand)
	}
	if existing, err := s.trainings.FindByCommandID(ctx, cmd.CommandID); err == nil {
		if existing.PlayerID() != cmd.PlayerID || existing.BuildingID() != cmd.BuildingID || existing.UnitTypeID() != cmd.UnitTypeID || existing.Count() != cmd.Count {
			return nil, fmt.Errorf("训练幂等键载荷冲突，commandID=%s: %w", cmd.CommandID, gameerr.ErrStateConflict)
		}
		return existing, nil
	} else if !stderrors.Is(err, gameerr.ErrDefinitionNotFound) {
		return nil, fmt.Errorf("按命令查询训练任务失败，commandID=%s: %w", cmd.CommandID, err)
	}
	profile, err := s.players.FindByPlayerID(ctx, cmd.PlayerID)
	if err != nil {
		return nil, err
	}
	buildingAggregate, err := s.buildings.FindByID(ctx, cmd.BuildingID)
	if err != nil {
		return nil, err
	}
	if buildingAggregate.PlayerID() != cmd.PlayerID || buildingAggregate.Status() != building.StatusOperational {
		return nil, fmt.Errorf("训练建筑不可用，buildingID=%d: %w", cmd.BuildingID, gameerr.ErrBuildingNotReady)
	}
	buildingDefinition, err := s.catalog.Building(ctx, buildingAggregate.TypeID())
	if err != nil {
		return nil, err
	}
	unitDefinition, err := s.catalog.Unit(ctx, cmd.UnitTypeID)
	if err != nil {
		return nil, err
	}
	if unitDefinition.FactionID != profile.FactionID() {
		return nil, fmt.Errorf("单位阵营与玩家不匹配，unitTypeID=%s: %w", cmd.UnitTypeID, gameerr.ErrFactionMismatch)
	}
	if !slices.Contains(buildingDefinition.TrainableUnitIDs, cmd.UnitTypeID) {
		return nil, fmt.Errorf("建筑不能训练目标单位，buildingTypeID=%s，unitTypeID=%s: %w", buildingDefinition.ID, cmd.UnitTypeID, gameerr.ErrUnitNotTrainable)
	}
	cost, err := scaleAndNegate(unitDefinition.TrainCost, cmd.Count)
	if err != nil {
		return nil, err
	}
	duration, err := safeMultiply(unitDefinition.TrainTimeMs, cmd.Count)
	if err != nil {
		return nil, err
	}
	completeAt, err := safeAdd(cmd.NowMs, duration)
	if err != nil {
		return nil, err
	}
	aggregate, err := training.NewTask(s.ids.Next(), cmd.PlayerID, cmd.BuildingID, unitDefinition.ID, cmd.Count, cmd.NowMs, completeAt, cmd.CommandID)
	if err != nil {
		return nil, err
	}
	operationID := "start-training:" + cmd.CommandID
	if err := s.resources.Change(ctx, cmd.PlayerID, cost, operationID); err != nil {
		return nil, fmt.Errorf("扣除训练资源失败，playerID=%d，unitTypeID=%s: %w", cmd.PlayerID, cmd.UnitTypeID, err)
	}
	saved, _, err := s.trainings.SaveIfAbsent(ctx, aggregate)
	if err != nil {
		if reverseErr := s.resources.Reverse(ctx, operationID); reverseErr != nil {
			s.logger.Error("训练失败回滚资源异常", zap.String("request_id", cmd.CommandID), zap.Int64("player_id", cmd.PlayerID), zap.Error(reverseErr))
		}
		return nil, fmt.Errorf("保存训练任务失败，taskID=%d: %w", aggregate.ID(), err)
	}
	s.logger.Info("开始训练单位", zap.String("request_id", cmd.CommandID), zap.Int64("player_id", cmd.PlayerID), zap.Int64("training_id", saved.ID()), zap.String("unit_type_id", saved.UnitTypeID()), zap.Int64("count", saved.Count()), zap.String("result", "success"))
	return saved, nil
}

// CompleteTraining 完成已达到结束时间的训练任务。
func (s *Service) CompleteTraining(ctx context.Context, playerID int64, taskID int64, nowMs int64, requestID string) (*training.Task, error) {
	if requestID == "" || playerID <= 0 || taskID <= 0 || nowMs <= 0 {
		return nil, fmt.Errorf("完成训练命令参数非法，taskID=%d: %w", taskID, gameerr.ErrInvalidCommand)
	}
	aggregate, err := s.trainings.FindByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("加载训练任务失败，taskID=%d: %w", taskID, err)
	}
	if aggregate.PlayerID() != playerID {
		return nil, fmt.Errorf("训练任务不属于当前玩家，taskID=%d: %w", taskID, gameerr.ErrStateConflict)
	}
	if err := aggregate.Complete(nowMs); err != nil {
		return nil, err
	}
	operationID := "complete-training:" + aggregate.CommandID()
	if err := s.roster.Grant(ctx, playerID, aggregate.UnitTypeID(), aggregate.Count(), operationID); err != nil {
		return nil, fmt.Errorf("发放已训练单位失败，taskID=%d: %w", taskID, err)
	}
	if err := s.trainings.Save(ctx, aggregate); err != nil {
		return nil, fmt.Errorf("保存训练完成状态失败，taskID=%d: %w", taskID, err)
	}
	s.logger.Info("单位训练完成", zap.String("request_id", requestID), zap.Int64("player_id", playerID), zap.Int64("training_id", taskID), zap.String("result", "success"))
	return aggregate, nil
}

func negate(amounts map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(amounts))
	for resourceID, amount := range amounts {
		result[resourceID] = -amount
	}
	return result
}

func scaleAndNegate(amounts map[string]int64, count int64) (map[string]int64, error) {
	result := make(map[string]int64, len(amounts))
	for resourceID, amount := range amounts {
		total, err := safeMultiply(amount, count)
		if err != nil {
			return nil, fmt.Errorf("训练资源数量溢出，resourceID=%s: %w", resourceID, err)
		}
		result[resourceID] = -total
	}
	return result, nil
}

func safeMultiply(left int64, right int64) (int64, error) {
	if left <= 0 || right <= 0 || left > math.MaxInt64/right {
		return 0, fmt.Errorf("正整数乘法越界，left=%d，right=%d: %w", left, right, gameerr.ErrInvalidCommand)
	}
	return left * right, nil
}

func safeAdd(left int64, right int64) (int64, error) {
	if left <= 0 || right <= 0 || left > math.MaxInt64-right {
		return 0, fmt.Errorf("时间戳加法越界，left=%d，right=%d: %w", left, right, gameerr.ErrInvalidCommand)
	}
	return left + right, nil
}
