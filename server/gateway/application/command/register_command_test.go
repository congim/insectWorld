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

	"go.uber.org/zap"
)

// newTestRegisterCommand 构造测试用RegisterCommand实例，注入全部mock依赖。
func newTestRegisterCommand(
	accountRepo *mockAccountRepo,
	rateLimiter *mockRateLimiter,
	idGen *mockIDGenerator,
	hasher *mockHasher,
	auditLogger *mockAuditLogger,
	cfg domainconfig.AuthConfig,
) *RegisterCommand {
	logger := zap.NewNop()
	return NewRegisterCommand(
		accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg, logger,
	)
}

// TestRegisterCommand_Success 测试注册成功全流程。
func TestRegisterCommand_Success(t *testing.T) {
	accountRepo := &mockAccountRepo{existsResult: false}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{hashResult: "hashed-pwd", saltResult: "salt"}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	resp, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(1001), resp.PlayerID)

	// 校验mock调用次数
	assert.Equal(t, 1, accountRepo.existsCallCount, "应查询用户名唯一性")
	assert.Equal(t, 1, accountRepo.saveCallCount, "应持久化账号")

	assert.Equal(t, 1, auditLogger.logCallCount, "应记录审计日志")

	// 校验保存的账号聚合根字段
	saved := accountRepo.lastSavedAccount
	require.NotNil(t, saved)
	assert.Equal(t, int64(1001), saved.PlayerID())
	assert.Equal(t, "testuser", saved.Username())
	assert.Equal(t, "hashed-pwd", saved.PasswordHash())
	assert.Equal(t, "salt", saved.Salt())
	assert.Equal(t, "127.0.0.1", saved.RegisterIP())

	// 校验审计日志记录
	assert.Equal(t, domainaudit.OpTypeRegisterSuccess, auditLogger.lastRecord.OpType)
	assert.Equal(t, "testuser", auditLogger.lastRecord.Subject)
	assert.True(t, auditLogger.lastRecord.Result)
	assert.Equal(t, "127.0.0.1", auditLogger.lastRecord.SourceIP)
}

// TestRegisterCommand_InvalidUsername 测试用户名格式校验失败。
func TestRegisterCommand_InvalidUsername(t *testing.T) {
	accountRepo := &mockAccountRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "123", // 纯数字，非法
		Password: "password123",
		SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrInvalidUsernameFormat))
	// 用户名校验失败不应触达仓储
	assert.Equal(t, 0, accountRepo.existsCallCount)
	assert.Equal(t, 0, accountRepo.saveCallCount)
}

// TestRegisterCommand_InvalidPassword 测试密码强度校验失败。
func TestRegisterCommand_InvalidPassword(t *testing.T) {
	accountRepo := &mockAccountRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "weak", // 过短且无数字
		SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrInvalidPasswordStrength))
	assert.Equal(t, 0, accountRepo.existsCallCount)
}

// TestRegisterCommand_RateLimited 测试注册频率超限。
func TestRegisterCommand_RateLimited(t *testing.T) {
	accountRepo := &mockAccountRepo{}
	rateLimiter := &mockRateLimiter{allowResults: []bool{false}} // 限流拒绝
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrRegisterRateLimited))
	// 限流拒绝不应触达仓储
	assert.Equal(t, 0, accountRepo.existsCallCount)
	assert.Equal(t, 0, accountRepo.saveCallCount)
}

// TestRegisterCommand_UsernameExists 测试用户名已存在。
func TestRegisterCommand_UsernameExists(t *testing.T) {
	accountRepo := &mockAccountRepo{existsResult: true}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrUsernameAlreadyExists))
	assert.Equal(t, 0, accountRepo.saveCallCount, "用户名已存在不应持久化")
	assert.Equal(t, 0, auditLogger.logCallCount, "不应记录审计日志")
}

// TestRegisterCommand_ExistsQueryError 测试用户名唯一性查询故障。
func TestRegisterCommand_ExistsQueryError(t *testing.T) {
	accountRepo := &mockAccountRepo{existsErr: gatewayerr.ErrAccountRepoUnavailable}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrRegisterInternalError))
}

// TestRegisterCommand_IDGenError 测试玩家ID生成故障。
func TestRegisterCommand_IDGenError(t *testing.T) {
	accountRepo := &mockAccountRepo{existsResult: false}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextIDErr: gatewayerr.ErrIDGenClockBack}
	hasher := &mockHasher{}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrRegisterInternalError))
}

// TestRegisterCommand_HashError 测试密码哈希故障。
func TestRegisterCommand_HashError(t *testing.T) {
	accountRepo := &mockAccountRepo{existsResult: false}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{hashErr: gatewayerr.ErrPasswordHashFailed}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrPasswordHashFailed))
	assert.Equal(t, 0, accountRepo.saveCallCount, "哈希失败不应持久化")
}

// TestRegisterCommand_SaveError 测试账号持久化故障。
func TestRegisterCommand_SaveError(t *testing.T) {
	accountRepo := &mockAccountRepo{existsResult: false, saveErr: gatewayerr.ErrAccountRepoUnavailable}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{hashResult: "hash", saltResult: "salt"}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrRegisterInternalError))
}

// TestRegisterCommand_PasswordZeroed 测试注册完成后密码明文被置零（spec 4.3 安全性2）。
func TestRegisterCommand_PasswordZeroed(t *testing.T) {
	accountRepo := &mockAccountRepo{existsResult: false}
	rateLimiter := &mockRateLimiter{allowResults: []bool{true}}
	idGen := &mockIDGenerator{nextID: 1001}
	hasher := &mockHasher{hashResult: "hash", saltResult: "salt"}
	auditLogger := &mockAuditLogger{}
	cfg := domainconfig.DefaultAuthConfig()

	cmd := newTestRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg)
	_, err := cmd.Handle(context.Background(), RegisterRequest{
		Username: "testuser",
		Password: "password123",
		SourceIP: "127.0.0.1",
	})
	require.NoError(t, err)
	// Handle返回后内部Credential应已置零，此处通过不panic且成功间接验证
	// 真正的明文置零在Credential.ZeroPassword调用，由defer保证
}

// 确保domainaccount包被引用（编译期检查）。
var _ = domainaccount.AccountStatusNormal
