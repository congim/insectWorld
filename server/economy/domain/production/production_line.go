// Package production 生产线domain实体，维护生产状态与Tick产出。
package production

import "context"

// ProductionLine 生产线domain实体，维护生产状态与Tick产出。
type ProductionLine struct {
	lineID         int64 // 生产线ID，全局唯一
	playerID       int64 // 玩家ID
	resourceType   int64 // 产出资源类型ID
	productionRate int64 // 产出速率，每Tick产出数量
	lastTickTime   int64 // 上次Tick时间戳（毫秒）
	active         bool  // 是否活跃
}

// NewProductionLine 创建生产线实例。
func NewProductionLine(lineID, playerID, resourceType, productionRate int64) *ProductionLine {
	return &ProductionLine{
		lineID:         lineID,
		playerID:       playerID,
		resourceType:   resourceType,
		productionRate: productionRate,
		active:         true,
	}
}

// LineID 返回生产线ID。
func (p *ProductionLine) LineID() int64 { return p.lineID }

// PlayerID 返回玩家ID。
func (p *ProductionLine) PlayerID() int64 { return p.playerID }

// ResourceType 返回产出资源类型。
func (p *ProductionLine) ResourceType() int64 { return p.resourceType }

// IsActive 判断是否活跃。
func (p *ProductionLine) IsActive() bool { return p.active }

// Stop 停止生产。
func (p *ProductionLine) Stop() { p.active = false }

// Start 启动生产。
func (p *ProductionLine) Start() { p.active = true }

// Tick 执行一次生产Tick，返回产出量。
func (p *ProductionLine) Tick(ctx context.Context, tickTime int64) int64 {
	if !p.active {
		return 0
	}
	p.lastTickTime = tickTime
	return p.productionRate
}
