// Package gamemap 地图聚合根，维护地图格子状态与地形定义。
// 本文件定义MapInitializer domain service，按配置的格子类型初始化空间管理器。
package gamemap

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/config"
)

// 默认地图尺寸常量，配置未加载时作为兜底默认值。
// 后续接入完整配置解析后，从game.json配置注入实际尺寸。
const (
	defaultMapWidth  int32 = 100 // 默认地图宽度，格子数
	defaultMapHeight int32 = 100 // 默认地图高度，格子数
)

// MapInitializer 地图初始化domain service，按配置的格子类型初始化空间管理器。
// 从共享内核config查询game.json与terrains.json配置，创建Map聚合根。
type MapInitializer struct {
	configQuery config.ConfigQueryAPI // 配置查询接口，查询game.json与terrains.json
	logger      *zap.Logger           // 结构化日志器（规范7）
}

// NewMapInitializer 创建地图初始化domain service实例。
func NewMapInitializer(configQuery config.ConfigQueryAPI, logger *zap.Logger) *MapInitializer {
	return &MapInitializer{
		configQuery: configQuery,
		logger:      logger,
	}
}

// GameConfig game.json编译后的配置结构，描述地图全局参数。
type GameConfig struct {
	GridType int   // 格子类型：1=hex 2=quad 3=free（规范8用int枚举）
	Width    int32 // 地图宽度（格子数）
	Height   int32 // 地图高度（格子数）
}

// Init 按配置初始化地图聚合根。
// 从config查询game.json获取格子类型与尺寸，创建Map聚合根。
func (mi *MapInitializer) Init(ctx context.Context, mapID int64) (*Map, error) {
	// 从config查询game.json配置（通过扩展点查询）
	gameCfgAny, err := mi.configQuery.QueryByExtensionPoint(ctx, config.ExtPointTerrains)
	if err != nil {
		mi.logger.Error("地图初始化失败，配置查询失败",
			zap.Int64("map_id", mapID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("地图初始化失败，配置查询失败: %w", err)
	}

	// TODO 后续接入game.json扩展点后，从gameCfgAny解析GameConfig
	// 当前阶段使用默认配置初始化
	gameCfg := mi.parseGameConfig(gameCfgAny)

	m := NewMap(mapID, gameCfg.Width, gameCfg.Height, gameCfg.GridType)

	mi.logger.Info("地图初始化成功",
		zap.Int64("map_id", mapID),
		zap.Int32("width", gameCfg.Width),
		zap.Int32("height", gameCfg.Height),
		zap.Int("grid_type", gameCfg.GridType),
	)

	return m, nil
}

// parseGameConfig 解析game.json配置，当前阶段提供默认值。
// TODO 后续接入完整配置解析逻辑，从配置数据解析格子类型与尺寸。
func (mi *MapInitializer) parseGameConfig(cfg any) GameConfig {
	if cfg == nil {
		return GameConfig{
			GridType: GridTypeQuad,
			Width:    defaultMapWidth,
			Height:   defaultMapHeight,
		}
	}
	return GameConfig{
		GridType: GridTypeQuad,
		Width:    defaultMapWidth,
		Height:   defaultMapHeight,
	}
}

// MapRepository Map聚合根仓储接口，在domain层声明（规范3），infrastructure层实现。
type MapRepository interface {
	// LoadMap 加载地图聚合根
	LoadMap(ctx context.Context, mapID int64) (*Map, error)
	// SaveMap 保存地图聚合根
	SaveMap(ctx context.Context, m *Map) error
}
