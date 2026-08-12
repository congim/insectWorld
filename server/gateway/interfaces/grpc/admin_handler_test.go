package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"
	adminpb "insectworld/server/shared/proto/admin"

	"go.uber.org/zap"
)

// mockAuthChecker 鉴权接口mock。
type mockAuthChecker struct {
	adminID int64
	err     error
}

func (m *mockAuthChecker) CheckAdminToken(ctx context.Context, token string) (int64, error) {
	return m.adminID, m.err
}

var _ AuthChecker = (*mockAuthChecker)(nil)

// mockAuditLoggerHandle 审计日志接口mock。
type mockAuditLoggerHandle struct {
	logs []auditLogEntry
}

type auditLogEntry struct {
	adminID  int64
	opType   int
	targetID int64
	content  string
	result   bool
}

func (m *mockAuditLoggerHandle) LogAudit(ctx context.Context, adminID int64, opType int, targetID int64, content string, result bool) {
	m.logs = append(m.logs, auditLogEntry{adminID, opType, targetID, content, result})
}

var _ AuditLogger = (*mockAuditLoggerHandle)(nil)

// mockPlayerAdmin 玩家管理接口mock。
type mockPlayerAdmin struct {
	banErr            error
	unbanErr          error
	banCalled         bool
	unbanCalled       bool
	lastBanPlayerID   int64
	lastBanDuration   int64
	lastBanReason     string
	lastUnbanPlayerID int64
}

func (m *mockPlayerAdmin) BanPlayer(ctx context.Context, playerID int64, durationMs int64, reason string) error {
	m.banCalled = true
	m.lastBanPlayerID = playerID
	m.lastBanDuration = durationMs
	m.lastBanReason = reason
	return m.banErr
}

func (m *mockPlayerAdmin) UnbanPlayer(ctx context.Context, playerID int64) error {
	m.unbanCalled = true
	m.lastUnbanPlayerID = playerID
	return m.unbanErr
}

var _ PlayerAdmin = (*mockPlayerAdmin)(nil)

// newTestAdminHandler 构造测试用AdminHandler实例。
func newTestAdminHandler(
	authChecker *mockAuthChecker,
	auditLogger *mockAuditLoggerHandle,
	playerAdmin *mockPlayerAdmin,
) *AdminHandler {
	logger := zap.NewNop()
	return NewAdminHandler(authChecker, auditLogger, playerAdmin, logger)
}

