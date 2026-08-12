package websocket

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainaccount "insectworld/server/gateway/domain/account"
	domainaudit "insectworld/server/gateway/domain/audit"
	domainconfig "insectworld/server/gateway/domain/config"
	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsecurity "insectworld/server/gateway/domain/security"
	domaintoken "insectworld/server/gateway/domain/token"

	"go.uber.org/zap"

	"insectworld/server/gateway/application/command"
	"insectworld/server/gateway/application/query"
	infraacc "insectworld/server/gateway/infrastructure/auth/account"
	infratoken "insectworld/server/gateway/infrastructure/auth/token"
	infraeventbus "insectworld/server/gateway/infrastructure/eventbus"
	infraidgen "insectworld/server/gateway/infrastructure/idgen"
	infrasession "insectworld/server/gateway/infrastructure/persistence/session"
	infraratelimit "insectworld/server/gateway/infrastructure/ratelimit"
	infrawebsocket "insectworld/server/gateway/infrastructure/websocket"
)

// memAccountRepo 内存账号仓储，供WSAuthHandler集成测试使用。
type memAccountRepo struct {
	mu       sync.RWMutex
	accounts map[int64]*domainaccount.PlayerAccount
}

func newMemAccountRepo() *memAccountRepo {
	return &memAccountRepo{accounts: make(map[int64]*domainaccount.PlayerAccount)}
}

func (r *memAccountRepo) Save(ctx context.Context, account *domainaccount.PlayerAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts[account.PlayerID()] = account
	return nil
}

func (r *memAccountRepo) FindByID(ctx context.Context, playerID int64) (*domainaccount.PlayerAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.accounts[playerID]
	if !ok {
		return nil, gatewayerr.ErrAccountNotFoundSentinel
	}
	return a, nil
}

func (r *memAccountRepo) FindByUsername(ctx context.Context, username string) (*domainaccount.PlayerAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.accounts {
		if a.Username() == username {
			return a, nil
		}
	}
	return nil, gatewayerr.ErrAccountNotFoundSentinel
}

func (r *memAccountRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.accounts {
		if a.Username() == username {
			return true, nil
		}
	}
	return false, nil
}

// memTokenBlacklist 内存令牌黑名单。
type memTokenBlacklist struct {
	mu   sync.RWMutex
	data map[string]bool
}

func newMemTokenBlacklist() *memTokenBlacklist {
	return &memTokenBlacklist{data: make(map[string]bool)}
}

func (b *memTokenBlacklist) Invalidate(ctx context.Context, playerID int64, tokenVersion int, remainingTTLSeconds int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[blacklistKey(playerID, tokenVersion)] = true
	return nil
}

func (b *memTokenBlacklist) IsInvalid(ctx context.Context, playerID int64, tokenVersion int) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.data[blacklistKey(playerID, tokenVersion)], nil
}

func blacklistKey(playerID int64, v int) string {
	return string(rune(playerID)) + ":" + string(rune(v))
}

// memFailureTracker 内存登录失败计数器。
type memFailureTracker struct {
	mu     sync.RWMutex
	counts map[string]int
}

func newMemFailureTracker() *memFailureTracker {
	return &memFailureTracker{counts: make(map[string]int)}
}

func (t *memFailureTracker) RecordFailure(ctx context.Context, username string) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts[username]++
	return t.counts[username], nil
}

func (t *memFailureTracker) IsLocked(ctx context.Context, username string) (bool, error) {
	return false, nil
}

func (t *memFailureTracker) RemainingLockSeconds(ctx context.Context, username string) (int64, error) {
	return 0, nil
}

func (t *memFailureTracker) ResetClear(ctx context.Context, username string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts[username] = 0
	return nil
}

// memAuditLogger 内存审计日志。
type memAuditLogger struct {
	mu   sync.Mutex
	logs []*domainaudit.AuditRecord
}

func (l *memAuditLogger) LogRecord(ctx context.Context, record *domainaudit.AuditRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, record)
	return nil
}

// 确保内存mock实现对应接口。
var _ domainaccount.AccountRepository = (*memAccountRepo)(nil)
var _ domaintoken.TokenBlacklist = (*memTokenBlacklist)(nil)
var _ domainsecurity.LoginFailureTracker = (*memFailureTracker)(nil)
var _ domainaudit.AuditLogger = (*memAuditLogger)(nil)

