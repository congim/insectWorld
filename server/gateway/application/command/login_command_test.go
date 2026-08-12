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
	domainsecurity "insectworld/server/gateway/domain/security"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"

	"go.uber.org/zap"
)

// newTestLoginCommand 构造测试用LoginCommand实例。
func newTestLoginCommand(
	accountRepo *mockAccountRepo,
	sessionRepo *mockSessionRepo,
	rateLimiter *mockRateLimiter,
	bruteForce *domainsecurity.BruteForceProtector,
	hasher *mockHasher,
	tokenSigner *mockTokenSigner,
	eventBus *mockEventBus,
	auditLogger *mockAuditLogger,
	connManager *mockConnManager,
	cfg domainconfig.AuthConfig,
) *LoginCommand {
	logger := zap.NewNop()
	return NewLoginCommand(
		accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg, logger,
	)
}

// newNormalAccount 构造一个正常状态的账号聚合根。
func newNormalAccount() *domainaccount.PlayerAccount {
	return domainaccount.NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
}

// newBannedAccount 构造一个已封禁的账号聚合根。
func newBannedAccount() *domainaccount.PlayerAccount {
	a := newNormalAccount()
	_ = a.Ban("违规", 0)
	return a
}

// TestLoginCommand_Success 测试登录成功全流程（无旧会话需踢下线）。
func TestLoginCommand_Success(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameAcc: newNormalAccount()}
	sessionRepo := &mockSessionRepo{findByPlayerErr: gatewayerr.ErrSessionNotFound} // 无旧会话
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}               // IP和账号均允许
	tracker := &mockLoginFailureTracker{isLockedResult: false}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{verifyResult: true}
	tokenSigner := &mockTokenSigner{signResult: "token-str"}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	resp, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
		DeviceID: "device-001",
		ConnID:   "conn-001",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "token-str", resp.AccessToken)
	assert.Equal(t, int64(1001), resp.PlayerID)
	assert.Equal(t, cfg.SessionTTLMs, resp.SessionTTLms)

	// 校验会话保存
	assert.Equal(t, 1, sessionRepo.saveCallCount, "应保存新会话")
	saved := sessionRepo.lastSavedSession
	require.NotNil(t, saved)
	assert.Equal(t, int64(1001), saved.PlayerID())
	assert.Equal(t, "conn-001", saved.ConnID())
	assert.Equal(t, "device-001", saved.DeviceID())

	// 校验登录成功清零失败计数
	assert.True(t, tracker.resetCalled, "登录成功应清零失败计数")

	// 校验审计日志
	assert.Equal(t, domainaudit.OpTypeLoginSuccess, auditLogger.lastRecord.OpType)
	assert.True(t, auditLogger.lastRecord.Result)

	// 校验上线事件发布
	assert.Equal(t, 1, eventBus.publishCount, "应发布上线事件")
}

// TestLoginCommand_IPRateLimited 测试IP限流拒绝。
func TestLoginCommand_IPRateLimited(t *testing.T) {
	accountRepo := &mockAccountRepo{}
	sessionRepo := &mockSessionRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{false}} // IP限流拒绝
	tracker := &mockLoginFailureTracker{}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{}
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrLoginRateLimited))
	assert.Equal(t, 0, accountRepo.findByUsernameCallCount, "限流拒绝不应查询账号")
}

// TestLoginCommand_AccountRateLimited 测试账号限流拒绝。
func TestLoginCommand_AccountRateLimited(t *testing.T) {
	accountRepo := &mockAccountRepo{}
	sessionRepo := &mockSessionRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, false}} // IP允许，账号拒绝
	tracker := &mockLoginFailureTracker{}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{}
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrLoginRateLimited))
	assert.Equal(t, 0, accountRepo.findByUsernameCallCount)
}

// TestLoginCommand_AccountNotFound 测试账号不存在。
func TestLoginCommand_AccountNotFound(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameErr: gatewayerr.ErrAccountNotFoundSentinel}
	sessionRepo := &mockSessionRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{}
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountNotFound))
}

// TestLoginCommand_AccountQueryError 测试账号查询故障。
func TestLoginCommand_AccountQueryError(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameErr: gatewayerr.ErrAccountRepoUnavailable}
	sessionRepo := &mockSessionRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{}
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrLoginInternalError))
}

// TestLoginCommand_AccountBanned 测试账号已封禁。
func TestLoginCommand_AccountBanned(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameAcc: newBannedAccount()}
	sessionRepo := &mockSessionRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{}
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountBanned))
	// 封禁拦截应记录审计日志
	assert.Equal(t, 1, auditLogger.logCallCount)
	assert.Equal(t, domainaudit.OpTypeBanIntercept, auditLogger.lastRecord.OpType)
}

// TestLoginCommand_AccountLocked 测试账号已锁定。
func TestLoginCommand_AccountLocked(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameAcc: newNormalAccount()}
	sessionRepo := &mockSessionRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{isLockedResult: true, remainingResult: 600}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{}
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountLocked))
}

