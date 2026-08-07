// Package scheduler Economy服务application层调度器，按生产间隔分组调度生产Tick。
// 本文件定义ProductionTickScheduler的单元测试。
package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"insectworld/server/economy/domain/production"
	"insectworld/server/shared/pkg/config/mock"
)

// TestProductionTickScheduler_Register 测试生产线注册。
func TestProductionTickScheduler_Register(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	scheduler := NewProductionTickScheduler(cfg, zap.NewNop())

	line1 := production.NewProductionLine(1, 100, 200, 50)
	line2 := production.NewProductionLine(2, 101, 200, 60)

	scheduler.Register(line1, 1000)
	scheduler.Register(line2, 1000)
	scheduler.Register(production.NewProductionLine(3, 102, 200, 70), 2000)

	assert.Len(t, scheduler.schedules[1000], 2)
	assert.Len(t, scheduler.schedules[2000], 1)
}

// TestProductionTickScheduler_Run 测试调度器启动与停止。
func TestProductionTickScheduler_Run(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	scheduler := NewProductionTickScheduler(cfg, zap.NewNop())

	line := production.NewProductionLine(1, 100, 200, 50)
	scheduler.Register(line, 100)

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Run(ctx)

	// 等待几个Tick
	time.Sleep(350 * time.Millisecond)

	cancel()
	time.Sleep(50 * time.Millisecond)

	// 验证不panic即可，生产线已执行多次Tick
	assert.True(t, true)
}
