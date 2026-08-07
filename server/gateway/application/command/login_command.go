// Package command Gateway服务application层命令，编排用户认证操作。
package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/eventbus"

	domainaccount "insectworld/server/gateway/domain/account"
	domainaudit "insectworld/server/gateway/domain/audit"
	gatewayerr "insectworld/server/gateway/domain/errors"
	domainevent "insectworld/server/gateway/domain/event"
	domainratelimit "insectworld/server/gateway/domain/ratelimit"
	domainsecurity "insectworld/server/gateway/domain/security"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"
	"insectworld/server/gateway/infrastructure/config"
	"insectworld/server/gateway/infrastructure/websocket"
)

// 限流维度常量（规范1就近归属）。
const (
	dimensionLoginIP      = "login:ip"      // 登录限流维度：按IP
	dimensionLoginAccount = "login:account" // 登录限流维度：按账号名
)

// 踢下线通知消息内容。
const kickOutMessage = `{"type":"kick_out","reason":"single_login"}`

// LoginCommand 登录命令，编排用户登录流程。
//
// 编排顺序严格遵循design.md 2.1.3.4节：校验登录频率→查询账号档案→
// 校验账号状态→校验暴力破解锁定→校验密码→单点登录踢下线→创建新会话→
// 签发令牌→发布上线事件→审计日志→返回。
type LoginCommand struct {
	accountRepo domainaccount.AccountRepository     // 账号仓储
	sessionRepo domainsession.SessionRepository     // 会话仓储
	rateLimiter domainratelimit.RateLimiter         // 限流器
	bruteForce  *domainsecurity.BruteForceProtector // 暴力破解防护
	hasher      domainaccount.PasswordHasher        // 密码哈希器
	tokenSigner domaintoken.TokenSigner             // 令牌签发器
	eventBus    eventbus.EventBus                   // 事件总线
	auditLogger domainaudit.AuditLogger             // 审计日志
	connManager *websocket.ConnectionManager        // 连接管理器
	cfg         config.AuthConfig                   // 认证配置
	logger      *zap.Logger                         // 结构化日志
}

// NewLoginCommand 创建登录命令实例。
func NewLoginCommand(
	accountRepo domainaccount.AccountRepository,
	sessionRepo domainsession.SessionRepository,
	rateLimiter domainratelimit.RateLimiter,
	bruteForce *domainsecurity.BruteForceProtector,
	hasher domainaccount.PasswordHasher,
	tokenSigner domaintoken.TokenSigner,
	eventBus eventbus.EventBus,
	auditLogger domainaudit.AuditLogger,
	connManager *websocket.ConnectionManager,
	cfg config.AuthConfig,
	logger *zap.Logger,
) *LoginCommand {
	return &LoginCommand{
		accountRepo: accountRepo,
		sessionRepo: sessionRepo,
		rateLimiter: rateLimiter,
		bruteForce:  bruteForce,
		hasher:      hasher,
		tokenSigner: tokenSigner,
		eventBus:    eventBus,
		auditLogger: auditLogger,
		connManager: connManager,
		cfg:         cfg,
		logger:      logger,
	}
}

