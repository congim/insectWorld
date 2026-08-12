package query

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"

	"go.uber.org/zap"
)

// futureExpireTime 返回未来过期时间戳，供鉴权测试使用。
func futureExpireTime() int64 {
	return time.Now().UnixMilli() + 3600000
}

// queryTokenSigner 令牌签发器mock。
type queryTokenSigner struct {
	verifyResult domaintoken.TokenPayload
	verifyErr    error
}

func (m *queryTokenSigner) Sign(ctx context.Context, payload domaintoken.TokenPayload) (string, error) {
	return "", nil
}

func (m *queryTokenSigner) Verify(ctx context.Context, tokenStr string) (domaintoken.TokenPayload, error) {
	return m.verifyResult, m.verifyErr
}

var _ domaintoken.TokenSigner = (*queryTokenSigner)(nil)

// queryTokenBlacklist 令牌黑名单mock。
type queryTokenBlacklist struct {
	isInvalidResult bool
	isInvalidErr    error
}

func (m *queryTokenBlacklist) Invalidate(ctx context.Context, playerID int64, tokenVersion int, remainingTTLSeconds int64) error {
	return nil
}

func (m *queryTokenBlacklist) IsInvalid(ctx context.Context, playerID int64, tokenVersion int) (bool, error) {
	return m.isInvalidResult, m.isInvalidErr
}

var _ domaintoken.TokenBlacklist = (*queryTokenBlacklist)(nil)

// querySessionRepo 会话仓储mock。
type querySessionRepo struct {
	findByPlayerSess *domainsession.OnlineSession
	findByPlayerErr  error
}

func (m *querySessionRepo) Save(ctx context.Context, sess *domainsession.OnlineSession) error {
	return nil
}
func (m *querySessionRepo) FindByPlayerID(ctx context.Context, playerID int64) (*domainsession.OnlineSession, error) {
	return m.findByPlayerSess, m.findByPlayerErr
}
func (m *querySessionRepo) Delete(ctx context.Context, playerID int64) error { return nil }
func (m *querySessionRepo) FindExpired(ctx context.Context, thresholdTime int64, limit int) ([]*domainsession.OnlineSession, error) {
	return nil, nil
}

var _ domainsession.SessionRepository = (*querySessionRepo)(nil)

// newTestAuthenticateQuery 构造测试用AuthenticateQuery实例。
func newTestAuthenticateQuery(
	tokenSigner *queryTokenSigner,
	tokenBlacklist *queryTokenBlacklist,
	sessionRepo *querySessionRepo,
) *AuthenticateQuery {
	logger := zap.NewNop()
	return NewAuthenticateQuery(tokenSigner, tokenBlacklist, sessionRepo, logger)
}

// TestAuthenticateQuery_Success 测试鉴权成功返回playerID。
func TestAuthenticateQuery_Success(t *testing.T) {
	tokenSigner := &queryTokenSigner{verifyResult: domaintoken.TokenPayload{
		PlayerID: 1001, IssueTime: 1700000000000, ExpireTime: futureExpireTime(), Version: 1,
	}}
	tokenBlacklist := &queryTokenBlacklist{isInvalidResult: false}
	sessionRepo := &querySessionRepo{
		findByPlayerSess: domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device"),
	}

	cmd := newTestAuthenticateQuery(tokenSigner, tokenBlacklist, sessionRepo)
	playerID, err := cmd.Handle(context.Background(), "valid-token")

	require.NoError(t, err)
	assert.Equal(t, int64(1001), playerID)
}

// TestAuthenticateQuery_TokenMissing 测试令牌缺失返回ErrTokenMissing。
func TestAuthenticateQuery_TokenMissing(t *testing.T) {
	cmd := newTestAuthenticateQuery(&queryTokenSigner{}, &queryTokenBlacklist{}, &querySessionRepo{})
	_, err := cmd.Handle(context.Background(), "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenMissing))
}

// TestAuthenticateQuery_TokenInvalid 测试令牌无效返回ErrTokenInvalid。
func TestAuthenticateQuery_TokenInvalid(t *testing.T) {
	tokenSigner := &queryTokenSigner{verifyErr: gatewayerr.ErrTokenInvalid}
	cmd := newTestAuthenticateQuery(tokenSigner, &queryTokenBlacklist{}, &querySessionRepo{})
	_, err := cmd.Handle(context.Background(), "invalid-token")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenInvalid))
}

