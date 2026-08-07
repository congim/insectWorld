// Package query Gateway服务application层查询，编排鉴权与读模型查询。
//
// application层不直接import infrastructure（规范3），通过domain层接口 + DI组装。
package query

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"
)

// AuthenticateQuery 鉴权查询，编排令牌校验流程。
//
// 编排顺序严格遵循design.md 2.1.3.5节：令牌缺失→格式校验→签名校验→
// 过期校验→黑名单查询→会话存在性查询→返回playerID。
type AuthenticateQuery struct {
	tokenSigner    domaintoken.TokenSigner         // 令牌签发器
	tokenBlacklist domaintoken.TokenBlacklist      // 令牌黑名单
	sessionRepo    domainsession.SessionRepository // 会话仓储
	logger         *zap.Logger                     // 结构化日志
}

// NewAuthenticateQuery 创建鉴权查询实例。
func NewAuthenticateQuery(
	tokenSigner domaintoken.TokenSigner,
	tokenBlacklist domaintoken.TokenBlacklist,
	sessionRepo domainsession.SessionRepository,
	logger *zap.Logger,
) *AuthenticateQuery {
	return &AuthenticateQuery{
		tokenSigner:    tokenSigner,
		tokenBlacklist: tokenBlacklist,
		sessionRepo:    sessionRepo,
		logger:         logger,
	}
}

// Handle 执行鉴权查询，返回玩家ID。
//
// 令牌缺失返回TOKEN_MISSING，格式/签名错误返回TOKEN_INVALID，
// 过期返回TOKEN_EXPIRED，黑名单返回TOKEN_INVALID，会话不存在返回TOKEN_INVALID。
// 鉴权日志不含密码/哈希/盐（spec 5.5.1 规则7）。
func (q *AuthenticateQuery) Handle(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, gatewayerr.ErrTokenMissing
	}

	payload, err := q.tokenSigner.Verify(ctx, token)
	if err != nil {
		if errors.Is(err, gatewayerr.ErrTokenInvalid) {
			q.logger.Warn("鉴权令牌签名不匹配，疑似伪造")
		}
		return 0, gatewayerr.ErrTokenInvalid
	}

	now := time.Now().UnixMilli()
	if now >= payload.ExpireTime {
		q.logger.Warn("鉴权令牌已过期",
			zap.Int64("player_id", payload.PlayerID),
		)
		return 0, gatewayerr.ErrTokenExpired
	}

	isInvalid, err := q.tokenBlacklist.IsInvalid(ctx, payload.PlayerID, payload.Version)
	if err != nil {
		q.logger.Warn("令牌黑名单查询降级，依赖会话存在性兜底",
			zap.Int64("player_id", payload.PlayerID),
			zap.Error(err),
		)
	} else if isInvalid {
		q.logger.Warn("鉴权令牌已在黑名单",
			zap.Int64("player_id", payload.PlayerID),
		)
		return 0, gatewayerr.ErrTokenInvalid
	}

	_, err = q.sessionRepo.FindByPlayerID(ctx, payload.PlayerID)
	if err != nil {
		if errors.Is(err, gatewayerr.ErrSessionNotFound) {
			q.logger.Warn("鉴权会话不存在",
				zap.Int64("player_id", payload.PlayerID),
			)
			return 0, gatewayerr.ErrTokenInvalid
		}
		return 0, gatewayerr.ErrTokenInvalid
	}

	return payload.PlayerID, nil
}
