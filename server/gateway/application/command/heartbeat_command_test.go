package command

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"

	"go.uber.org/zap"
)

// newTestHeartbeatCommand 构造测试用HeartbeatCommand实例。
func newTestHeartbeatCommand(
	tokenSigner *mockTokenSigner,
	sessionRepo *mockSessionRepo,
) *HeartbeatCommand {
	logger := zap.NewNop()
	return NewHeartbeatCommand(tokenSigner, sessionRepo, logger)
}

// TestHeartbeatCommand_Success 测试心跳更新成功。
func TestHeartbeatCommand_Success(t *testing.T) {
	tokenSigner := &mockTokenSigner{verifyResult: domaintoken.TokenPayload{PlayerID: 1001}}
	sessionRepo := &mockSessionRepo{
		findByPlayerSess: domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device"),
	}

	cmd := newTestHeartbeatCommand(tokenSigner, sessionRepo)
	err := cmd.Handle(context.Background(), HeartbeatRequest{AccessToken: "valid", PlayerID: 1001})

	require.NoError(t, err)
	assert.Equal(t, 1, sessionRepo.saveCallCount, "应保存更新后的会话")
}

// TestHeartbeatCommand_InvalidToken 测试令牌无效返回ErrTokenInvalid。
func TestHeartbeatCommand_InvalidToken(t *testing.T) {
	tokenSigner := &mockTokenSigner{verifyErr: gatewayerr.ErrTokenInvalid}
	sessionRepo := &mockSessionRepo{}

	cmd := newTestHeartbeatCommand(tokenSigner, sessionRepo)
	err := cmd.Handle(context.Background(), HeartbeatRequest{AccessToken: "invalid", PlayerID: 1001})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenInvalid))
	assert.Equal(t, 0, sessionRepo.findByPlayerCallCount, "令牌无效不应查询会话")
}

// TestHeartbeatCommand_SessionNotFound 测试会话不存在返回ErrNotLoggedIn。
func TestHeartbeatCommand_SessionNotFound(t *testing.T) {
	tokenSigner := &mockTokenSigner{verifyResult: domaintoken.TokenPayload{PlayerID: 1001}}
	sessionRepo := &mockSessionRepo{findByPlayerErr: gatewayerr.ErrSessionNotFound}

	cmd := newTestHeartbeatCommand(tokenSigner, sessionRepo)
	err := cmd.Handle(context.Background(), HeartbeatRequest{AccessToken: "valid", PlayerID: 1001})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrNotLoggedIn))
}

// TestHeartbeatCommand_SessionQueryError 测试会话查询故障返回错误。
func TestHeartbeatCommand_SessionQueryError(t *testing.T) {
	tokenSigner := &mockTokenSigner{verifyResult: domaintoken.TokenPayload{PlayerID: 1001}}
	sessionRepo := &mockSessionRepo{findByPlayerErr: gatewayerr.ErrSessionRepoUnavailable}

	cmd := newTestHeartbeatCommand(tokenSigner, sessionRepo)
	err := cmd.Handle(context.Background(), HeartbeatRequest{AccessToken: "valid", PlayerID: 1001})

	require.Error(t, err)
}

// TestHeartbeatCommand_SaveError 测试会话保存故障返回错误。
func TestHeartbeatCommand_SaveError(t *testing.T) {
	tokenSigner := &mockTokenSigner{verifyResult: domaintoken.TokenPayload{PlayerID: 1001}}
	sessionRepo := &mockSessionRepo{
		findByPlayerSess: domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device"),
		saveErr:          gatewayerr.ErrSessionRepoUnavailable,
	}

	cmd := newTestHeartbeatCommand(tokenSigner, sessionRepo)
	err := cmd.Handle(context.Background(), HeartbeatRequest{AccessToken: "valid", PlayerID: 1001})

	require.Error(t, err)
}
