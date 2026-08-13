// Package memory 提供Growth纵切本地运行与测试使用的并发安全内存适配器。
package memory

import (
	"context"
	"fmt"
	"sync"

	"insectworld/server/game/domain/building"
	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/game/domain/player"
	"insectworld/server/game/domain/training"
)

// PlayerRepository 是并发安全的内存玩家档案仓储。
type PlayerRepository struct {
	mu        sync.RWMutex              // 保护全部索引
	byID      map[int64]*player.Profile // 玩家ID索引
	byCommand map[string]int64          // 创建命令ID到玩家ID索引
}

// NewPlayerRepository 创建空的内存玩家档案仓储。
func NewPlayerRepository() *PlayerRepository {
	return &PlayerRepository{byID: make(map[int64]*player.Profile), byCommand: make(map[string]int64)}
}

// FindByPlayerID 按玩家ID读取档案副本。
func (r *PlayerRepository) FindByPlayerID(_ context.Context, playerID int64) (*player.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.byID[playerID]
	if !ok {
		return nil, fmt.Errorf("玩家不存在，playerID=%d: %w", playerID, gameerr.ErrPlayerNotFound)
	}
	return value.Clone(), nil
}

// FindByCommandID 按创建命令幂等键读取档案副本。
func (r *PlayerRepository) FindByCommandID(ctx context.Context, commandID string) (*player.Profile, error) {
	r.mu.RLock()
	playerID, ok := r.byCommand[commandID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("玩家创建命令不存在，commandID=%s: %w", commandID, gameerr.ErrPlayerNotFound)
	}
	return r.FindByPlayerID(ctx, playerID)
}

// SaveIfAbsent 原子保存玩家档案；重复命令返回既有档案且created为false。
func (r *PlayerRepository) SaveIfAbsent(_ context.Context, profile *player.Profile) (*player.Profile, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if playerID, ok := r.byCommand[profile.CommandID()]; ok {
		existing := r.byID[playerID]
		if existing.PlayerID() != profile.PlayerID() || existing.FactionID() != profile.FactionID() || existing.Nickname() != profile.Nickname() || existing.ConfigVersion() != profile.ConfigVersion() {
			return nil, false, fmt.Errorf("玩家创建命令载荷冲突，commandID=%s: %w", profile.CommandID(), gameerr.ErrStateConflict)
		}
		return existing.Clone(), false, nil
	}
	if _, ok := r.byID[profile.PlayerID()]; ok {
		return nil, false, fmt.Errorf("玩家ID已存在，playerID=%d: %w", profile.PlayerID(), gameerr.ErrPlayerAlreadyExists)
	}
	r.byID[profile.PlayerID()] = profile.Clone()
	r.byCommand[profile.CommandID()] = profile.PlayerID()
	return profile.Clone(), true, nil
}

// BuildingRepository 是并发安全的内存建筑仓储。
type BuildingRepository struct {
	mu        sync.RWMutex                 // 保护全部索引
	byID      map[int64]*building.Building // 建筑ID索引
	byCommand map[string]int64             // 建造命令ID到建筑ID索引
}

// NewBuildingRepository 创建空的内存建筑仓储。
func NewBuildingRepository() *BuildingRepository {
	return &BuildingRepository{byID: make(map[int64]*building.Building), byCommand: make(map[string]int64)}
}

// FindByID 按建筑ID读取聚合副本。
func (r *BuildingRepository) FindByID(_ context.Context, buildingID int64) (*building.Building, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.byID[buildingID]
	if !ok {
		return nil, fmt.Errorf("建筑不存在，buildingID=%d: %w", buildingID, gameerr.ErrBuildingNotFound)
	}
	return value.Clone(), nil
}

// FindByCommandID 按建造命令幂等键读取聚合副本。
func (r *BuildingRepository) FindByCommandID(ctx context.Context, commandID string) (*building.Building, error) {
	r.mu.RLock()
	id, ok := r.byCommand[commandID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("建造命令不存在，commandID=%s: %w", commandID, gameerr.ErrBuildingNotFound)
	}
	return r.FindByID(ctx, id)
}

