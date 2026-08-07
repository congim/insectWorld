// Package alliance 联盟聚合根，维护联盟成员、等级与外交关系。
// 本文件定义Alliance聚合根、PermissionChecker与WelfareService的单元测试。
package alliance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/config/mock"
)

// mockAllianceRepo 测试用AllianceRepository mock。
type mockAllianceRepo struct {
	alliance *Alliance
}

func (m *mockAllianceRepo) LoadAlliance(ctx context.Context, allianceID int64) (*Alliance, error) {
	return m.alliance, nil
}

func (m *mockAllianceRepo) SaveAlliance(ctx context.Context, a *Alliance) error {
	return nil
}

// newTestAllianceRepo 创建测试用联盟仓储，包含指定玩家。
func newTestAllianceRepo(playerID int64) *mockAllianceRepo {
	a := NewAlliance(1, "测试联盟", 50)
	_, _ = a.AddMember(context.Background(), playerID, 1000)
	return &mockAllianceRepo{alliance: a}
}

// TestNewAlliance 测试联盟创建。
func TestNewAlliance(t *testing.T) {
	a := NewAlliance(1, "测试联盟", 50)
	assert.Equal(t, int64(1), a.AllianceID())
	assert.Equal(t, 0, a.MemberCount())
}

// TestAlliance_AddMember 测试添加成员。
func TestAlliance_AddMember(t *testing.T) {
	a := NewAlliance(1, "测试联盟", 50)

	event, err := a.AddMember(context.Background(), 101, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.AllianceID)
	assert.Equal(t, int64(101), event.PlayerID)
	assert.Equal(t, 1, event.ChangeType)
	assert.Equal(t, 1, a.MemberCount())

	role, ok := a.GetMemberRole(101)
	assert.True(t, ok)
	assert.Equal(t, RoleMember, role)
}

// TestAlliance_AddMember_Full 测试联盟已满时添加成员失败。
func TestAlliance_AddMember_Full(t *testing.T) {
	a := NewAlliance(1, "测试联盟", 1)

	_, err := a.AddMember(context.Background(), 101, 1000)
	require.NoError(t, err)

	_, err = a.AddMember(context.Background(), 102, 2000)
	assert.Error(t, err)
}

// TestAlliance_AddMember_Cooldown 测试加入冷却中添加成员失败。
func TestAlliance_AddMember_Cooldown(t *testing.T) {
	a := NewAlliance(1, "测试联盟", 50)
	a.SetJoinCooldown(5000)

	_, err := a.AddMember(context.Background(), 101, 3000)
	assert.Error(t, err)
}

// TestAlliance_AddMember_CooldownExpired 测试冷却结束后添加成员成功。
func TestAlliance_AddMember_CooldownExpired(t *testing.T) {
	a := NewAlliance(1, "测试联盟", 50)
	a.SetJoinCooldown(5000)

	_, err := a.AddMember(context.Background(), 101, 5000)
	require.NoError(t, err)
	assert.Equal(t, 1, a.MemberCount())
}

// TestAlliance_RemoveMember 测试移除成员。
func TestAlliance_RemoveMember(t *testing.T) {
	a := NewAlliance(1, "测试联盟", 50)

	_, err := a.AddMember(context.Background(), 101, 1000)
	require.NoError(t, err)

	event, err := a.RemoveMember(context.Background(), 101)
	require.NoError(t, err)
	assert.Equal(t, int64(101), event.PlayerID)
	assert.Equal(t, 2, event.ChangeType)
	assert.Equal(t, 0, a.MemberCount())

	_, ok := a.GetMemberRole(101)
	assert.False(t, ok)
}

// TestAlliance_RemoveMember_NotInAlliance 测试移除不在联盟中的成员失败。
func TestAlliance_RemoveMember_NotInAlliance(t *testing.T) {
	a := NewAlliance(1, "测试联盟", 50)

	_, err := a.RemoveMember(context.Background(), 999)
	assert.Error(t, err)
}

// TestPermissionChecker_Check 测试权限校验通过。
func TestPermissionChecker_Check(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.AlliancePermissions = map[string][]string{
		"3": {"collect", "view"},
		"2": {"collect", "view", "kick"},
		"1": {"collect", "view", "kick", "disband"},
	}
	checker := NewPermissionChecker(cfg, newTestAllianceRepo(101), zap.NewNop())

	result := checker.Check(context.Background(), 1, 101, "collect")
	assert.True(t, result)
}

// TestPermissionChecker_Check_Denied 测试权限校验不通过。
func TestPermissionChecker_Check_Denied(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.AlliancePermissions = map[string][]string{
		"3": {"collect", "view"},
	}
	checker := NewPermissionChecker(cfg, newTestAllianceRepo(101), zap.NewNop())

	result := checker.Check(context.Background(), 1, 101, "kick")
	assert.False(t, result)
}

// TestPermissionChecker_Check_NoConfig 测试权限配置为空时校验失败。
func TestPermissionChecker_Check_NoConfig(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	checker := NewPermissionChecker(cfg, newTestAllianceRepo(101), zap.NewNop())

	result := checker.Check(context.Background(), 1, 101, "collect")
	assert.False(t, result)
}

// TestWelfareService_Distribute 测试福利发放成功。
func TestWelfareService_Distribute(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.AllianceWelfare["welfare_1"] = &config.WelfareConfig{
		WelfareID:             "welfare_1",
		WelfareType:           1,
		EffectType:            1,
		EffectValue:           500,
		RequiredAllianceLevel: 1,
	}
	svc := NewWelfareService(cfg, zap.NewNop())

	event, err := svc.Distribute(context.Background(), 1, 101, "welfare_1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.AllianceID)
	assert.Equal(t, int64(101), event.PlayerID)
	assert.Equal(t, 1, event.WelfareType)
}

// TestWelfareService_Distribute_NotFound 测试福利配置不存在时发放失败。
func TestWelfareService_Distribute_NotFound(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	svc := NewWelfareService(cfg, zap.NewNop())

	_, err := svc.Distribute(context.Background(), 1, 101, "nonexistent")
	assert.Error(t, err)
}
