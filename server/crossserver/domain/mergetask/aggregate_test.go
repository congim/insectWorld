// Package mergetask 合服任务聚合根单元测试。
package mergetask

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMergeTask 测试合服任务创建。
func TestNewMergeTask(t *testing.T) {
	m := NewMergeTask(1, []int64{100, 200}, 300, 50000, 1000)
	assert.Equal(t, int64(1), m.TaskID())
	assert.Equal(t, StatusPending, m.Status())
	assert.Equal(t, int64(0), m.Progress())
}

// TestMergeTask_Start 测试开始迁移。
func TestMergeTask_Start(t *testing.T) {
	m := NewMergeTask(1, []int64{100, 200}, 300, 50000, 1000)
	err := m.Start()
	require.NoError(t, err)
	assert.Equal(t, StatusMigrating, m.Status())
}

// TestMergeTask_Start_AlreadyRunning 测试重复开始失败。
func TestMergeTask_Start_AlreadyRunning(t *testing.T) {
	m := NewMergeTask(1, []int64{100, 200}, 300, 50000, 1000)
	_ = m.Start()
	err := m.Start()
	assert.Error(t, err)
}

// TestMergeTask_AdvanceProgress 测试推进迁移进度。
func TestMergeTask_AdvanceProgress(t *testing.T) {
	m := NewMergeTask(1, []int64{100, 200}, 300, 50000, 1000)
	_ = m.Start()
	m.AdvanceProgress(1000, PhasePlayer)
	assert.Equal(t, int64(1000), m.Progress())
}

// TestMergeTask_EnterVerify 测试进入校验阶段。
func TestMergeTask_EnterVerify(t *testing.T) {
	m := NewMergeTask(1, []int64{100, 200}, 300, 50000, 1000)
	_ = m.Start()
	m.EnterVerify()
	assert.Equal(t, StatusVerifying, m.Status())
}

// TestMergeTask_Complete 测试完成合服。
func TestMergeTask_Complete(t *testing.T) {
	m := NewMergeTask(1, []int64{100, 200}, 300, 50000, 1000)
	_ = m.Start()
	m.EnterVerify()

	event, err := m.Complete(5000)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, m.Status())
	assert.Equal(t, int64(300), event.TargetZone)
}

// TestMergeTask_Complete_NotVerifying 测试非校验中状态完成失败。
func TestMergeTask_Complete_NotVerifying(t *testing.T) {
	m := NewMergeTask(1, []int64{100, 200}, 300, 50000, 1000)
	_ = m.Start()
	_, err := m.Complete(5000)
	assert.Error(t, err)
}

// TestMergeTask_Fail 测试合服失败。
func TestMergeTask_Fail(t *testing.T) {
	m := NewMergeTask(1, []int64{100, 200}, 300, 50000, 1000)
	_ = m.Start()
	event := m.Fail("数据迁移失败", 5000)
	assert.Equal(t, StatusFailed, m.Status())
	assert.Equal(t, "数据迁移失败", event.ErrorMsg)
}