// SaveIfAbsent 原子保存建筑；重复命令返回既有建筑且created为false。
func (r *BuildingRepository) SaveIfAbsent(_ context.Context, aggregate *building.Building) (*building.Building, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byCommand[aggregate.CommandID()]; ok {
		existing := r.byID[id]
		if existing.PlayerID() != aggregate.PlayerID() || existing.TypeID() != aggregate.TypeID() || existing.ConfigVersion() != aggregate.ConfigVersion() {
			return nil, false, fmt.Errorf("建造命令载荷冲突，commandID=%s: %w", aggregate.CommandID(), gameerr.ErrStateConflict)
		}
		return existing.Clone(), false, nil
	}
	if _, ok := r.byID[aggregate.ID()]; ok {
		return nil, false, fmt.Errorf("建筑ID冲突，buildingID=%d: %w", aggregate.ID(), gameerr.ErrStateConflict)
	}
	r.byID[aggregate.ID()] = aggregate.Clone()
	r.byCommand[aggregate.CommandID()] = aggregate.ID()
	return aggregate.Clone(), true, nil
}

// Save 保存已存在建筑的最新状态。
func (r *BuildingRepository) Save(_ context.Context, aggregate *building.Building) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[aggregate.ID()]; !ok {
		return fmt.Errorf("建筑不存在，buildingID=%d: %w", aggregate.ID(), gameerr.ErrBuildingNotFound)
	}
	r.byID[aggregate.ID()] = aggregate.Clone()
	return nil
}

// TrainingRepository 是并发安全的内存训练任务仓储。
type TrainingRepository struct {
	mu        sync.RWMutex             // 保护全部索引
	byID      map[int64]*training.Task // 训练任务ID索引
	byCommand map[string]int64         // 训练命令ID到任务ID索引
}

// NewTrainingRepository 创建空的内存训练任务仓储。
func NewTrainingRepository() *TrainingRepository {
	return &TrainingRepository{byID: make(map[int64]*training.Task), byCommand: make(map[string]int64)}
}

// FindByID 按训练任务ID读取聚合副本。
func (r *TrainingRepository) FindByID(_ context.Context, taskID int64) (*training.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.byID[taskID]
	if !ok {
		return nil, fmt.Errorf("训练任务不存在，taskID=%d: %w", taskID, gameerr.ErrTrainingNotFound)
	}
	return value.Clone(), nil
}

// FindByCommandID 按训练命令幂等键读取聚合副本。
func (r *TrainingRepository) FindByCommandID(ctx context.Context, commandID string) (*training.Task, error) {
	r.mu.RLock()
	id, ok := r.byCommand[commandID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("训练命令不存在，commandID=%s: %w", commandID, gameerr.ErrTrainingNotFound)
	}
	return r.FindByID(ctx, id)
}

// SaveIfAbsent 原子保存训练任务；重复命令返回既有任务且created为false。
func (r *TrainingRepository) SaveIfAbsent(_ context.Context, aggregate *training.Task) (*training.Task, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byCommand[aggregate.CommandID()]; ok {
		existing := r.byID[id]
		if existing.PlayerID() != aggregate.PlayerID() || existing.BuildingID() != aggregate.BuildingID() || existing.UnitTypeID() != aggregate.UnitTypeID() || existing.Count() != aggregate.Count() || existing.ConfigVersion() != aggregate.ConfigVersion() {
			return nil, false, fmt.Errorf("训练命令载荷冲突，commandID=%s: %w", aggregate.CommandID(), gameerr.ErrStateConflict)
		}
		return existing.Clone(), false, nil
	}
	if _, ok := r.byID[aggregate.ID()]; ok {
		return nil, false, fmt.Errorf("训练任务ID冲突，taskID=%d: %w", aggregate.ID(), gameerr.ErrStateConflict)
	}
	r.byID[aggregate.ID()] = aggregate.Clone()
	r.byCommand[aggregate.CommandID()] = aggregate.ID()
	return aggregate.Clone(), true, nil
}

// Save 保存已存在训练任务的最新状态。
func (r *TrainingRepository) Save(_ context.Context, aggregate *training.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[aggregate.ID()]; !ok {
		return fmt.Errorf("训练任务不存在，taskID=%d: %w", aggregate.ID(), gameerr.ErrTrainingNotFound)
	}
	r.byID[aggregate.ID()] = aggregate.Clone()
	return nil
}
