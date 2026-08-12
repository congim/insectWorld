package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainaudit "insectworld/server/gateway/domain/audit"
	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"

	"go.uber.org/zap"
)

// futureExpireTime 返回未来过期时间戳，供登出测试使用。
func futureExpireTime() int64 {
	return time.Now().UnixMilli() + 3600000
}

// newTestLogoutCommand 构造测试用LogoutCommand实例。
func newTestLogoutCommand(
	tokenSigner *mockTokenSigner,
	tokenBlacklist *mockTokenBlacklist,
	sessionRepo *mockSessionRepo,
	eventBus *mockEventBus,
	auditLogger *mockAuditLogger,
) *LogoutCommand {
	logger := zap.NewNop()
	return NewLogoutCommand(tokenSigner, tokenBlacklist, sessionRepo, eventBus, auditLogger, logger)
}

// TestLogoutCommand_Success 测试登出成功全流程。
func TestLogoutCommand_Success(t *testing.T) {
	tokenSigner := &mockTokenSigner{
		verifyResult: domaintoken.TokenPayload{PlayerID: 1001, IssueTime: 1700000000000, ExpireTime: futureExpireTime(), Version: 1},
	}
	tokenBlacklist := &mockTokenBlacklist{}
	sessionRepo := &mockSessionRepo{findByPlayerSess: domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device")}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}

	cmd := newTestLogoutCommand(tokenSigner, tokenBlacklist, sessionRepo, eventBus, auditLogger)
	resp, err := cmd.Handle(context.Background(), LogoutRequest{AccessToken: "valid-token", PlayerID: 1001})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)

	// 应删除会话
	assert.Equal(t, 1, sessionRepo.deleteCallCount)
	// 应将令牌加入黑名单
	assert.Equal(t, 1, tokenBlacklist.invalidateCallCount)
	// 应发布下线事件
	assert.Equal(t, 1, eventBus.publishCount)
	// 应记录审计日志
	assert.Equal(t, 1, auditLogger.logCallCount)
	assert.Equal(t, domainaudit.OpTypeLogout, auditLogger.lastRecord.OpType)
}

// TestLogoutCommand_InvalidTokenIdempotent 测试令牌无效幂等返回成功。
func TestLogoutCommand_InvalidTokenIdempotent(t *testing.T) {
	tokenSigner := &mockTokenSigner{verifyErr: gatewayerr.ErrTokenInvalid}
	tokenBlacklist := &mockTokenBlacklist{}
	sessionRepo := &mockSessionRepo{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}

	cmd := newTestLogoutCommand(tokenSigner, tokenBlacklist, sessionRepo, eventBus, auditLogger)
	resp, err := cmd.Handle(context.Background(), LogoutRequest{AccessToken: "invalid", PlayerID: 1001})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success, "令牌无效应幂等返回成功")
	// 不应触达会话仓储
	assert.Equal(t, 0, sessionRepo.deleteCallCount)
	assert.Equal(t, 0, tokenBlacklist.invalidateCallCount)
}

// TestLogoutCommand_SessionNotFoundIdempotent 测试会话不存在幂等返回成功。
func TestLogoutCommand_SessionNotFoundIdempotent(t *testing.T) {
	tokenSigner := &mockTokenSigner{
		verifyResult: domaintoken.TokenPayload{PlayerID: 1001, ExpireTime: futureExpireTime()},
	}
	tokenBlacklist := &mockTokenBlacklist{}
	sessionRepo := &mockSessionRepo{findByPlayerErr: gatewayerr.ErrSessionNotFound}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}

	cmd := newTestLogoutCommand(tokenSigner, tokenBlacklist, sessionRepo, eventBus, auditLogger)
	resp, err := cmd.Handle(context.Background(), LogoutRequest{AccessToken: "valid", PlayerID: 1001})

	require.NoError(t, err)
	assert.True(t, resp.Success, "会话不存在应幂等返回成功")
	assert.Equal(t, 0, sessionRepo.deleteCallCount)
}

// TestLogoutCommand_SessionQueryError 测试会话查询故障返回错误。
func TestLogoutCommand_SessionQueryError(t *testing.T) {
	tokenSigner := &mockTokenSigner{
		verifyResult: domaintoken.TokenPayload{PlayerID: 1001, ExpireTime: futureExpireTime()},
	}
	tokenBlacklist := &mockTokenBlacklist{}
	sessionRepo := &mockSessionRepo{findByPlayerErr: gatewayerr.ErrSessionRepoUnavailable}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}

	cmd := newTestLogoutCommand(tokenSigner, tokenBlacklist, sessionRepo, eventBus, auditLogger)
	_, err := cmd.Handle(context.Background(), LogoutRequest{AccessToken: "valid", PlayerID: 1001})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrLogoutInternalError))
}

// TestLogoutCommand_DeleteError 测试会话删除故障返回错误。
func TestLogoutCommand_DeleteError(t *testing.T) {
	tokenSigner := &mockTokenSigner{
		verifyResult: domaintoken.TokenPayload{PlayerID: 1001, ExpireTime: futureExpireTime()},
	}
	tokenBlacklist := &mockTokenBlacklist{}
	sessionRepo := &mockSessionRepo{
		findByPlayerSess: domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device"),
		deleteErr:        gatewayerr.ErrSessionRepoUnavailable,
	}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}

	cmd := newTestLogoutCommand(tokenSigner, tokenBlacklist, sessionRepo, eventBus, auditLogger)
	_, err := cmd.Handle(context.Background(), LogoutRequest{AccessToken: "valid", PlayerID: 1001})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrLogoutInternalError))
}
