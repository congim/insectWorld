// Package token 令牌签发与黑名单infrastructure层实现。
//
// TokenSignerImpl使用HMAC-SHA256签名防伪造（spec 4.3 安全性3），
// 签名密钥不入日志（规范7脱敏）。Base64URL编码。
package token

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domaintoken "insectworld/server/gateway/domain/token"
)

// TokenSignerImpl HMAC-SHA256令牌签发器，实现TokenSigner接口。
type TokenSignerImpl struct {
	signingKey []byte      // HMAC签名密钥，不入日志（规范7脱敏）
	logger     *zap.Logger // 结构化日志
}

// NewTokenSignerImpl 创建令牌签发器实例。
//
// signingKey为HMAC签名密钥，为空时返回错误（密钥未配置）。
func NewTokenSignerImpl(signingKey []byte, logger *zap.Logger) (*TokenSignerImpl, error) {
	if len(signingKey) == 0 {
		return nil, fmt.Errorf("令牌签名密钥未配置: %w", gatewayerr.ErrTokenSignerUnavailable)
	}
	return &TokenSignerImpl{
		signingKey: signingKey,
		logger:     logger,
	}, nil
}

// Sign 对令牌负载计算签名，返回"payload.signature"格式的令牌字符串。
//
// payload序列化为JSON后Base64URL编码，HMAC-SHA256计算签名。
func (s *TokenSignerImpl) Sign(ctx context.Context, payload domaintoken.TokenPayload) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("令牌负载序列化失败: %w", err)
	}

	payloadB64 := base64.URLEncoding.EncodeToString(payloadBytes)
	signature := s.computeHMAC(payloadB64)
	tokenStr := payloadB64 + "." + signature
	return tokenStr, nil
}

// Verify 校验令牌字符串的签名与格式，返回令牌负载。
//
// 签名不匹配或格式错误返回ErrTokenInvalid。
func (s *TokenSignerImpl) Verify(ctx context.Context, tokenStr string) (domaintoken.TokenPayload, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return domaintoken.TokenPayload{}, gatewayerr.ErrTokenInvalid
	}

	payloadB64 := parts[0]
	signature := parts[1]
	expectedSig := s.computeHMAC(payloadB64)
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		s.logger.Warn("令牌签名不匹配，疑似伪造",
			zap.String("payload_prefix", safePrefix(payloadB64, 8)),
		)
		return domaintoken.TokenPayload{}, gatewayerr.ErrTokenInvalid
	}

	payloadBytes, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return domaintoken.TokenPayload{}, gatewayerr.ErrTokenInvalid
	}

	var payload domaintoken.TokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return domaintoken.TokenPayload{}, gatewayerr.ErrTokenInvalid
	}
	return payload, nil
}

// computeHMAC 计算HMAC-SHA256签名并返回Base64URL编码字符串。
func (s *TokenSignerImpl) computeHMAC(data string) string {
	h := hmac.New(sha256.New, s.signingKey)
	h.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// safePrefix 安全返回字符串前n个字符，避免越界。
func safePrefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
