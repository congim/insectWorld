package command

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainaccount "insectworld/server/gateway/domain/account"
	domainaudit "insectworld/server/gateway/domain/audit"
	gatewayerr "insectworld/server/gateway/domain/errors"

	"go.uber.org/zap"
)

// newTestUnbanCommand 构造测试用UnbanCommand实例。
func newTestUnbanCommand(
	accountRepo *mockAccountRepo,
	auditLogger *mockAuditLogger,
) *UnbanCommand {
	logger := zap.NewNop()
	return NewUnbanCommand(accountRepo, auditLogger, logger)
}

// TestUnbanCommand_Success 测试解封成功。
func TestUnbanCommand_Success(t *testing.T) {
	account := domainaccount.NewPlayerAccount(1001, "user", "hash", "salt", "127.0.0.1", 1700000000000)
	_ = account.Ban("违规", 0)
	accountRepo := &mockAccountRepo{findByIDAccount: account}
	auditLogger := &mockAuditLogger{}

	cmd := newTestUnbanCommand(accountRepo, auditLogger)
	err := cmd.Handle(context.Background(), 1001, "admin-001")

	require.NoError(t, err)
	assert.Equal(t, 1, accountRepo.saveCallCount)
	saved := accountRepo.lastSavedAccount
	require.NotNil(t, saved)
	assert.Equal(t, domainaccount.AccountStatusNormal, saved.Status())
	assert.Equal(t, "", saved.BanReason())
	assert.Equal(t, 1, auditLogger.logCallCount)
	assert.Equal(t, domainaudit.OpTypeBanIntercept, auditLogger.lastRecord.OpType)
}

// TestUnbanCommand_AccountNotFound 测试账号不存在。
func TestUnbanCommand_AccountNotFound(t *testing.T) {
	accountRepo := &mockAccountRepo{findByIDErr: gatewayerr.ErrAccountNotFoundSentinel}
	auditLogger := &mockAuditLogger{}

	cmd := newTestUnbanCommand(accountRepo, auditLogger)
	err := cmd.Handle(context.Background(), 9999, "admin-001")

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountNotFound))
}

// TestUnbanCommand_AccountQueryError 测试账号查询故障。
func TestUnbanCommand_AccountQueryError(t *testing.T) {
	accountRepo := &mockAccountRepo{findByIDErr: gatewayerr.ErrAccountRepoUnavailable}
	auditLogger := &mockAuditLogger{}

	cmd := newTestUnbanCommand(accountRepo, auditLogger)
	err := cmd.Handle(context.Background(), 1001, "admin-001")

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountRepoUnavailable))
}

// TestUnbanCommand_SaveError 测试解封状态持久化故障。
func TestUnbanCommand_SaveError(t *testing.T) {
	account := domainaccount.NewPlayerAccount(1001, "user", "hash", "salt", "127.0.0.1", 1700000000000)
	accountRepo := &mockAccountRepo{findByIDAccount: account, saveErr: gatewayerr.ErrAccountRepoUnavailable}
	auditLogger := &mockAuditLogger{}

	cmd := newTestUnbanCommand(accountRepo, auditLogger)
	err := cmd.Handle(context.Background(), 1001, "admin-001")

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountRepoUnavailable))
}
