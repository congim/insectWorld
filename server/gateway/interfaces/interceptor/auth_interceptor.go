// Package interceptor Gateway服务interfaces层鉴权拦截器，实现业务请求鉴权接入。
//
// interfaces层依赖application层（规范3），通过AuthenticateQuery编排鉴权逻辑。
// 除注册/登录外所有业务请求必须鉴权（spec 5.5.1 规则1）。
package interceptor

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"insectworld/server/gateway/application/query"
	gatewayerr "insectworld/server/gateway/domain/errors"
)

// 鉴权白名单路径前缀，这些路径跳过鉴权（spec 5.5.1 规则1例外）。
var authWhitelist = map[string]bool{
	"/auth/register": true, // 注册接口跳过鉴权
	"/auth/login":    true, // 登录接口跳过鉴权
}

// AuthInterceptor 业务请求鉴权拦截器，在Gateway路由分发前校验令牌。
type AuthInterceptor struct {
	authQuery *query.AuthenticateQuery // 鉴权查询
	logger    *zap.Logger              // 结构化日志
}

// NewAuthInterceptor 创建鉴权拦截器实例。
func NewAuthInterceptor(authQuery *query.AuthenticateQuery, logger *zap.Logger) *AuthInterceptor {
	return &AuthInterceptor{
		authQuery: authQuery,
		logger:    logger,
	}
}

// Intercept 鉴权拦截，从请求头提取访问令牌并校验。
//
// path为请求路径，token为访问令牌。
// 白名单路径直接放行返回0，鉴权通过返回playerID，鉴权失败返回错误。
// 鉴权通过则将playerID注入ctx供下游业务服务使用。
func (i *AuthInterceptor) Intercept(ctx context.Context, path string, token string) (int64, context.Context, error) {
	if i.isWhitelisted(path) {
		return 0, ctx, nil
	}

	if token == "" {
		i.logger.Warn("鉴权令牌缺失", zap.String("path", path))
		return 0, ctx, gatewayerr.ErrTokenMissing
	}

	playerID, err := i.authQuery.Handle(ctx, token)
	if err != nil {
		i.logger.Warn("鉴权失败",
			zap.String("path", path),
			zap.Error(err),
		)
		return 0, ctx, err
	}

	newCtx := context.WithValue(ctx, playerIDKey{}, playerID)
	i.logger.Debug("鉴权通过",
		zap.String("path", path),
		zap.Int64("player_id", playerID),
	)
	return playerID, newCtx, nil
}

// isWhitelisted 判断路径是否在鉴权白名单中。
func (i *AuthInterceptor) isWhitelisted(path string) bool {
	for prefix := range authWhitelist {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// playerIDKey context键类型，避免键冲突。
type playerIDKey struct{}

// PlayerIDFromContext 从context提取玩家ID。
func PlayerIDFromContext(ctx context.Context) (int64, bool) {
	playerID, ok := ctx.Value(playerIDKey{}).(int64)
	return playerID, ok
}