// TestAuthenticateQuery_TokenExpired 测试令牌过期返回ErrTokenExpired。
func TestAuthenticateQuery_TokenExpired(t *testing.T) {
	// ExpireTime设为过去时间
	tokenSigner := &queryTokenSigner{verifyResult: domaintoken.TokenPayload{
		PlayerID: 1001, IssueTime: 1700000000000, ExpireTime: 1, Version: 1,
	}}
	cmd := newTestAuthenticateQuery(tokenSigner, &queryTokenBlacklist{}, &querySessionRepo{})
	_, err := cmd.Handle(context.Background(), "expired-token")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenExpired))
}

// TestAuthenticateQuery_TokenBlacklisted 测试令牌在黑名单返回ErrTokenInvalid。
func TestAuthenticateQuery_TokenBlacklisted(t *testing.T) {
	tokenSigner := &queryTokenSigner{verifyResult: domaintoken.TokenPayload{
		PlayerID: 1001, IssueTime: 1700000000000, ExpireTime: futureExpireTime(), Version: 1,
	}}
	tokenBlacklist := &queryTokenBlacklist{isInvalidResult: true}
	cmd := newTestAuthenticateQuery(tokenSigner, tokenBlacklist, &querySessionRepo{})
	_, err := cmd.Handle(context.Background(), "blacklisted-token")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenInvalid))
}

// TestAuthenticateQuery_BlacklistDegrade 测试黑名单故障降级为依赖会话存在性兜底。
func TestAuthenticateQuery_BlacklistDegrade(t *testing.T) {
	tokenSigner := &queryTokenSigner{verifyResult: domaintoken.TokenPayload{
		PlayerID: 1001, IssueTime: 1700000000000, ExpireTime: futureExpireTime(), Version: 1,
	}}
	tokenBlacklist := &queryTokenBlacklist{isInvalidErr: gatewayerr.ErrTokenBlacklistUnavailable}
	sessionRepo := &querySessionRepo{
		findByPlayerSess: domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device"),
	}
	cmd := newTestAuthenticateQuery(tokenSigner, tokenBlacklist, sessionRepo)
	// 黑名单故障降级，会话存在则鉴权成功
	playerID, err := cmd.Handle(context.Background(), "valid-token")
	require.NoError(t, err)
	assert.Equal(t, int64(1001), playerID)
}

// TestAuthenticateQuery_SessionNotFound 测试会话不存在返回ErrTokenInvalid。
func TestAuthenticateQuery_SessionNotFound(t *testing.T) {
	tokenSigner := &queryTokenSigner{verifyResult: domaintoken.TokenPayload{
		PlayerID: 1001, IssueTime: 1700000000000, ExpireTime: futureExpireTime(), Version: 1,
	}}
	tokenBlacklist := &queryTokenBlacklist{isInvalidResult: false}
	sessionRepo := &querySessionRepo{findByPlayerErr: gatewayerr.ErrSessionNotFound}
	cmd := newTestAuthenticateQuery(tokenSigner, tokenBlacklist, sessionRepo)
	_, err := cmd.Handle(context.Background(), "valid-token")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenInvalid))
}

// TestAuthenticateQuery_SessionRepoError 测试会话仓储故障返回ErrTokenInvalid。
func TestAuthenticateQuery_SessionRepoError(t *testing.T) {
	tokenSigner := &queryTokenSigner{verifyResult: domaintoken.TokenPayload{
		PlayerID: 1001, IssueTime: 1700000000000, ExpireTime: futureExpireTime(), Version: 1,
	}}
	tokenBlacklist := &queryTokenBlacklist{isInvalidResult: false}
	sessionRepo := &querySessionRepo{findByPlayerErr: gatewayerr.ErrSessionRepoUnavailable}
	cmd := newTestAuthenticateQuery(tokenSigner, tokenBlacklist, sessionRepo)
	_, err := cmd.Handle(context.Background(), "valid-token")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrTokenInvalid))
}
