// Package session 会话仓储infrastructure层实现，提供Redis与内存适配。
package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"

	"go.uber.org/zap"
)

// TestSessionRepoMemory_SaveAndFind 测试内存会话仓储保存与查询。
func TestSessionRepoMemory_SaveAndFind(t *testing.T) {
	logger := zap.NewNop()
	repo := NewSessionRepoMemory(5*time.Minute, logger)
	ctx := context.Background()

	sess := domainsession.NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
	require.NoError(t, repo.Save(ctx, sess))

	found, err := repo.FindByPlayerID(ctx, 1001)
	require.NoError(t, err)
	assert.Equal(t, int64(1001), found.PlayerID())
	assert.Equal(t, "conn-001", found.ConnID())
	assert.Equal(t, 1, found.TokenVersion())
}

// TestSessionRepoMemory_FindNotFound 测试查询不存在的会话返回ErrSessionNotFound。
func TestSessionRepoMemory_FindNotFound(t *testing.T) {
	logger := zap.NewNop()
	repo := NewSessionRepoMemory(5*time.Minute, logger)
	ctx := context.Background()

	_, err := repo.FindByPlayerID(ctx, 9999)
	require.Error(t, err)
	assert.Equal(t, gatewayerr.ErrSessionNotFound, err)
}

// TestSessionRepoMemory_Delete 测试会话删除。
func TestSessionRepoMemory_Delete(t *testing.T) {
	logger := zap.NewNop()
	repo := NewSessionRepoMemory(5*time.Minute, logger)
	ctx := context.Background()

	sess := domainsession.NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
	require.NoError(t, repo.Save(ctx, sess))

	require.NoError(t, repo.Delete(ctx, 1001))
	_, err := repo.FindByPlayerID(ctx, 1001)
	assert.Equal(t, gatewayerr.ErrSessionNotFound, err)
}

// TestSessionRepoMemory_DeleteNonExistent 测试删除不存在的会话不报错（幂等）。
func TestSessionRepoMemory_DeleteNonExistent(t *testing.T) {
	logger := zap.NewNop()
	repo := NewSessionRepoMemory(5*time.Minute, logger)
	err := repo.Delete(context.Background(), 9999)
	require.NoError(t, err)
}

// TestSessionRepoMemory_FindExpired 测试超时会话查询。
func TestSessionRepoMemory_FindExpired(t *testing.T) {
	logger := zap.NewNop()
	repo := NewSessionRepoMemory(5*time.Minute, logger)
	ctx := context.Background()

	// 会话1：心跳时间1000，已超时
	sess1 := domainsession.NewOnlineSession(1001, "conn-001", 1000, 1, "device-001")
	require.NoError(t, repo.Save(ctx, sess1))

	// 会话2：心跳时间2000，已超时
	sess2 := domainsession.NewOnlineSession(1002, "conn-002", 2000, 1, "device-002")
	require.NoError(t, repo.Save(ctx, sess2))

	// 会话3：心跳时间3000，未超时
	sess3 := domainsession.NewOnlineSession(1003, "conn-003", 3000, 1, "device-003")
	require.NoError(t, repo.Save(ctx, sess3))

	// threshold=2500，应返回心跳<2500的会话（sess1和sess2）
	expired, err := repo.FindExpired(ctx, 2500, 100)
	require.NoError(t, err)
	assert.Len(t, expired, 2)

	playerIDs := map[int64]bool{}
	for _, s := range expired {
		playerIDs[s.PlayerID()] = true
	}
	assert.True(t, playerIDs[1001])
	assert.True(t, playerIDs[1002])
	assert.False(t, playerIDs[1003])
}

// TestSessionRepoMemory_FindExpiredWithLimit 测试超时会话查询limit参数生效。
func TestSessionRepoMemory_FindExpiredWithLimit(t *testing.T) {
	logger := zap.NewNop()
	repo := NewSessionRepoMemory(5*time.Minute, logger)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		sess := domainsession.NewOnlineSession(int64(1000+i), "conn", 1000, 1, "device")
		require.NoError(t, repo.Save(ctx, sess))
	}

	// limit=2，应最多返回2条
	expired, err := repo.FindExpired(ctx, 2000, 2)
	require.NoError(t, err)
	assert.Len(t, expired, 2)
}

// TestSessionRepoMemory_Overwrite 测试同一玩家ID会话保存覆盖旧值。
func TestSessionRepoMemory_Overwrite(t *testing.T) {
	logger := zap.NewNop()
	repo := NewSessionRepoMemory(5*time.Minute, logger)
	ctx := context.Background()

	sess1 := domainsession.NewOnlineSession(1001, "conn-old", 1700000000000, 1, "device-old")
	require.NoError(t, repo.Save(ctx, sess1))

	sess2 := domainsession.NewOnlineSession(1001, "conn-new", 1700000001000, 2, "device-new")
	require.NoError(t, repo.Save(ctx, sess2))

	found, err := repo.FindByPlayerID(ctx, 1001)
	require.NoError(t, err)
	assert.Equal(t, "conn-new", found.ConnID())
	assert.Equal(t, 2, found.TokenVersion())
	assert.Equal(t, "device-new", found.DeviceID())
}

// 确保 SessionRepoMemory 实现 SessionRepository 接口（编译期检查）。
var _ domainsession.SessionRepository = (*SessionRepoMemory)(nil)
