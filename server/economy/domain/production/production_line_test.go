// Package production 生产线domain实体，维护生产状态与Tick产出。
// 本文件定义ProductionLine的单元测试。
package production

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewProductionLine 测试生产线创建。
func TestNewProductionLine(t *testing.T) {
	line := NewProductionLine(1, 100, 200, 50)

	assert.Equal(t, int64(1), line.LineID())
	assert.Equal(t, int64(100), line.PlayerID())
	assert.Equal(t, int64(200), line.ResourceType())
	assert.True(t, line.IsActive())
}

// TestProductionLine_Tick 测试生产Tick产出。
func TestProductionLine_Tick(t *testing.T) {
	line := NewProductionLine(1, 100, 200, 50)

	produced := line.Tick(context.Background(), 1000)
	assert.Equal(t, int64(50), produced)
}

// TestProductionLine_Tick_Inactive 测试非活跃生产线Tick产出为0。
func TestProductionLine_Tick_Inactive(t *testing.T) {
	line := NewProductionLine(1, 100, 200, 50)
	line.Stop()

	produced := line.Tick(context.Background(), 1000)
	assert.Equal(t, int64(0), produced)
	assert.False(t, line.IsActive())
}

// TestProductionLine_StopStart 测试停止与启动生产。
func TestProductionLine_StopStart(t *testing.T) {
	line := NewProductionLine(1, 100, 200, 50)

	line.Stop()
	assert.False(t, line.IsActive())
	assert.Equal(t, int64(0), line.Tick(context.Background(), 1000))

	line.Start()
	assert.True(t, line.IsActive())
	assert.Equal(t, int64(50), line.Tick(context.Background(), 2000))
}