// buildFullWSAuthHandler 构造完整WSAuthHandler实例，注入真实/内存依赖。
func buildFullWSAuthHandler(t *testing.T) (*WSAuthHandler, *memAccountRepo, *infratoken.TokenSignerImpl) {
	logger := zap.NewNop()

	signer, err := infratoken.NewTokenSignerImpl([]byte("ws-test-key"), logger)
	require.NoError(t, err)

	accountRepo := newMemAccountRepo()
	sessionRepo := infrasession.NewSessionRepoMemory(5*time.Minute, logger)
	rateLimiter := infraratelimit.NewRateLimiterImpl(map[string]infraratelimit.RateConfig{
		"register:ip":   {Rate: 100, Burst: 100},
		"login:ip":      {Rate: 100, Burst: 100},
		"login:account": {Rate: 100, Burst: 100},
	}, logger)
	idGen, err := infraidgen.NewSnowflakeIDGen(1)
	require.NoError(t, err)
	hasher := infraacc.NewBcryptHasher(4, logger) // 低cost加速测试
	blacklist := newMemTokenBlacklist()
	tracker := newMemFailureTracker()
	bruteForce := domainsecurity.NewBruteForceProtector(tracker, 5, 900000)
	eventBus := infraeventbus.NewInMemoryEventBus(logger)
	auditLogger := &memAuditLogger{}
	connManager := infrawebsocket.NewConnectionManager(logger)
	cfg := domainconfig.DefaultAuthConfig()
	cfg.TokenSigningKey = "ws-test-key"

	registerCmd := command.NewRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, cfg, logger)
	loginCmd := command.NewLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher,
		signer, eventBus, auditLogger, connManager, cfg, logger)
	logoutCmd := command.NewLogoutCommand(signer, blacklist, sessionRepo, eventBus, auditLogger, logger)
	heartbeatCmd := command.NewHeartbeatCommand(signer, sessionRepo, logger)
	authQuery := query.NewAuthenticateQuery(signer, blacklist, sessionRepo, logger)

	handler := NewWSAuthHandler(registerCmd, loginCmd, logoutCmd, heartbeatCmd, authQuery, logger)
	return handler, accountRepo, signer
}

// decodeResp 解析WSAuthHandler响应。
func decodeResp(t *testing.T, data []byte) AuthResponse {
	var resp AuthResponse
	require.NoError(t, json.Unmarshal(data, &resp))
	return resp
}

// TestWSAuthHandler_RegisterSuccess 测试注册消息处理成功。
func TestWSAuthHandler_RegisterSuccess(t *testing.T) {
	handler, accountRepo, _ := buildFullWSAuthHandler(t)

	msg, _ := json.Marshal(AuthMessage{Type: MsgTypeRegister, Username: "newuser", Password: "pass1234"})
	resp, err := handler.HandleMessage(context.Background(), msg, "127.0.0.1", "conn-001")
	require.NoError(t, err)

	authResp := decodeResp(t, resp)
	assert.True(t, authResp.Success)
	assert.Equal(t, MsgTypeRegister, authResp.Type)
	assert.Greater(t, authResp.PlayerID, int64(0))

	accountRepo.mu.RLock()
	defer accountRepo.mu.RUnlock()
	assert.Len(t, accountRepo.accounts, 1, "应保存账号")
}

// TestWSAuthHandler_RegisterFail 测试注册消息处理失败（用户名非法）。
func TestWSAuthHandler_RegisterFail(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)

	msg, _ := json.Marshal(AuthMessage{Type: MsgTypeRegister, Username: "123", Password: "pass1234"})
	resp, err := handler.HandleMessage(context.Background(), msg, "127.0.0.1", "conn-001")
	require.NoError(t, err)

	authResp := decodeResp(t, resp)
	assert.False(t, authResp.Success)
	assert.Equal(t, MsgTypeRegister, authResp.Type)
	assert.Equal(t, 17010, authResp.ErrorCode, "应返回用户名格式错误码")
}

// TestWSAuthHandler_LoginSuccess 测试登录消息处理成功。
func TestWSAuthHandler_LoginSuccess(t *testing.T) {
	handler, accountRepo, _ := buildFullWSAuthHandler(t)

	// 先注册
	regMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeRegister, Username: "loginuser", Password: "pass1234"})
	_, _ = handler.HandleMessage(context.Background(), regMsg, "127.0.0.1", "conn-001")

	// 登录
	loginMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeLogin, Username: "loginuser", Password: "pass1234", DeviceID: "device-001"})
	resp, err := handler.HandleMessage(context.Background(), loginMsg, "127.0.0.1", "conn-002")
	require.NoError(t, err)

	authResp := decodeResp(t, resp)
	assert.True(t, authResp.Success)
	assert.Equal(t, MsgTypeLogin, authResp.Type)
	assert.NotEmpty(t, authResp.Token)
	assert.Greater(t, authResp.PlayerID, int64(0))
	assert.Greater(t, authResp.SessionTTLms, int64(0))

	accountRepo.mu.RLock()
	defer accountRepo.mu.RUnlock()
	assert.Len(t, accountRepo.accounts, 1)
}

