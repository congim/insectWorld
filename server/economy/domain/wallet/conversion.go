// Package wallet 玩家钱包聚合根，维护各资源余额的一致性边界。
// 本文件定义ConversionAggregate资源转换聚合根与TradeOrder交易订单聚合根。
package wallet

import (
	"context"
	"fmt"

	economyerr "insectworld/server/economy/domain/errors"
)

// 转换状态常量（规范1）。
const (
	ConversionStatusPending   = 1 // 待执行
	ConversionStatusCompleted = 2 // 已完成
	ConversionStatusFailed    = 3 // 已失败
)

// ConversionAggregate 资源转换聚合根，封装转换规则校验与原子执行。
// 资源转换订单有独立生命周期（创建/执行/完成），需聚合根（规范4）。
type ConversionAggregate struct {
	conversionID    int64           // 转换订单ID，全局唯一，由雪花算法生成
	playerID        int64           // 玩家ID
	ruleID          string          // 转换规则ID，对应economy.json的conversion_rules
	inputResources  map[int64]int64 // 输入资源映射，key=资源类型ID，value=数量
	outputResources map[int64]int64 // 输出资源映射，key=资源类型ID，value=数量
	status          int             // 转换状态：1=待执行 2=已完成 3=已失败
	createdAt       int64           // 创建时间戳（毫秒）
}

// NewConversion 创建资源转换聚合根实例。
func NewConversion(conversionID, playerID int64, ruleID string, input map[int64]int64, createdAt int64) *ConversionAggregate {
	return &ConversionAggregate{
		conversionID:    conversionID,
		playerID:        playerID,
		ruleID:          ruleID,
		inputResources:  input,
		outputResources: make(map[int64]int64),
		status:          ConversionStatusPending,
		createdAt:       createdAt,
	}
}

// Convert 执行资源转换，校验输入充足后原子执行。
// wallet为玩家钱包，conversionRatio为转换比例（由配置注入）。
func (c *ConversionAggregate) Convert(ctx context.Context, wallet *PlayerWallet, conversionRatio float64) error {
	if c.status != ConversionStatusPending {
		return fmt.Errorf("转换执行失败，状态非待执行，conversionID=%d: %w", c.conversionID, economyerr.ErrRuleViolation)
	}

	// 校验输入资源充足
	if !wallet.CheckSufficient(c.inputResources) {
		return fmt.Errorf("转换执行失败，输入资源不足，conversionID=%d: %w", c.conversionID, economyerr.ErrResourceInsufficient)
	}

	// 扣减输入资源
	for resType, amount := range c.inputResources {
		if err := wallet.Consume(ctx, resType, amount); err != nil {
			return err
		}
	}

	// 计算并增加输出资源
	for resType, amount := range c.inputResources {
		output := int64(float64(amount) * conversionRatio)
		c.outputResources[resType] = output
		wallet.AddBalance(resType, output)
	}

	c.status = ConversionStatusCompleted
	return nil
}

// 交易类型常量（规范1）。
const (
	TradeTypePlayerPlayer     = 1 // 玩家间交易
	TradeTypePlayerNPC        = 2 // 玩家与NPC交易
	TradeTypeAllianceAlliance = 3 // 联盟间交易
)

// 交易状态常量（规范1）。
const (
	TradeStatusPending   = 1 // 待确认
	TradeStatusCompleted = 2 // 已完成
	TradeStatusCancelled = 3 // 已取消
)

// TradeOrder 交易订单聚合根，支持玩家间/玩家NPC/联盟间交易。
type TradeOrder struct {
	orderID        int64           // 交易订单ID，全局唯一
	tradeType      int             // 交易类型：1=玩家间 2=玩家NPC 3=联盟间，由economy.json配置驱动
	fromPlayerID   int64           // 卖出方玩家ID
	toPlayerID     int64           // 买入方玩家ID（NPC交易时为NPC ID）
	fromAllianceID int64           // 卖出方联盟ID（联盟间交易）
	toAllianceID   int64           // 买入方联盟ID（联盟间交易）
	resources      map[int64]int64 // 交易资源映射，key=资源类型ID，value=数量
	status         int             // 交易状态：1=待确认 2=已完成 3=已取消
	createdAt      int64           // 创建时间戳（毫秒）
}

// NewTradeOrder 创建交易订单聚合根实例。
func NewTradeOrder(orderID int64, tradeType int, fromPlayerID, toPlayerID int64, resources map[int64]int64, createdAt int64) *TradeOrder {
	return &TradeOrder{
		orderID:      orderID,
		tradeType:    tradeType,
		fromPlayerID: fromPlayerID,
		toPlayerID:   toPlayerID,
		resources:    resources,
		status:       TradeStatusPending,
		createdAt:    createdAt,
	}
}

// Execute 执行交易，校验卖出方资源充足后原子执行。
func (t *TradeOrder) Execute(ctx context.Context, fromWallet, toWallet *PlayerWallet) error {
	if t.status != TradeStatusPending {
		return fmt.Errorf("交易执行失败，状态非待确认，orderID=%d: %w", t.orderID, economyerr.ErrRuleViolation)
	}

	if !fromWallet.CheckSufficient(t.resources) {
		return fmt.Errorf("交易执行失败，卖出方资源不足，orderID=%d: %w", t.orderID, economyerr.ErrResourceInsufficient)
	}

	for resType, amount := range t.resources {
		if err := fromWallet.Consume(ctx, resType, amount); err != nil {
			return err
		}
		toWallet.AddBalance(resType, amount)
	}

	t.status = TradeStatusCompleted
	return nil
}

// SetAlliance 设置联盟间交易的联盟ID。
func (t *TradeOrder) SetAlliance(fromAllianceID, toAllianceID int64) {
	t.fromAllianceID = fromAllianceID
	t.toAllianceID = toAllianceID
}
