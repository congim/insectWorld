// Package main 游戏包契约验证工具测试。
package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatorValidate 验证工具真实编译仓库内两个游戏包。
func TestValidatorValidate(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "gamepacks"))
	result := NewValidator(root, "0.1.0").Validate()

	require.Equal(t, ResultPass, result.Status)
	assert.Len(t, result.PackResults, 2)
	for _, pack := range result.PackResults {
		assert.Equal(t, ResultPass, pack.Status)
		assert.NotEmpty(t, pack.PackID)
	}
}

// TestValidatorValidateMissingRoot 验证不存在的根目录返回失败而非跳过。
func TestValidatorValidateMissingRoot(t *testing.T) {
	result := NewValidator(filepath.Join(t.TempDir(), "missing"), "0.1.0").Validate()

	require.Equal(t, ResultFail, result.Status)
	require.Len(t, result.PackResults, 1)
	assert.Contains(t, result.PackResults[0].Detail, "读取游戏包根目录失败")
}