// TestWSAuthHandler_LoginFail 测试登录消息处理失败（账号不存在）。
func TestWSAuthHandler_LoginFail(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)

	loginMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeLogin, Username: "nouser", Password: "pass1234"})
	resp, err := handler.HandleMessage(context.Background(), loginMsg, "127.0.0.1", "conn-001")
	require.NoError(t, err)

	authResp := decodeResp(t, resp)
	assert.False(t, authResp.Success)
	assert.Equal(t, MsgTypeLogin, authResp.Type)
	assert.Equal(t, 17020, authResp.ErrorCode, "应返回账号不存在错误码")
}

// TestWSAuthHandler_LogoutSuccess 测试登出消息处理。
func TestWSAuthHandler_LogoutSuccess(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)

	// 登出无效令牌应幂等返回成功
	logoutMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeLogout, Token: "invalid-token", PlayerID: 1001})
	resp, err := handler.HandleMessage(context.Background(), logoutMsg, "127.0.0.1", "conn-001")
	require.NoError(t, err)

	authResp := decodeResp(t, resp)
	assert.True(t, authResp.Success, "登出应幂等返回成功")
	assert.Equal(t, MsgTypeLogout, authResp.Type)
}

// TestWSAuthHandler_HeartbeatFail 测试心跳消息处理失败（令牌无效）。
func TestWSAuthHandler_HeartbeatFail(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)

	hbMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeHeartbeat, Token: "invalid-token", PlayerID: 1001})
	resp, err := handler.HandleMessage(context.Background(), hbMsg, "127.0.0.1", "conn-001")
	require.NoError(t, err)

	authResp := decodeResp(t, resp)
	assert.False(t, authResp.Success)
	assert.Equal(t, MsgTypeHeartbeat, authResp.Type)
	assert.Equal(t, 17001, authResp.ErrorCode, "应返回令牌无效错误码")
}

// TestWSAuthHandler_AuthenticateFail 测试鉴权消息处理失败。
func TestWSAuthHandler_AuthenticateFail(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)

	authMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeAuthenticate, Token: "invalid-token"})
	resp, err := handler.HandleMessage(context.Background(), authMsg, "127.0.0.1", "conn-001")
	require.NoError(t, err)

	authResp := decodeResp(t, resp)
	assert.False(t, authResp.Success)
	assert.Equal(t, MsgTypeAuthenticate, authResp.Type)
	assert.Equal(t, 17001, authResp.ErrorCode)
}

// TestWSAuthHandler_AuthenticateSuccess 测试鉴权消息处理成功。
func TestWSAuthHandler_AuthenticateSuccess(t *testing.T) {
	handler, _, signer := buildFullWSAuthHandler(t)

	// 先注册并登录获取有效令牌
	regMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeRegister, Username: "authuser", Password: "pass1234"})
	_, _ = handler.HandleMessage(context.Background(), regMsg, "127.0.0.1", "conn-001")
	loginMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeLogin, Username: "authuser", Password: "pass1234", DeviceID: "device-001"})
	loginResp, _ := handler.HandleMessage(context.Background(), loginMsg, "127.0.0.1", "conn-002")
	loginAuthResp := decodeResp(t, loginResp)
	require.True(t, loginAuthResp.Success)

	// 用有效令牌鉴权
	authMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeAuthenticate, Token: loginAuthResp.Token})
	resp, err := handler.HandleMessage(context.Background(), authMsg, "127.0.0.1", "conn-003")
	require.NoError(t, err)

	authResp := decodeResp(t, resp)
	assert.True(t, authResp.Success)
	assert.Equal(t, MsgTypeAuthenticate, authResp.Type)
	assert.Equal(t, loginAuthResp.PlayerID, authResp.PlayerID)

	_ = signer
}

