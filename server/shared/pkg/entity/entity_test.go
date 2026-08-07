// Package entity 实体共享内核，提供跨服务共享的实体基础类型与值对象。
package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPosition_Equal 测试坐标相等性判定。
func TestPosition_Equal(t *testing.T) {
	tests := []struct {
		name   string
		p      Position
		other  Position
		expect bool
	}{
		{"相同坐标", Position{X: 10, Y: 20}, Position{X: 10, Y: 20}, true},
		{"X不同", Position{X: 10, Y: 20}, Position{X: 11, Y: 20}, false},
		{"Y不同", Position{X: 10, Y: 20}, Position{X: 10, Y: 21}, false},
		{"都不同", Position{X: 10, Y: 20}, Position{X: 11, Y: 21}, false},
		{"零值", Position{}, Position{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.p.Equal(tt.other))
		})
	}
}

// TestPosition_Distance 测试曼哈顿距离计算。
func TestPosition_Distance(t *testing.T) {
	tests := []struct {
		name   string
		p      Position
		other  Position
		expect int32
	}{
		{"同点距离为0", Position{X: 5, Y: 5}, Position{X: 5, Y: 5}, 0},
		{"水平距离", Position{X: 0, Y: 0}, Position{X: 3, Y: 0}, 3},
		{"垂直距离", Position{X: 0, Y: 0}, Position{X: 0, Y: 4}, 4},
		{"对角距离", Position{X: 0, Y: 0}, Position{X: 3, Y: 4}, 7},
		{"负方向距离", Position{X: 5, Y: 5}, Position{X: 2, Y: 1}, 7},
		{"混合方向", Position{X: -3, Y: 5}, Position{X: 2, Y: 1}, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.p.Distance(tt.other))
		})
	}
}

// TestCoordinate_Equal 测试行列坐标相等性判定。
func TestCoordinate_Equal(t *testing.T) {
	tests := []struct {
		name   string
		c      Coordinate
		other  Coordinate
		expect bool
	}{
		{"相同行列", Coordinate{Row: 1, Col: 2}, Coordinate{Row: 1, Col: 2}, true},
		{"行不同", Coordinate{Row: 1, Col: 2}, Coordinate{Row: 2, Col: 2}, false},
		{"列不同", Coordinate{Row: 1, Col: 2}, Coordinate{Row: 1, Col: 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.c.Equal(tt.other))
		})
	}
}

// TestEntityTypeConstants 测试实体类型枚举常量值。
func TestEntityTypeConstants(t *testing.T) {
	assert.Equal(t, EntityType(1), EntityTypeUnit)
	assert.Equal(t, EntityType(2), EntityTypeBuilding)
	assert.Equal(t, EntityType(3), EntityTypeHero)
	assert.Equal(t, EntityType(4), EntityTypeResource)
	assert.Equal(t, EntityType(5), EntityTypeStronghold)
}

// TestEntityStatusConstants 测试实体状态枚举常量值。
func TestEntityStatusConstants(t *testing.T) {
	assert.Equal(t, EntityStatus(1), EntityStatusIdle)
	assert.Equal(t, EntityStatus(2), EntityStatusMoving)
	assert.Equal(t, EntityStatus(3), EntityStatusCombat)
	assert.Equal(t, EntityStatus(4), EntityStatusDead)
	assert.Equal(t, EntityStatus(5), EntityStatusGathering)
}