// Handle 执行登录命令。
//
// 严格遵循design.md 2.1.3.4节分支判定，覆盖spec 5.2.1全部规则与5.2.3全部异常场景。
// 分支顺序：频率校验先于账号查询（防DB消耗）、账号状态校验先于密码校验、
// 锁定校验先于密码校验、失败计数仅在密码错误后递增。
func (c *LoginCommand) Handle(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	credential := domainaccount.NewCredential(req.Username, req.Password)
	defer credential.ZeroPassword()

	now := time.Now().UnixMilli()
	c.logger.Info("登录请求接收",
		zap.String("username", req.Username),
		zap.String("source_ip", req.SourceIP),
	)

	if !c.rateLimiter.Allow(ctx, dimensionLoginIP, req.SourceIP) {
		c.logger.Warn("登录频率超限(IP)", zap.String("source_ip", req.SourceIP))
		return nil, gatewayerr.ErrLoginRateLimited
	}
	if !c.rateLimiter.Allow(ctx, dimensionLoginAccount, credential.Username()) {
		c.logger.Warn("登录频率超限(账号)", zap.String("username", req.Username))
		return nil, gatewayerr.ErrLoginRateLimited
	}

	account, err := c.accountRepo.FindByUsername(ctx, credential.Username())
	if err != nil {
		if errors.Is(err, gatewayerr.ErrAccountNotFoundSentinel) {
			c.logger.Warn("账号不存在", zap.String("username", req.Username))
			return nil, gatewayerr.ErrAccountNotFound
		}
		return nil, fmt.Errorf("账号查询失败: %w", gatewayerr.ErrLoginInternalError)
	}

	if account.IsBanned(now) {
		c.logger.Warn("账号已被封禁", zap.Int64("player_id", account.PlayerID()))
		_ = c.auditLogger.LogRecord(ctx, &domainaudit.AuditRecord{
			OpType: domainaudit.OpTypeBanIntercept, Subject: credential.Username(),
			Result: false, SourceIP: req.SourceIP, OpTime: now,
		})
		return nil, gatewayerr.ErrAccountBanned
	}

	locked, remainingSeconds, _ := c.bruteForce.CheckLocked(ctx, credential.Username())
	if locked {
		c.logger.Warn("账号已被锁定",
			zap.String("username", req.Username),
			zap.Int64("remaining_seconds", remainingSeconds),
		)
		return nil, gatewayerr.ErrAccountLocked
	}

	passwordMatch, err := account.VerifyPassword(ctx, credential.Password(), c.hasher)
	if err != nil {
		return nil, fmt.Errorf("密码校验失败: %w", gatewayerr.ErrLoginInternalError)
	}
	if !passwordMatch {
		_, _ = c.bruteForce.OnLoginFailure(ctx, credential.Username())
		_ = c.auditLogger.LogRecord(ctx, &domainaudit.AuditRecord{
			OpType: domainaudit.OpTypeLoginFailure, Subject: credential.Username(),
			Result: false, SourceIP: req.SourceIP, OpTime: now,
		})
		c.logger.Warn("密码错误", zap.String("username", req.Username))
		return nil, gatewayerr.ErrPasswordIncorrect
	}

	_ = c.bruteForce.OnLoginSuccess(ctx, credential.Username())

	if c.cfg.SingleLoginEnabled {
		if err := c.kickOutExistingSession(ctx, account.PlayerID(), now); err != nil {
			c.logger.Warn("单点登录踢下线失败",
				zap.Int64("player_id", account.PlayerID()),
				zap.Error(err),
			)
		}
	}

	session := domainsession.NewOnlineSession(account.PlayerID(), req.ConnID, now, c.cfg.TokenVersion, req.DeviceID)
	if err := c.sessionRepo.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("会话创建失败: %w", gatewayerr.ErrLoginInternalError)
	}

	expireTime := now + c.cfg.SessionTTLMs
	payload := domaintoken.TokenPayload{
		PlayerID:   account.PlayerID(),
		IssueTime:  now,
		ExpireTime: expireTime,
		Version:    c.cfg.TokenVersion,
	}
	tokenStr, err := c.tokenSigner.Sign(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("令牌签发失败: %w", gatewayerr.ErrLoginInternalError)
	}

	c.publishOnlineEvent(ctx, account.PlayerID(), now, req.SourceIP)

	_ = c.auditLogger.LogRecord(ctx, &domainaudit.AuditRecord{
		OpType: domainaudit.OpTypeLoginSuccess, Subject: credential.Username(),
		Result: true, SourceIP: req.SourceIP, OpTime: now,
		Extra: fmt.Sprintf(`{"player_id":%d}`, account.PlayerID()),
	})

	c.logger.Info("登录成功",
		zap.Int64("player_id", account.PlayerID()),
		zap.String("username", req.Username),
		zap.String("source_ip", req.SourceIP),
	)
	return &LoginResponse{
		AccessToken:  tokenStr,
		PlayerID:     account.PlayerID(),
		SessionTTLms: c.cfg.SessionTTLMs,
	}, nil
}

// kickOutExistingSession 单点登录踢下线：销毁旧会话+推送踢下线通知+发布下线事件。
func (c *LoginCommand) kickOutExistingSession(ctx context.Context, playerID int64, now int64) error {
	oldSession, err := c.sessionRepo.FindByPlayerID(ctx, playerID)
	if err != nil {
		if errors.Is(err, gatewayerr.ErrSessionNotFound) {
			return nil
		}
		return err
	}
	_ = oldSession.Destroy()
	if err := c.sessionRepo.Delete(ctx, playerID); err != nil {
		return err
	}
	_ = c.connManager.Send(ctx, playerID, []byte(kickOutMessage))
	c.publishOfflineEvent(ctx, playerID, now, domainevent.OfflineReasonKicked)
	return nil
}

// publishOnlineEvent 发布玩家上线事件。
func (c *LoginCommand) publishOnlineEvent(ctx context.Context, playerID, loginTime int64, sourceIP string) {
	event := &domainevent.PlayerOnlineEvent{
		PlayerID:  playerID,
		LoginTime: loginTime,
		SourceIP:  sourceIP,
	}
	domainEvt, err := event.ToDomainEvent(fmt.Sprintf("online-%d-%d", playerID, loginTime), 1)
	if err != nil {
		c.logger.Error("上线事件序列化失败", zap.Error(err))
		return
	}
	if err := c.eventBus.Publish(ctx, domainEvt); err != nil {
		c.logger.Error("上线事件发布失败", zap.Int64("player_id", playerID), zap.Error(err))
	}
}

// publishOfflineEvent 发布玩家下线事件。
func (c *LoginCommand) publishOfflineEvent(ctx context.Context, playerID, offlineTime int64, reason int) {
	event := &domainevent.PlayerOfflineEvent{
		PlayerID:    playerID,
		OfflineTime: offlineTime,
		Reason:      reason,
	}
	domainEvt, err := event.ToDomainEvent(fmt.Sprintf("offline-%d-%d", playerID, offlineTime), 1)
	if err != nil {
		c.logger.Error("下线事件序列化失败", zap.Error(err))
		return
	}
	if err := c.eventBus.Publish(ctx, domainEvt); err != nil {
		c.logger.Error("下线事件发布失败", zap.Int64("player_id", playerID), zap.Error(err))
	}
}