// TestWSAuthHandler_HeartbeatSuccess 测试心跳消息处理成功（有效令牌+已登录会话）。
func TestWSAuthHandler_HeartbeatSuccess(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)

	// 注册并登录获取有效令牌
	regMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeRegister, Username: "hbuser", Password: "pass1234"})
	_, _ = handler.HandleMessage(context.Background(), regMsg, "127.0.0.1", "conn-001")
	loginMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeLogin, Username: "hbuser", Password: "pass1234", DeviceID: "device-001"})
	loginResp, _ := handler.HandleMessage(context.Background(), loginMsg, "127.0.0.1", "conn-002")
	loginAuthResp := decodeResp(t, loginResp)
	require.True(t, loginAuthResp.Success, "登录应成功")

	// 发送心跳
	hbMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeHeartbeat, Token: loginAuthResp.Token, PlayerID: loginAuthResp.PlayerID})
	resp, err := handler.HandleMessage(context.Background(), hbMsg, "127.0.0.1", "conn-002")
	require.NoError(t, err)

	hbResp := decodeResp(t, resp)
	assert.True(t, hbResp.Success, "心跳应成功")
	assert.Equal(t, MsgTypeHeartbeat, hbResp.Type)
}

// TestWSAuthHandler_LogoutWithValidToken 测试登出成功路径（有效令牌+已登录会话）。
func TestWSAuthHandler_LogoutWithValidToken(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)

	// 注册并登录
	regMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeRegister, Username: "logoutuser", Password: "pass1234"})
	_, _ = handler.HandleMessage(context.Background(), regMsg, "127.0.0.1", "conn-001")
	loginMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeLogin, Username: "logoutuser", Password: "pass1234", DeviceID: "device-001"})
	loginResp, _ := handler.HandleMessage(context.Background(), loginMsg, "127.0.0.1", "conn-002")
	loginAuthResp := decodeResp(t, loginResp)
	require.True(t, loginAuthResp.Success, "登录应成功")

	// 登出
	logoutMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeLogout, Token: loginAuthResp.Token, PlayerID: loginAuthResp.PlayerID})
	resp, err := handler.HandleMessage(context.Background(), logoutMsg, "127.0.0.1", "conn-002")
	require.NoError(t, err)

	logoutResp := decodeResp(t, resp)
	assert.True(t, logoutResp.Success, "登出应成功")
	assert.Equal(t, MsgTypeLogout, logoutResp.Type)
}

// TestWSServer_HandleAuthViaHTTP 测试WSServer的handleAuth端点通过真实WebSocket连接。
func TestWSServer_HandleAuthViaHTTP(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)
	logger := zap.NewNop()

	srv := NewWSServer(handler, logger)

	// 使用httptest启动HTTP服务器，复用handleAuth处理器
	ts := httptest.NewServer(http.HandlerFunc(srv.handleAuth))
	defer ts.Close()

	// 将http://替换为ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// 建立WebSocket连接
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	if resp != nil {
		resp.Body.Close()
	}

	// 发送注册消息
	regMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeRegister, Username: "wsuser", Password: "pass1234"})
	err = conn.WriteMessage(websocket.TextMessage, regMsg)
	require.NoError(t, err)

	// 读取响应
	_, respBytes, err := conn.ReadMessage()
	require.NoError(t, err)
	regResp := decodeResp(t, respBytes)
	assert.True(t, regResp.Success, "通过WebSocket注册应成功")
	assert.Equal(t, MsgTypeRegister, regResp.Type)

	// 发送登录消息
	loginMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeLogin, Username: "wsuser", Password: "pass1234", DeviceID: "device-001"})
	err = conn.WriteMessage(websocket.TextMessage, loginMsg)
	require.NoError(t, err)

	_, respBytes, err = conn.ReadMessage()
	require.NoError(t, err)
	loginResp := decodeResp(t, respBytes)
	assert.True(t, loginResp.Success, "通过WebSocket登录应成功")
	assert.NotEmpty(t, loginResp.Token)
}

// TestWSServer_Start 测试WSServer.Start启动和优雅关闭。
func TestWSServer_Start(t *testing.T) {
	handler, _, _ := buildFullWSAuthHandler(t)
	logger := zap.NewNop()

	srv := NewWSServer(handler, logger)

	// 获取空闲端口
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动服务器
	err = srv.Start(ctx, addr)
	require.NoError(t, err)

	// 等待服务器就绪
	time.Sleep(100 * time.Millisecond)

	// 建立WebSocket连接并发送消息
	wsURL := "ws://" + addr + "/auth"
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	if resp != nil {
		resp.Body.Close()
	}

	// 发送注册消息
	regMsg, _ := json.Marshal(AuthMessage{Type: MsgTypeRegister, Username: "startuser", Password: "pass1234"})
	err = conn.WriteMessage(websocket.TextMessage, regMsg)
	require.NoError(t, err)

	_, respBytes, err := conn.ReadMessage()
	require.NoError(t, err)
	regResp := decodeResp(t, respBytes)
	assert.True(t, regResp.Success, "通过Start启动的服务器注册应成功")

	// 关闭服务器
	cancel()
	time.Sleep(100 * time.Millisecond)
}
