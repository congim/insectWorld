// Package auth 认证domain service，编排账号聚合根与令牌签发器完成登录/鉴权/登出。
// AuthService是domain service而非聚合根，不持有可变状态，只编排Account聚合根与Token值对象。
// 对应spec.md 5.1.7.1节Account上下文功能2"认证"。
package auth

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
	"insectworld/server/gateway/domain/token"
)

// LoginResult 登录认证结果值对象。
type LoginResult struct {
	PlayerID   int64  // 玩家ID
	Token      string // 签发的访问令牌字符串
	ExpireTime int64  // 令牌过期时间戳（毫秒）
}

// AuthService 认证domain service，编排登录/鉴权/登出流程。
type AuthService struct {
	tokenSigner    token.TokenSigner    // 令牌签发器，infrastructure层注入
	tokenBlacklist token.TokenBlacklist // 令牌黑名单，infrastructure层注入
	logger         *zap.Logger          // 结构化日志器（规范7）
}

// NewAuthService 创建认证domain service实例。
// tokenSigner和tokenBlacklist由infrastructure层实现，cmd/main.go组装时注入。
func NewAuthService(
	tokenSigner token.TokenSigner,
	tokenBlacklist token.TokenBlacklist,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		tokenSigner:    tokenSigner,
		tokenBlacklist: tokenBlacklist,
		logger:         logger,
	}
}

// IssueToken 签发访问令牌。
// 在application层完成账号密码校验与封禁校验后调用，生成HMAC-SHA256签名的令牌。
// playerID为玩家ID，tokenVersion为令牌版本号（登出时递增），now为当前时间戳（毫秒），
// ttl为令牌有效期（毫秒）。
func (s *AuthService) IssueToken(ctx context.Context, playerID int64, tokenVersion int, now int64, ttl int64) (*LoginResult, error) {
	expireTime := now + ttl
	accessToken := token.NewAccessToken(playerID, now, expireTime, tokenVersion)

	tokenStr, err := s.tokenSigner.Sign(ctx, accessToken.ToPayload())
	if err != nil {
		s.logger.Error("令牌签发失败",
			zap.Int64("player_id", playerID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("令牌签发失败: %w", gatewayerr.ErrTokenSignerUnavailable)
	}
	accessToken.SetSignature(tokenStr)

	s.logger.Info("令牌签发成功",
		zap.Int64("player_id", playerID),
		zap.Int64("expire_time", expireTime),
	)

	return &LoginResult{
		PlayerID:   playerID,
		Token:      tokenStr,
		ExpireTime: expireTime,
	}, nil
}

// VerifyToken 校验访问令牌有效性。
// 校验链：签名校验→过期校验→黑名单校验。
// tokenStr为待校验的令牌字符串，now为当前时间戳（毫秒）。
// 返回令牌负载（含玩家ID），校验失败返回对应错误。
func (s *AuthService) VerifyToken(ctx context.Context, tokenStr string, now int64) (token.TokenPayload, error) {
	payload, err := s.tokenSigner.Verify(ctx, tokenStr)
	if err != nil {
		return token.TokenPayload{}, fmt.Errorf("令牌签名校验失败: %w", gatewayerr.ErrTokenInvalid)
	}

	if now >= payload.ExpireTime {
		return token.TokenPayload{}, fmt.Errorf("令牌已过期: %w", gatewayerr.ErrTokenExpired)
	}

	isInvalid, err := s.tokenBlacklist.IsInvalid(ctx, payload.PlayerID, payload.Version)
	if err != nil {
		s.logger.Warn("令牌黑名单查询降级",
			zap.Int64("player_id", payload.PlayerID),
			zap.Error(err),
		)
	}
	if isInvalid {
		return token.TokenPayload{}, fmt.Errorf("令牌已失效: %w", gatewayerr.ErrTokenInvalid)
	}

	return payload, nil
}

// Logout 登出，将令牌加入黑名单使其即刻失效。
// playerID为玩家ID，tokenVersion为令牌版本号，remainingTTL为令牌剩余有效期（秒）。
func (s *AuthService) Logout(ctx context.Context, playerID int64, tokenVersion int, remainingTTL int64) error {
	err := s.tokenBlacklist.Invalidate(ctx, playerID, tokenVersion, remainingTTL)
	if err != nil {
		s.logger.Error("令牌黑名单写入失败",
			zap.Int64("player_id", playerID),
			zap.Error(err),
		)
		return fmt.Errorf("登出失败: %w", gatewayerr.ErrTokenBlacklistUnavailable)
	}

	s.logger.Info("登出成功", zap.Int64("player_id", playerID))
	return nil
}
