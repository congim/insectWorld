package interceptor

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/gateway/application/query"
	domainconfig "insectworld/server/gateway/domain/config"
	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"
	infratoken "insectworld/server/gateway/infrastructure/auth/token"
	infraaudit "insectworld/server/gateway/infrastructure/persistence/session"

	"go.uber.org/zap"
)

// TestAuthInterceptor_Whitelist 测试白名单路径直接放行。
func TestAuthInterceptor_Whitelist(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewAuthInterceptor(nil, logger)

	tests := []string{"/auth/register", "/auth/login", "/auth/register?x=1", "/auth/login/sub"}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			playerID, _, err := interceptor.Intercept(context.Background(), path, "")
			require.NoError(t, err, "白名单路径应放行")
			assert.Equal(t, int64(0), playerID)
		})
	}
}

// TestAuthInterceptor_TokenMissing 测试非白名单路径令牌缺失返回ErrTokenMissing。
func TestAuthInterceptor_TokenMissing(t *testing.T) {
	logger := zap.NewNop()
	interceptor := NewAuthInterceptor(nil, logger)

	_, _, err := interceptor.Intercept(context.Background(), "/business/action", "")
	require.Error(t, err)
	assert.Equal(t, gatewayerr.ErrTokenMissing, err)
}

// TestAuthInterceptor_AuthSuccess 测试鉴权通过返回playerID并注入ctx。
func TestAuthInterceptor_AuthSuccess(t *testing.T) {
	logger := zap.NewNop()

	// 构造真实鉴权依赖
	signer, err := infratoken.NewTokenSignerImpl([]byte("test-key"), logger)
	require.NoError(t, err)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	blacklist := infratoken.NewTokenBlacklistImpl(client, logger)
	sessionRepo := infraaudit.NewSessionRepoMemory(5*time.Minute, logger)
	authQuery := query.NewAuthenticateQuery(signer, blacklist, sessionRepo, logger)

	// 签发有效令牌
	payload := domaintoken.TokenPayload{
		PlayerID:   1001,
		IssueTime:  time.Now().UnixMilli(),
		ExpireTime: time.Now().UnixMilli() + 3600000,
		Version:    1,
	}
	tokenStr, err := signer.Sign(context.Background(), payload)
	require.NoError(t, err)
	// 保存会话
	require.NoError(t, sessionRepo.Save(context.Background(),
		domainsession.NewOnlineSession(1001, "conn", payload.IssueTime, 1, "device")))

	interceptor := NewAuthInterceptor(authQuery, logger)
	playerID, newCtx, err := interceptor.Intercept(context.Background(), "/combat/attack", tokenStr)

	require.NoError(t, err)
	assert.Equal(t, int64(1001), playerID)
	// playerID应注入ctx
	ctxPlayerID, ok := PlayerIDFromContext(newCtx)
	require.True(t, ok)
	assert.Equal(t, int64(1001), ctxPlayerID)
}

// TestAuthInterceptor_AuthFail 测试鉴权失败返回错误。
func TestAuthInterceptor_AuthFail(t *testing.T) {
	logger := zap.NewNop()
	signer, err := infratoken.NewTokenSignerImpl([]byte("test-key"), logger)
	require.NoError(t, err)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	blacklist := infratoken.NewTokenBlacklistImpl(client, logger)
	sessionRepo := infraaudit.NewSessionRepoMemory(5*time.Minute, logger)
	authQuery := query.NewAuthenticateQuery(signer, blacklist, sessionRepo, logger)

	interceptor := NewAuthInterceptor(authQuery, logger)
	// 用无效令牌
	_, _, err = interceptor.Intercept(context.Background(), "/combat/attack", "invalid-token")
	require.Error(t, err)
	assert.Equal(t, gatewayerr.ErrTokenInvalid, err)
}

// TestPlayerIDFromContext 测试从context提取玩家ID。
func TestPlayerIDFromContext(t *testing.T) {
	t.Run("存在playerID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), playerIDKey{}, int64(1001))
		playerID, ok := PlayerIDFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, int64(1001), playerID)
	})

	t.Run("不存在playerID", func(t *testing.T) {
		_, ok := PlayerIDFromContext(context.Background())
		assert.False(t, ok)
	})

	t.Run("playerID类型不匹配", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), playerIDKey{}, "not-int64")
		_, ok := PlayerIDFromContext(ctx)
		assert.False(t, ok)
	})
}

// TestAuthWhitelist 测试白名单配置覆盖注册与登录路径。
func TestAuthWhitelist(t *testing.T) {
	assert.True(t, authWhitelist["/auth/register"])
	assert.True(t, authWhitelist["/auth/login"])
	assert.Len(t, authWhitelist, 2)
}

// 确保domainconfig包被引用（编译期检查）。
var _ = domainconfig.DefaultAuthConfig
