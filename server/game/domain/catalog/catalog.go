// Package catalog 定义成长领域读取游戏包内容所需的稳定契约。
package catalog

import "context"

// FactionDefinition 阵营定义，只暴露玩家初始化需要的数据。
type FactionDefinition struct {
	ID                string           // ID 是跨版本稳定阵营ID
	StartingResources map[string]int64 // StartingResources 是初始资源数量，key为稳定资源ID
}

// BuildingDefinition 建筑定义，只暴露建造与训练校验需要的数据。
type BuildingDefinition struct {
	ID               string           // ID 是跨版本稳定建筑ID
	FactionID        string           // FactionID 是所属阵营稳定ID
	BuildCost        map[string]int64 // BuildCost 是建造资源消耗，key为稳定资源ID
	BuildTimeMs      int64            // BuildTimeMs 是建造耗时，单位毫秒
	TrainableUnitIDs []string         // TrainableUnitIDs 是该建筑允许训练的单位ID
}

// UnitDefinition 单位定义，只暴露训练流程需要的数据。
type UnitDefinition struct {
	ID          string           // ID 是跨版本稳定单位ID
	FactionID   string           // FactionID 是所属阵营稳定ID
	TrainCost   map[string]int64 // TrainCost 是单个单位训练消耗，key为稳定资源ID
	TrainTimeMs int64            // TrainTimeMs 是单个单位训练耗时，单位毫秒
}

// Reader 是只读配置端口，运行实例必须绑定同一个已编译游戏包版本。
type Reader interface {
	DefaultFaction(ctx context.Context) (FactionDefinition, error)
	Faction(ctx context.Context, factionID string) (FactionDefinition, error)
	Building(ctx context.Context, buildingID string) (BuildingDefinition, error)
	Unit(ctx context.Context, unitID string) (UnitDefinition, error)
}
