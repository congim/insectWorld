package command

import (
	"context"

	domainaccount "insectworld/server/gateway/domain/account"
	domainaudit "insectworld/server/gateway/domain/audit"
	domainidgen "insectworld/server/gateway/domain/idgen"
	domainratelimit "insectworld/server/gateway/domain/ratelimit"
	domainsecurity "insectworld/server/gateway/domain/security"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"
	domainwebsocket "insectworld/server/gateway/domain/websocket"

	"insectworld/server/shared/pkg/eventbus"
)

// mockLoginFailureTracker 登录失败计数器mock，供BruteForceProtector构造使用。
type mockLoginFailureTracker struct {
	recordFailCount int
	recordFailErr   error
	isLockedResult  bool
	isLockedErr     error
	remainingResult int64
	remainingErr    error
	resetErr        error

	recordFailCalled bool
	isLockedCalled   bool
	resetCalled      bool
}

func (m *mockLoginFailureTracker) RecordFailure(ctx context.Context, username string) (int, error) {
	m.recordFailCalled = true
	return m.recordFailCount, m.recordFailErr
}

func (m *mockLoginFailureTracker) IsLocked(ctx context.Context, username string) (bool, error) {
	m.isLockedCalled = true
	return m.isLockedResult, m.isLockedErr
}

func (m *mockLoginFailureTracker) RemainingLockSeconds(ctx context.Context, username string) (int64, error) {
	return m.remainingResult, m.remainingErr
}

func (m *mockLoginFailureTracker) ResetClear(ctx context.Context, username string) error {
	m.resetCalled = true
	return m.resetErr
}

var _ domainsecurity.LoginFailureTracker = (*mockLoginFailureTracker)(nil)

// mockAccountRepo 账号仓储mock，可配置各方法返回值与调用计数。
type mockAccountRepo struct {
	saveErr           error
	findByIDAccount   *domainaccount.PlayerAccount
	findByIDErr       error
	findByUsernameAcc *domainaccount.PlayerAccount
	findByUsernameErr error
	existsResult      bool
	existsErr         error

	saveCallCount           int
	findByIDCallCount       int
	findByUsernameCallCount int
	existsCallCount         int
	lastSavedAccount        *domainaccount.PlayerAccount
}

func (m *mockAccountRepo) Save(ctx context.Context, account *domainaccount.PlayerAccount) error {
	m.saveCallCount++
	m.lastSavedAccount = account
	return m.saveErr
}

func (m *mockAccountRepo) FindByID(ctx context.Context, playerID int64) (*domainaccount.PlayerAccount, error) {
	m.findByIDCallCount++
	return m.findByIDAccount, m.findByIDErr
}

func (m *mockAccountRepo) FindByUsername(ctx context.Context, username string) (*domainaccount.PlayerAccount, error) {
	m.findByUsernameCallCount++
	return m.findByUsernameAcc, m.findByUsernameErr
}

func (m *mockAccountRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	m.existsCallCount++
	return m.existsResult, m.existsErr
}

var _ domainaccount.AccountRepository = (*mockAccountRepo)(nil)

// mockSessionRepo 会话仓储mock。
type mockSessionRepo struct {
	saveErr           error
	findByPlayerSess  *domainsession.OnlineSession
	findByPlayerErr   error
	deleteErr         error
	findExpiredResult []*domainsession.OnlineSession
	findExpiredErr    error

	saveCallCount         int
	findByPlayerCallCount int
	deleteCallCount       int
	findExpiredCallCount  int
	lastSavedSession      *domainsession.OnlineSession
	deletedPlayerIDs      []int64
}

func (m *mockSessionRepo) Save(ctx context.Context, sess *domainsession.OnlineSession) error {
	m.saveCallCount++
	m.lastSavedSession = sess
	return m.saveErr
}

func (m *mockSessionRepo) FindByPlayerID(ctx context.Context, playerID int64) (*domainsession.OnlineSession, error) {
	m.findByPlayerCallCount++
	return m.findByPlayerSess, m.findByPlayerErr
}

func (m *mockSessionRepo) Delete(ctx context.Context, playerID int64) error {
	m.deleteCallCount++
	m.deletedPlayerIDs = append(m.deletedPlayerIDs, playerID)
	return m.deleteErr
}

func (m *mockSessionRepo) FindExpired(ctx context.Context, thresholdTime int64, limit int) ([]*domainsession.OnlineSession, error) {
	m.findExpiredCallCount++
	return m.findExpiredResult, m.findExpiredErr
}

