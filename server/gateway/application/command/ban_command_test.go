package command

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainaccount "insectworld/server/gateway/domain/account"
	domainaudit "insectworld/server/gateway/domain/audit"
	domainconfig "insectworld/server/gateway/domain/config"
	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"

	"go.uber.org/zap"
)

// newTestBanCommand 构造测试用BanCommand实例。
func newTestBanCommand(
	accountRepo *mockAccountRepo,
	sessionRepo *mockSessionRepo,
	tokenBlacklist *mockTokenBlacklist,
	eventBus *mockEventBus,
	auditLogger *mockAuditLogger,
	connManager *mockConnManager,
	cfg domainconfig.AuthConfig,
) *BanCommand {
	logger := zap.NewNop()
	return NewBanCommand(accountRepo, sessionRepo, tokenBlacklist, eventBus, auditLogger, connManager, cfg, logger)
}

// TestBanCommand_Success 测试封禁成功（玩家不在线）。
func TestBanCommand_Success(t *testing.T) {
	accountRepo := &mockAccountRepo{findByIDAccount: domainaccount.NewPlayerAccount(1001, "user", "hash", "salt", "127.0.0.1", 1700000000000)}
	sessionRepo := &mockSessionRepo{findByPlayerErr: gatewayerr.ErrSessionNotFound} // 不在线
	tokenBlacklist := &mockTokenBlacklist{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestBanCommand(accountRepo, sessionRepo, tokenBlacklist, eventBus, auditLogger, connManager, cfg)
	err := cmd.Handle(context.Background(), 1001, 3600000, "违规", "admin-001")

	require.NoError(t, err)
	// 应持久化封禁状态
	assert.Equal(t, 1, accountRepo.saveCallCount)
	saved := accountRepo.lastSavedAccount
	require.NotNil(t, saved)
	assert.Equal(t, domainaccount.AccountStatusBanned, saved.Status())
	assert.Equal(t, "违规", saved.BanReason())
	// 应记录审计日志
	assert.Equal(t, 1, auditLogger.logCallCount)
	assert.Equal(t, domainaudit.OpTypeBanIntercept, auditLogger.lastRecord.OpType)
	// 不在线不应踢下线
	assert.Equal(t, 0, connManager.sendCount)
}

// TestBanCommand_SuccessOnline 测试封禁成功且玩家在线，应踢下线。
func TestBanCommand_SuccessOnline(t *testing.T) {
	accountRepo := &mockAccountRepo{findByIDAccount: domainaccount.NewPlayerAccount(1001, "user", "hash", "salt", "127.0.0.1", 1700000000000)}
	sessionRepo := &mockSessionRepo{
		findByPlayerSess: domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device"),
	}
	tokenBlacklist := &mockTokenBlacklist{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestBanCommand(accountRepo, sessionRepo, tokenBlacklist, eventBus, auditLogger, connManager, cfg)
	err := cmd.Handle(context.Background(), 1001, 0, "严重违规", "admin-001")

	require.NoError(t, err)
	// 在线应踢下线
	assert.Equal(t, 1, sessionRepo.deleteCallCount, "应删除会话")
	assert.Equal(t, 1, tokenBlacklist.invalidateCallCount, "应令牌失效")
	assert.Equal(t, 1, connManager.sendCount, "应推送踢下线通知")
	assert.Equal(t, 1, eventBus.publishCount, "应发布下线事件")
}

// TestBanCommand_AccountNotFound 测试账号不存在。
func TestBanCommand_AccountNotFound(t *testing.T) {
	accountRepo := &mockAccountRepo{findByIDErr: gatewayerr.ErrAccountNotFoundSentinel}
	sessionRepo := &mockSessionRepo{}
	tokenBlacklist := &mockTokenBlacklist{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestBanCommand(accountRepo, sessionRepo, tokenBlacklist, eventBus, auditLogger, connManager, cfg)
	err := cmd.Handle(context.Background(), 9999, 3600000, "违规", "admin-001")

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountNotFound))
}

// TestBanCommand_AccountQueryError 测试账号查询故障。
func TestBanCommand_AccountQueryError(t *testing.T) {
	accountRepo := &mockAccountRepo{findByIDErr: gatewayerr.ErrAccountRepoUnavailable}
	sessionRepo := &mockSessionRepo{}
	tokenBlacklist := &mockTokenBlacklist{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestBanCommand(accountRepo, sessionRepo, tokenBlacklist, eventBus, auditLogger, connManager, cfg)
	err := cmd.Handle(context.Background(), 1001, 3600000, "违规", "admin-001")

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountRepoUnavailable))
}

// TestBanCommand_SaveError 测试封禁状态持久化故障。
func TestBanCommand_SaveError(t *testing.T) {
	accountRepo := &mockAccountRepo{
		findByIDAccount: domainaccount.NewPlayerAccount(1001, "user", "hash", "salt", "127.0.0.1", 1700000000000),
		saveErr:         gatewayerr.ErrAccountRepoUnavailable,
	}
	sessionRepo := &mockSessionRepo{}
	tokenBlacklist := &mockTokenBlacklist{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestBanCommand(accountRepo, sessionRepo, tokenBlacklist, eventBus, auditLogger, connManager, cfg)
	err := cmd.Handle(context.Background(), 1001, 3600000, "违规", "admin-001")

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountRepoUnavailable))
}
