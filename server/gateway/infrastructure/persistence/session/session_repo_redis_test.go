package session

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"

	"go.uber.org/zap"
)

// newRedisTestClient 启动miniredis并返回客户端与清理函数。
func newRedisTestClient(t *testing.T) (*redis.Client, *miniredis.Miniredis, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, mr, func() {
		_ = client.Close()
		mr.Close()
	}
}

// TestSessionRepoRedis_SaveAndFind 测试Redis会话仓储保存与查询往返。
func TestSessionRepoRedis_SaveAndFind(t *testing.T) {
	client, _, cleanup := newRedisTestClient(t)
	defer cleanup()
	logger := zap.NewNop()
	repo := NewSessionRepoRedis(client, 300000, logger)
	ctx := context.Background()

	sess := domainsession.NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
	require.NoError(t, repo.Save(ctx, sess))

	found, err := repo.FindByPlayerID(ctx, 1001)
	require.NoError(t, err)
	assert.Equal(t, int64(1001), found.PlayerID())
	assert.Equal(t, "conn-001", found.ConnID())
	assert.Equal(t, "device-001", found.DeviceID())
	assert.Equal(t, 1, found.TokenVersion())
}

// TestSessionRepoRedis_FindNotFound 测试查询不存在的会话返回ErrSessionNotFound。
func TestSessionRepoRedis_FindNotFound(t *testing.T) {
	client, _, cleanup := newRedisTestClient(t)
	defer cleanup()
	logger := zap.NewNop()
	repo := NewSessionRepoRedis(client, 300000, logger)

	_, err := repo.FindByPlayerID(context.Background(), 9999)
	require.Error(t, err)
	assert.Equal(t, gatewayerr.ErrSessionNotFound, err)
}

// TestSessionRepoRedis_Delete 测试会话删除。
func TestSessionRepoRedis_Delete(t *testing.T) {
	client, _, cleanup := newRedisTestClient(t)
	defer cleanup()
	logger := zap.NewNop()
	repo := NewSessionRepoRedis(client, 300000, logger)
	ctx := context.Background()

	sess := domainsession.NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
	require.NoError(t, repo.Save(ctx, sess))
	require.NoError(t, repo.Delete(ctx, 1001))

	_, err := repo.FindByPlayerID(ctx, 1001)
	assert.Equal(t, gatewayerr.ErrSessionNotFound, err)
}

// TestSessionRepoRedis_FindExpired 测试超时会话查询。
func TestSessionRepoRedis_FindExpired(t *testing.T) {
	client, _, cleanup := newRedisTestClient(t)
	defer cleanup()
	logger := zap.NewNop()
	repo := NewSessionRepoRedis(client, 300000, logger)
	ctx := context.Background()

	// 会话1：心跳时间1000，已超时
	sess1 := domainsession.NewOnlineSession(1001, "conn-001", 1000, 1, "device-001")
	require.NoError(t, repo.Save(ctx, sess1))
	// 会话2：心跳时间3000，未超时
	sess2 := domainsession.NewOnlineSession(1002, "conn-002", 3000, 1, "device-002")
	require.NoError(t, repo.Save(ctx, sess2))

	expired, err := repo.FindExpired(ctx, 2500, 100)
	require.NoError(t, err)
	assert.Len(t, expired, 1)
	assert.Equal(t, int64(1001), expired[0].PlayerID())
}

// TestSessionRepoRedis_Overwrite 测试同一玩家会话保存覆盖旧值。
func TestSessionRepoRedis_Overwrite(t *testing.T) {
	client, _, cleanup := newRedisTestClient(t)
	defer cleanup()
	logger := zap.NewNop()
	repo := NewSessionRepoRedis(client, 300000, logger)
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, domainsession.NewOnlineSession(1001, "conn-old", 1700000000000, 1, "device-old")))
	require.NoError(t, repo.Save(ctx, domainsession.NewOnlineSession(1001, "conn-new", 1700000001000, 2, "device-new")))

	found, err := repo.FindByPlayerID(ctx, 1001)
	require.NoError(t, err)
	assert.Equal(t, "conn-new", found.ConnID())
	assert.Equal(t, 2, found.TokenVersion())
}

// TestSessionRepoRedis_SaveError 测试Redis故障时保存返回错误。
func TestSessionRepoRedis_SaveError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	logger := zap.NewNop()
	repo := NewSessionRepoRedis(client, 300000, logger)

	mr.Close() // 模拟故障

	sess := domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device")
	err = repo.Save(context.Background(), sess)
	require.Error(t, err)
}

// TestSessionRepoRedis_DeleteError 测试Redis故障时删除返回错误。
func TestSessionRepoRedis_DeleteError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	logger := zap.NewNop()
	repo := NewSessionRepoRedis(client, 300000, logger)

	mr.Close()

	err = repo.Delete(context.Background(), 1001)
	require.Error(t, err)
}