var _ domainsession.SessionRepository = (*mockSessionRepo)(nil)

// mockRateLimiter 限流器mock，可配置allow返回值序列。
type mockRateLimiter struct {
	allowResults []bool
	callCount    int
}

func (m *mockRateLimiter) Allow(ctx context.Context, dimension string, key string) bool {
	if m.callCount < len(m.allowResults) {
		r := m.allowResults[m.callCount]
		m.callCount++
		return r
	}
	m.callCount++
	return true
}

var _ domainratelimit.RateLimiter = (*mockRateLimiter)(nil)

// mockIDGenerator ID生成器mock。
type mockIDGenerator struct {
	nextID    int64
	nextIDErr error
}

func (m *mockIDGenerator) NextID(ctx context.Context) (int64, error) {
	return m.nextID, m.nextIDErr
}

var _ domainidgen.IDGenerator = (*mockIDGenerator)(nil)

// mockHasher 密码哈希器mock。
type mockHasher struct {
	hashResult   string
	saltResult   string
	hashErr      error
	verifyResult bool
	verifyErr    error
}

func (m *mockHasher) Hash(ctx context.Context, password string) (string, string, error) {
	return m.hashResult, m.saltResult, m.hashErr
}

func (m *mockHasher) Verify(ctx context.Context, password string, hash string, salt string) (bool, error) {
	return m.verifyResult, m.verifyErr
}

var _ domainaccount.PasswordHasher = (*mockHasher)(nil)

// mockTokenSigner 令牌签发器mock。
type mockTokenSigner struct {
	signResult   string
	signErr      error
	verifyResult domaintoken.TokenPayload
	verifyErr    error
}

func (m *mockTokenSigner) Sign(ctx context.Context, payload domaintoken.TokenPayload) (string, error) {
	return m.signResult, m.signErr
}

func (m *mockTokenSigner) Verify(ctx context.Context, tokenStr string) (domaintoken.TokenPayload, error) {
	return m.verifyResult, m.verifyErr
}

var _ domaintoken.TokenSigner = (*mockTokenSigner)(nil)

// mockTokenBlacklist 令牌黑名单mock。
type mockTokenBlacklist struct {
	invalidateErr   error
	isInvalidResult bool
	isInvalidErr    error

	invalidateCallCount int
	isInvalidCallCount  int
}

func (m *mockTokenBlacklist) Invalidate(ctx context.Context, playerID int64, tokenVersion int, remainingTTLSeconds int64) error {
	m.invalidateCallCount++
	return m.invalidateErr
}

func (m *mockTokenBlacklist) IsInvalid(ctx context.Context, playerID int64, tokenVersion int) (bool, error) {
	m.isInvalidCallCount++
	return m.isInvalidResult, m.isInvalidErr
}

var _ domaintoken.TokenBlacklist = (*mockTokenBlacklist)(nil)

// mockAuditLogger 审计日志mock。
type mockAuditLogger struct {
	logErr       error
	logCallCount int
	lastRecord   *domainaudit.AuditRecord
}

func (m *mockAuditLogger) LogRecord(ctx context.Context, record *domainaudit.AuditRecord) error {
	m.logCallCount++
	m.lastRecord = record
	return m.logErr
}

var _ domainaudit.AuditLogger = (*mockAuditLogger)(nil)

// mockConnManager 连接管理器mock。
type mockConnManager struct {
	sendErr    error
	sendCount  int
	lastPlayer int64
	lastMsg    []byte
}

func (m *mockConnManager) Send(ctx context.Context, playerID int64, msg []byte) error {
	m.sendCount++
	m.lastPlayer = playerID
	m.lastMsg = msg
	return m.sendErr
}

var _ domainwebsocket.ConnectionManager = (*mockConnManager)(nil)

// mockEventBus 事件总线mock。
type mockEventBus struct {
	publishErr      error
	publishCount    int
	publishedEvents []eventbus.DomainEvent
	subscribeErr    error
}

func (m *mockEventBus) Publish(ctx context.Context, event eventbus.DomainEvent) error {
	m.publishCount++
	m.publishedEvents = append(m.publishedEvents, event)
	return m.publishErr
}

func (m *mockEventBus) Subscribe(ctx context.Context, eventType string, handler eventbus.EventHandler) error {
	return m.subscribeErr
}

var _ eventbus.EventBus = (*mockEventBus)(nil)
