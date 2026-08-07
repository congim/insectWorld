// Package scheduler Economy服务application层调度器，按生产间隔分组调度生产Tick。
package scheduler

import (
	"context"
	"time"

	"go.uber.org/zap"

	"insectworld/server/economy/domain/production"
	"insectworld/server/shared/pkg/config"
)

// ProductionTickScheduler 生产Tick全局调度器，按生产间隔分组调度。
type ProductionTickScheduler struct {
	schedules   map[int64][]*production.ProductionLine // 调度表，key=调度间隔（毫秒），value=生产线列表
	configQuery config.ConfigQueryAPI                  // 配置查询接口
	logger      *zap.Logger                            // 结构化日志器（规范7）
}

// NewProductionTickScheduler 创建生产Tick调度器实例。
func NewProductionTickScheduler(configQuery config.ConfigQueryAPI, logger *zap.Logger) *ProductionTickScheduler {
	return &ProductionTickScheduler{
		schedules:   make(map[int64][]*production.ProductionLine),
		configQuery: configQuery,
		logger:      logger,
	}
}

// Register 注册生产线到调度器，按生产间隔分组。
func (s *ProductionTickScheduler) Register(line *production.ProductionLine, intervalMs int64) {
	s.schedules[intervalMs] = append(s.schedules[intervalMs], line)
}

// Run 启动调度器，通过context控制生命周期（规范9goroutine安全）。
func (s *ProductionTickScheduler) Run(ctx context.Context) {
	tickers := make(map[int64]*time.Ticker)
	for interval := range s.schedules {
		tickers[interval] = time.NewTicker(time.Duration(interval) * time.Millisecond)
	}

	for interval, ticker := range tickers {
		go func(interval int64, t *time.Ticker) {
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case now := <-t.C:
					s.tick(ctx, interval, now.UnixMilli())
				}
			}
		}(interval, ticker)
	}
}

// tick 执行一次生产Tick，调度指定间隔的所有生产线。
func (s *ProductionTickScheduler) tick(ctx context.Context, interval int64, now int64) {
	lines := s.schedules[interval]
	totalProduced := int64(0)
	for _, line := range lines {
		produced := line.Tick(ctx, now)
		totalProduced += produced
	}
	s.logger.Debug("生产Tick完成",
		zap.Int64("interval_ms", interval),
		zap.Int("line_count", len(lines)),
		zap.Int64("total_produced", totalProduced),
	)
}
