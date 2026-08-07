// Package buff buff系统共享内核，提供buff模型与效果执行契约。
package buff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuffTypeConstants 测试buff类型枚举常量值。
func TestBuffTypeConstants(t *testing.T) {
	assert.Equal(t, BuffType(1), BuffTypeAttrModifier)
	assert.Equal(t, BuffType(2), BuffTypeControl)
	assert.Equal(t, BuffType(3), BuffTypeTrigger)
	assert.Equal(t, BuffType(4), BuffTypeHeal)
}

// TestBuffStack_ApplyStack_Replace 测试替换叠加策略。
func TestBuffStack_ApplyStack_Replace(t *testing.T) {
	stack := NewBuffStack(stackStrategyReplace, 0)
	tests := []struct {
		name     string
		current  int
		incoming int
		expect   int
	}{
		{"新buff替换旧buff", 3, 1, 1},
		{"新层数大于旧", 1, 5, 5},
		{"零层替换", 3, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, stack.ApplyStack(tt.current, tt.incoming))
		})
	}
}

// TestBuffStack_ApplyStack_Stack 测试叠加策略。
func TestBuffStack_ApplyStack_Stack(t *testing.T) {
	stack := NewBuffStack(stackStrategyStack, 5)
	tests := []struct {
		name     string
		current  int
		incoming int
		expect   int
	}{
		{"正常叠加", 2, 3, 5},
		{"叠加达到上限", 3, 3, 5},
		{"叠加超过上限", 4, 3, 5},
		{"零层叠加", 0, 2, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, stack.ApplyStack(tt.current, tt.incoming))
		})
	}
}

// TestBuffStack_ApplyStack_Stack_NoLimit 测试无上限叠加策略。
func TestBuffStack_ApplyStack_Stack_NoLimit(t *testing.T) {
	stack := NewBuffStack(stackStrategyStack, 0)
	assert.Equal(t, 10, stack.ApplyStack(7, 3))
	assert.Equal(t, 100, stack.ApplyStack(50, 50))
}

// TestBuffStack_ApplyStack_Refresh 测试刷新叠加策略。
func TestBuffStack_ApplyStack_Refresh(t *testing.T) {
	stack := NewBuffStack(stackStrategyRefresh, 0)
	tests := []struct {
		name     string
		current  int
		incoming int
		expect   int
	}{
		{"新层数更大取新", 2, 5, 5},
		{"旧层数更大保持旧", 5, 2, 5},
		{"层数相同", 3, 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, stack.ApplyStack(tt.current, tt.incoming))
		})
	}
}

// TestBuffStack_ApplyStack_UnknownStrategy 测试未知策略默认不叠加。
func TestBuffStack_ApplyStack_UnknownStrategy(t *testing.T) {
	stack := NewBuffStack(99, 0)
	assert.Equal(t, 3, stack.ApplyStack(3, 5))
}