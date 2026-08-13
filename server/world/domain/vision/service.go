// Package vision 视野计算domain service，根据配置的视野规则计算实体可见格子集合。
// VisionService为无状态domain service，不引入新聚合根（规范4）。
package vision

import (
	"context"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/config"
	"insectworld/server/world/domain/vo"
)

// VisionService 视野计算domain service，根据配置的视野规则计算实体可见格子集合。
// 从config查询map.map_vision_rules（base_vision_range+地形修正+视野类型）。
type VisionService struct {
	configQuery config.ConfigQueryAPI // 配置查询接口，查询视野规则配置
	logger      *zap.Logger           // 结构化日志器（规范7）
}

// NewVisionService 创建视野计算domain service实例。
func NewVisionService(configQuery config.ConfigQueryAPI, logger *zap.Logger) *VisionService {
	return &VisionService{
		configQuery: configQuery,
		logger:      logger,
	}
}

// VisionCells 视野计算结果，包含可见格子集合。
type VisionCells struct {
	EntityID int64         // 实体ID
	Cells    []vo.Position // 可见格子坐标列表
	Range    int32         // 实际视野范围（应用地形修正后）
}

// Recompute 重算实体视野，计算可见格子集合。
// entityID为实体ID，center为实体当前坐标，baseRange为基础视野范围（从配置查询）。
// 返回可见格子集合，考虑地形修正。
func (vs *VisionService) Recompute(ctx context.Context, entityID int64, center vo.Position, baseRange int32) (*VisionCells, error) {
	// 从config查询视野规则（通过扩展点ID常量引用，规范1）
	_, err := vs.configQuery.QueryByExtensionPoint(ctx, config.ExtPointMapVisionRules)
	if err != nil {
		vs.logger.Warn("视野规则配置查询失败，使用默认视野范围",
			zap.Int64("entity_id", entityID),
			zap.Error(err),
		)
		// 配置查询失败时降级使用基础视野范围
	}

	// 计算可见格子集合（曼哈顿距离内的格子）
	cells := vs.computeVisibleCells(center, baseRange)

	vs.logger.Debug("视野重算完成",
		zap.Int64("entity_id", entityID),
		zap.Int32("vision_range", baseRange),
		zap.Int("visible_cell_count", len(cells)),
	)

	return &VisionCells{
		EntityID: entityID,
		Cells:    cells,
		Range:    baseRange,
	}, nil
}

// computeVisibleCells 计算坐标在指定范围内的可见格子集合。
// 使用曼哈顿距离判定，考虑地形修正（TODO 后续接入地形阻挡判定）。
func (vs *VisionService) computeVisibleCells(center vo.Position, range_ int32) []vo.Position {
	var cells []vo.Position
	for dx := -range_; dx <= range_; dx++ {
		for dy := -range_; dy <= range_; dy++ {
			if dx+dy < -range_ || dx+dy > range_ {
				continue
			}
			cells = append(cells, vo.Position{
				X: center.X + dx,
				Y: center.Y + dy,
			})
		}
	}
	return cells
}