// TestLoginCommand_PasswordIncorrect 测试密码错误，应记录失败计数。
func TestLoginCommand_PasswordIncorrect(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameAcc: newNormalAccount()}
	sessionRepo := &mockSessionRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{isLockedResult: false, recordFailCount: 1}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{verifyResult: false} // 密码不匹配
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "wrongpassword", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrPasswordIncorrect))
	// 密码错误应记录失败计数
	assert.True(t, tracker.recordFailCalled, "密码错误应调用失败计数器")
	// 应记录登录失败审计日志
	assert.Equal(t, 1, auditLogger.logCallCount)
	assert.Equal(t, domainaudit.OpTypeLoginFailure, auditLogger.lastRecord.OpType)
}

// TestLoginCommand_PasswordVerifyError 测试密码校验故障。
func TestLoginCommand_PasswordVerifyError(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameAcc: newNormalAccount()}
	sessionRepo := &mockSessionRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{isLockedResult: false}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{verifyErr: gatewayerr.ErrPasswordHashFailed}
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrLoginInternalError))
}

// TestLoginCommand_SessionSaveError 测试会话保存故障。
func TestLoginCommand_SessionSaveError(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameAcc: newNormalAccount()}
	sessionRepo := &mockSessionRepo{saveErr: gatewayerr.ErrSessionRepoUnavailable, findByPlayerErr: gatewayerr.ErrSessionNotFound}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{isLockedResult: false}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{verifyResult: true}
	tokenSigner := &mockTokenSigner{}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrLoginInternalError))
}

// TestLoginCommand_TokenSignError 测试令牌签发故障。
func TestLoginCommand_TokenSignError(t *testing.T) {
	accountRepo := &mockAccountRepo{findByUsernameAcc: newNormalAccount()}
	sessionRepo := &mockSessionRepo{findByPlayerErr: gatewayerr.ErrSessionNotFound}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{isLockedResult: false}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{verifyResult: true}
	tokenSigner := &mockTokenSigner{signErr: gatewayerr.ErrTokenSignerUnavailable}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrLoginInternalError))
}

// TestLoginCommand_SingleLoginKickOut 测试单点登录踢下线。
func TestLoginCommand_SingleLoginKickOut(t *testing.T) {
	oldSession := domainsession.NewOnlineSession(1001, "conn-old", 1700000000000, 1, "device-old")
	accountRepo := &mockAccountRepo{findByUsernameAcc: newNormalAccount()}
	sessionRepo := &mockSessionRepo{findByPlayerSess: oldSession}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{isLockedResult: false}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{verifyResult: true}
	tokenSigner := &mockTokenSigner{signResult: "token-str"}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig() // SingleLoginEnabled=true

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	resp, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
		DeviceID: "device-new", ConnID: "conn-new",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	// 应删除旧会话
	assert.Equal(t, 1, sessionRepo.deleteCallCount, "应删除旧会话")
	// 应推送踢下线通知
	assert.Equal(t, 1, connManager.sendCount, "应推送踢下线通知")
	assert.Equal(t, int64(1001), connManager.lastPlayer)
	// 应发布下线事件（踢下线）+上线事件 = 2个事件
	assert.Equal(t, 2, eventBus.publishCount, "应发布下线+上线两个事件")
}

// TestLoginCommand_SingleLoginDisabled 测试单点登录关闭时不踢下线。
func TestLoginCommand_SingleLoginDisabled(t *testing.T) {
	oldSession := domainsession.NewOnlineSession(1001, "conn-old", 1700000000000, 1, "device-old")
	accountRepo := &mockAccountRepo{findByUsernameAcc: newNormalAccount()}
	sessionRepo := &mockSessionRepo{findByPlayerSess: oldSession}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true, true}}
	tracker := &mockLoginFailureTracker{isLockedResult: false}
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	hasher := &mockHasher{verifyResult: true}
	tokenSigner := &mockTokenSigner{signResult: "token-str"}
	eventBus := &mockEventBus{}
	auditLogger := &mockAuditLogger{}
	connManager := &mockConnManager{}
	cfg := domainconfig.DefaultAuthConfig()
	cfg.SingleLoginEnabled = false // 关闭单点登录

	cmd := newTestLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		tokenSigner, eventBus, auditLogger, connManager, cfg)
	_, err := cmd.Handle(context.Background(), LoginRequest{
		Username: "testuser", Password: "password123", SourceIP: "127.0.0.1",
	})

	require.NoError(t, err)
	// 不踢下线
	assert.Equal(t, 0, sessionRepo.deleteCallCount, "单点登录关闭不应删除旧会话")
	assert.Equal(t, 0, connManager.sendCount, "单点登录关闭不应推送踢下线")
	// 仅发布上线事件
	assert.Equal(t, 1, eventBus.publishCount)
}

// 确保domaintoken包被引用。
var _ = domaintoken.TokenPayload{}
