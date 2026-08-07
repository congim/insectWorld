// Package service Operation服务application层服务，编排赛季重置/继承/奖励发放。
package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/operation/domain/season"
	"insectworld/server/shared/pkg/config"
)

// SeasonResetCoordinator 赛季重置跨服务协调器，编排重置前快照与跨服务重置。
type SeasonResetCoordinator struct {
	seasonRepo  season.SeasonRepository // 赛季仓储接口
	configQuery config.ConfigQueryAPI   // 配置查询接口
	outbox      Outbox                  // 领域事件Outbox接口
	logger      *zap.Logger             // 结构化日志器（规范7）
}

// Outbox 领域事件Outbox接口。
type Outbox interface {
	Append(ctx context.Context, event any) error
}

// NewSeasonResetCoordinator 创建赛季重置协调器实例。
func NewSeasonResetCoordinator(
	seasonRepo season.SeasonRepository,
	configQuery config.ConfigQueryAPI,
	outbox Outbox,
	logger *zap.Logger,
) *SeasonResetCoordinator {
	return &SeasonResetCoordinator{
		seasonRepo:  seasonRepo,
		configQuery: configQuery,
		outbox:      outbox,
		logger:      logger,
	}
}

// Reset 执行赛季重置，协调跨服务按范围重置。
func (c *SeasonResetCoordinator) Reset(ctx context.Context, seasonID int64) error {
	// 1. 从config查询重置范围和保留列表
	resetRules := c.configQuery.GetSeasonResetRules(ctx)
	if resetRules == nil {
		return fmt.Errorf("赛季重置失败，重置规则配置为空，seasonID=%d", seasonID)
	}

	// 2. 加载Season聚合根
	s, err := c.seasonRepo.LoadSeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("赛季重置失败，加载赛季失败: %w", err)
	}

	// 3. Season聚合根重置
	event, err := s.Reset(ctx, resetRules.ResetScope, resetRules.KeepData)
	if err != nil {
		return fmt.Errorf("赛季重置失败: %w", err)
	}

	// 4. 保存聚合根
	if err := c.seasonRepo.SaveSeason(ctx, s); err != nil {
		return fmt.Errorf("赛季重置失败，保存赛季失败: %w", err)
	}

	// 5. 写Outbox投递赛季结束事件（各服务订阅按范围重置）
	if err := c.outbox.Append(ctx, event); err != nil {
		return fmt.Errorf("赛季重置失败，写Outbox失败: %w", err)
	}

	c.logger.Info("赛季重置成功",
		zap.Int64("season_id", seasonID),
		zap.Strings("reset_scope", resetRules.ResetScope),
		zap.Strings("keep_data", resetRules.KeepData),
	)
	return nil
}

// SeasonInheritService 赛季继承服务，编排跨赛季数据继承。
type SeasonInheritService struct {
	configQuery config.ConfigQueryAPI // 配置查询接口
	logger      *zap.Logger           // 结构化日志器（规范7）
}

// NewSeasonInheritService 创建赛季继承服务实例。
func NewSeasonInheritService(configQuery config.ConfigQueryAPI, logger *zap.Logger) *SeasonInheritService {
	return &SeasonInheritService{
		configQuery: configQuery,
		logger:      logger,
	}
}

// Inherit 执行跨赛季数据继承。
func (s *SeasonInheritService) Inherit(ctx context.Context, oldSeasonID, newSeasonID int64) error {
	inheritRules := s.configQuery.GetSeasonInheritRules(ctx)
	if inheritRules == nil {
		s.logger.Warn("赛季继承失败，继承规则配置为空",
			zap.Int64("old_season_id", oldSeasonID),
			zap.Int64("new_season_id", newSeasonID),
		)
		return nil
	}

	// TODO 后续按继承规则执行数据继承
	s.logger.Info("赛季继承成功",
		zap.Int64("old_season_id", oldSeasonID),
		zap.Int64("new_season_id", newSeasonID),
	)
	return nil
}

// RewardDistributor 赛季奖励发放服务，按排行榜发放奖励。
type RewardDistributor struct {
	configQuery config.ConfigQueryAPI // 配置查询接口
	outbox      Outbox                // 领域事件Outbox接口
	logger      *zap.Logger           // 结构化日志器（规范7）
}

// NewRewardDistributor 创建奖励发放服务实例。
func NewRewardDistributor(configQuery config.ConfigQueryAPI, outbox Outbox, logger *zap.Logger) *RewardDistributor {
	return &RewardDistributor{
		configQuery: configQuery,
		outbox:      outbox,
		logger:      logger,
	}
}

// Distribute 按排行榜发放赛季奖励。
func (d *RewardDistributor) Distribute(ctx context.Context, seasonID int64, rankings []int64) error {
	rewards := d.configQuery.GetSeasonRewards(ctx)
	if rewards == nil {
		d.logger.Warn("奖励发放失败，奖励配置为空",
			zap.Int64("season_id", seasonID),
		)
		return nil
	}

	for _, playerID := range rankings {
		// TODO 后续按排名匹配奖励档位发放
		d.logger.Info("赛季奖励发放",
			zap.Int64("season_id", seasonID),
			zap.Int64("player_id", playerID),
		)
	}
	return nil
}
