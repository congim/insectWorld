// Package command Gateway服务application层命令，编排用户认证操作。
package command

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	domainaccount "insectworld/server/gateway/domain/account"
	domainaudit "insectworld/server/gateway/domain/audit"
	domainconfig "insectworld/server/gateway/domain/config"
	gatewayerr "insectworld/server/gateway/domain/errors"
	domainidgen "insectworld/server/gateway/domain/idgen"
	domainratelimit "insectworld/server/gateway/domain/ratelimit"
)

// 限流维度常量（规范1就近归属）。
const (
	dimensionRegisterIP = "register:ip" // 注册限流维度：按IP
)

// RegisterCommand 注册命令，编排用户注册流程。
//
// 编排顺序遵循spec 5.1.2：校验用户名格式→校验密码强度→校验注册频率→
// 查询用户名唯一性→生成玩家ID→密码加盐哈希→构造聚合根→持久化→审计日志→返回。
type RegisterCommand struct {
	accountRepo domainaccount.AccountRepository // 账号仓储
	rateLimiter domainratelimit.RateLimiter     // 限流器
	idGenerator domainidgen.IDGenerator         // ID生成器
	hasher      domainaccount.PasswordHasher    // 密码哈希器
	auditLogger domainaudit.AuditLogger         // 审计日志
	cfg         domainconfig.AuthConfig         // 认证配置
	logger      *zap.Logger                     // 结构化日志
}

// NewRegisterCommand 创建注册命令实例。
func NewRegisterCommand(
	accountRepo domainaccount.AccountRepository,
	rateLimiter domainratelimit.RateLimiter,
	idGenerator domainidgen.IDGenerator,
	hasher domainaccount.PasswordHasher,
	auditLogger domainaudit.AuditLogger,
	cfg domainconfig.AuthConfig,
	logger *zap.Logger,
) *RegisterCommand {
	return &RegisterCommand{
		accountRepo: accountRepo,
		rateLimiter: rateLimiter,
		idGenerator: idGenerator,
		hasher:      hasher,
		auditLogger: auditLogger,
		cfg:         cfg,
		logger:      logger,
	}
}

// Handle 执行注册命令。
//
// 严格遵循spec 5.1.2交互流程，覆盖spec 5.1.1全部规则与5.1.3全部异常场景。
// 注册不自动登录不签发令牌不建立会话（spec 5.1.1 规则7）。
func (c *RegisterCommand) Handle(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	credential := domainaccount.NewCredential(req.Username, req.Password)
	defer credential.ZeroPassword()

	now := time.Now().UnixMilli()
	c.logger.Info("注册请求接收",
		zap.String("username", req.Username),
		zap.String("source_ip", req.SourceIP),
	)

	if err := domainaccount.ValidateUsernameFormat(credential.Username(), c.cfg.UsernameMinLength, c.cfg.UsernameMaxLength); err != nil {
		c.logger.Warn("用户名格式校验失败", zap.String("username", req.Username), zap.Error(err))
		return nil, err
	}

	if err := domainaccount.ValidatePasswordStrength(credential.Password(), c.cfg.PasswordMinLength, c.cfg.PasswordMaxLength); err != nil {
		c.logger.Warn("密码强度校验失败", zap.String("username", req.Username), zap.Error(err))
		return nil, err
	}

	if !c.rateLimiter.Allow(ctx, dimensionRegisterIP, req.SourceIP) {
		c.logger.Warn("注册频率超限", zap.String("username", req.Username), zap.String("source_ip", req.SourceIP))
		return nil, gatewayerr.ErrRegisterRateLimited
	}

	exists, err := c.accountRepo.ExistsByUsername(ctx, credential.Username())
	if err != nil {
		return nil, fmt.Errorf("用户名唯一性查询失败: %w", gatewayerr.ErrRegisterInternalError)
	}
	if exists {
		c.logger.Warn("用户名已存在", zap.String("username", req.Username))
		return nil, gatewayerr.ErrUsernameAlreadyExists
	}

	playerID, err := c.idGenerator.NextID(ctx)
	if err != nil {
		return nil, fmt.Errorf("玩家ID生成失败: %w", gatewayerr.ErrRegisterInternalError)
	}

	hash, salt, err := c.hasher.Hash(ctx, credential.Password())
	if err != nil {
		return nil, err
	}

	account := domainaccount.NewPlayerAccount(playerID, credential.Username(), hash, salt, req.SourceIP, now)
	if err := c.accountRepo.Save(ctx, account); err != nil {
		return nil, fmt.Errorf("账号持久化失败: %w", gatewayerr.ErrRegisterInternalError)
	}

	_ = c.auditLogger.LogRecord(ctx, &domainaudit.AuditRecord{
		OpType:   domainaudit.OpTypeRegisterSuccess,
		Subject:  credential.Username(),
		Result:   true,
		SourceIP: req.SourceIP,
		OpTime:   now,
		Extra:    fmt.Sprintf(`{"player_id":%d}`, playerID),
	})

	c.logger.Info("注册成功",
		zap.Int64("player_id", playerID),
		zap.String("username", req.Username),
		zap.String("source_ip", req.SourceIP),
	)
	return &RegisterResponse{PlayerID: playerID}, nil
}
