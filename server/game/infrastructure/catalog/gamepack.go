// Package catalog 将已编译游戏包适配为Growth领域只读目录。
package catalog

import (
	"context"
	"fmt"

	domaincatalog "insectworld/server/game/domain/catalog"
	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/shared/pkg/gamepack"
)

// GamePackReader 在进程生命周期内绑定一个不可变的已编译游戏包版本。
type GamePackReader struct {
	defaultFactionID string                                      // 默认阵营稳定ID
	factions         map[string]domaincatalog.FactionDefinition  // 阵营定义索引
	buildings        map[string]domaincatalog.BuildingDefinition // 建筑定义索引
	units            map[string]domaincatalog.UnitDefinition     // 单位定义索引
}

// NewGamePackReader 从校验通过的游戏包建立只读索引。
func NewGamePackReader(pack *gamepack.CompiledPack) (*GamePackReader, error) {
	if pack == nil {
		return nil, fmt.Errorf("游戏包不能为空: %w", gameerr.ErrInvalidCommand)
	}
	reader := &GamePackReader{
		defaultFactionID: pack.Game.DefaultFactionID,
		factions:         make(map[string]domaincatalog.FactionDefinition, len(pack.Factions)),
		buildings:        make(map[string]domaincatalog.BuildingDefinition, len(pack.Buildings)),
		units:            make(map[string]domaincatalog.UnitDefinition, len(pack.Units)),
	}
	for _, value := range pack.Factions {
		reader.factions[value.ID] = domaincatalog.FactionDefinition{ID: value.ID, StartingResources: cloneAmounts(value.StartingResources)}
	}
	for _, value := range pack.Buildings {
		reader.buildings[value.ID] = domaincatalog.BuildingDefinition{ID: value.ID, FactionID: value.FactionID, BuildCost: cloneAmounts(value.BuildCost), BuildTimeMs: value.BuildTimeMs, TrainableUnitIDs: append([]string(nil), value.TrainableUnitIDs...)}
	}
	for _, value := range pack.Units {
		reader.units[value.ID] = domaincatalog.UnitDefinition{ID: value.ID, FactionID: value.FactionID, TrainCost: cloneAmounts(value.TrainCost), TrainTimeMs: value.TrainTimeMs}
	}
	return reader, nil
}

// DefaultFaction 返回当前游戏包默认阵营定义。
func (r *GamePackReader) DefaultFaction(ctx context.Context) (domaincatalog.FactionDefinition, error) {
	return r.Faction(ctx, r.defaultFactionID)
}

// Faction 返回指定阵营定义的防御性副本。
func (r *GamePackReader) Faction(_ context.Context, factionID string) (domaincatalog.FactionDefinition, error) {
	value, ok := r.factions[factionID]
	if !ok {
		return domaincatalog.FactionDefinition{}, fmt.Errorf("阵营定义不存在，factionID=%s: %w", factionID, gameerr.ErrDefinitionNotFound)
	}
	value.StartingResources = cloneAmounts(value.StartingResources)
	return value, nil
}

// Building 返回指定建筑定义的防御性副本。
func (r *GamePackReader) Building(_ context.Context, buildingID string) (domaincatalog.BuildingDefinition, error) {
	value, ok := r.buildings[buildingID]
	if !ok {
		return domaincatalog.BuildingDefinition{}, fmt.Errorf("建筑定义不存在，buildingID=%s: %w", buildingID, gameerr.ErrDefinitionNotFound)
	}
	value.BuildCost = cloneAmounts(value.BuildCost)
	value.TrainableUnitIDs = append([]string(nil), value.TrainableUnitIDs...)
	return value, nil
}

// Unit 返回指定单位定义的防御性副本。
func (r *GamePackReader) Unit(_ context.Context, unitID string) (domaincatalog.UnitDefinition, error) {
	value, ok := r.units[unitID]
	if !ok {
		return domaincatalog.UnitDefinition{}, fmt.Errorf("单位定义不存在，unitID=%s: %w", unitID, gameerr.ErrDefinitionNotFound)
	}
	value.TrainCost = cloneAmounts(value.TrainCost)
	return value, nil
}

func cloneAmounts(source map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