// TestAdminHandler_BanPlayerSuccess 测试封禁玩家成功。
func TestAdminHandler_BanPlayerSuccess(t *testing.T) {
	authChecker := &mockAuthChecker{adminID: 5001}
	auditLogger := &mockAuditLoggerHandle{}
	playerAdmin := &mockPlayerAdmin{}

	handler := newTestAdminHandler(authChecker, auditLogger, playerAdmin)
	resp, err := handler.BanPlayer(context.Background(), &adminpb.BanPlayerRequest{
		AdminToken: "admin-token",
		PlayerId:   1001,
		DurationMs: 3600000,
		Reason:     "违规",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Banned)
	assert.Equal(t, int32(0), resp.Header.ErrorCode)
	assert.Equal(t, "成功", resp.Header.ErrorMsg)

	// 应调用playerAdmin.BanPlayer
	assert.True(t, playerAdmin.banCalled)
	assert.Equal(t, int64(1001), playerAdmin.lastBanPlayerID)
	assert.Equal(t, int64(3600000), playerAdmin.lastBanDuration)
	assert.Equal(t, "违规", playerAdmin.lastBanReason)

	// 应记录审计日志（成功）
	require.Len(t, auditLogger.logs, 1)
	assert.Equal(t, int64(5001), auditLogger.logs[0].adminID)
	assert.Equal(t, AdminOpBanPlayer, auditLogger.logs[0].opType)
	assert.True(t, auditLogger.logs[0].result)
}

// TestAdminHandler_BanPlayerAuthFail 测试封禁鉴权失败。
func TestAdminHandler_BanPlayerAuthFail(t *testing.T) {
	authChecker := &mockAuthChecker{err: errors.New("token invalid")}
	auditLogger := &mockAuditLoggerHandle{}
	playerAdmin := &mockPlayerAdmin{}

	handler := newTestAdminHandler(authChecker, auditLogger, playerAdmin)
	_, err := handler.BanPlayer(context.Background(), &adminpb.BanPlayerRequest{
		AdminToken: "bad-token",
		PlayerId:   1001,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenInvalid))
	// 鉴权失败不应调用playerAdmin
	assert.False(t, playerAdmin.banCalled)
}

// TestAdminHandler_BanPlayerFail 测试封禁操作失败记录审计日志。
func TestAdminHandler_BanPlayerFail(t *testing.T) {
	authChecker := &mockAuthChecker{adminID: 5001}
	auditLogger := &mockAuditLoggerHandle{}
	playerAdmin := &mockPlayerAdmin{banErr: errors.New("account not found")}

	handler := newTestAdminHandler(authChecker, auditLogger, playerAdmin)
	_, err := handler.BanPlayer(context.Background(), &adminpb.BanPlayerRequest{
		AdminToken: "admin-token",
		PlayerId:   1001,
		DurationMs: 3600000,
		Reason:     "违规",
	})

	require.Error(t, err)
	// 失败也应记录审计日志（result=false）
	require.Len(t, auditLogger.logs, 1)
	assert.False(t, auditLogger.logs[0].result)
}

// TestAdminHandler_UnbanPlayerSuccess 测试解封玩家成功。
func TestAdminHandler_UnbanPlayerSuccess(t *testing.T) {
	authChecker := &mockAuthChecker{adminID: 5001}
	auditLogger := &mockAuditLoggerHandle{}
	playerAdmin := &mockPlayerAdmin{}

	handler := newTestAdminHandler(authChecker, auditLogger, playerAdmin)
	resp, err := handler.UnbanPlayer(context.Background(), &adminpb.UnbanPlayerRequest{
		AdminToken: "admin-token",
		PlayerId:   1001,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Unbanned)
	assert.True(t, playerAdmin.unbanCalled)
	assert.Equal(t, int64(1001), playerAdmin.lastUnbanPlayerID)
	require.Len(t, auditLogger.logs, 1)
	assert.Equal(t, AdminOpUnbanPlayer, auditLogger.logs[0].opType)
	assert.True(t, auditLogger.logs[0].result)
}

// TestAdminHandler_UnbanPlayerAuthFail 测试解封鉴权失败。
func TestAdminHandler_UnbanPlayerAuthFail(t *testing.T) {
	authChecker := &mockAuthChecker{err: errors.New("token invalid")}
	auditLogger := &mockAuditLoggerHandle{}
	playerAdmin := &mockPlayerAdmin{}

	handler := newTestAdminHandler(authChecker, auditLogger, playerAdmin)
	_, err := handler.UnbanPlayer(context.Background(), &adminpb.UnbanPlayerRequest{
		AdminToken: "bad-token",
		PlayerId:   1001,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenInvalid))
	assert.False(t, playerAdmin.unbanCalled)
}

// TestAdminHandler_UnbanPlayerFail 测试解封操作失败。
func TestAdminHandler_UnbanPlayerFail(t *testing.T) {
	authChecker := &mockAuthChecker{adminID: 5001}
	auditLogger := &mockAuditLoggerHandle{}
	playerAdmin := &mockPlayerAdmin{unbanErr: errors.New("account not found")}

	handler := newTestAdminHandler(authChecker, auditLogger, playerAdmin)
	_, err := handler.UnbanPlayer(context.Background(), &adminpb.UnbanPlayerRequest{
		AdminToken: "admin-token",
		PlayerId:   1001,
	})

	require.Error(t, err)
	require.Len(t, auditLogger.logs, 1)
	assert.False(t, auditLogger.logs[0].result)
}

// TestNewAdminHandler 测试创建AdminHandler实例。
func TestNewAdminHandler(t *testing.T) {
	handler := newTestAdminHandler(&mockAuthChecker{}, &mockAuditLoggerHandle{}, &mockPlayerAdmin{})
	require.NotNil(t, handler)
}

// TestAdminOpConstants 测试管理操作类型常量。
func TestAdminOpConstants(t *testing.T) {
	assert.Equal(t, 6, AdminOpBanPlayer)
	assert.Equal(t, 7, AdminOpUnbanPlayer)
}
