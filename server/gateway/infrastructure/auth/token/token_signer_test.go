// Package token 令牌签发与黑名单infrastructure层实现。
package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
	domaintoken "insectworld/server/gateway/domain/token"
)

// TestTokenSignerImpl_SignAndVerify 测试令牌签发与校验往返。
func TestTokenSignerImpl_SignAndVerify(t *testing.T) {
	logger := zap.NewNop()
	signer, err := NewTokenSignerImpl([]byte("test-signing-key"), logger)
	require.NoError(t, err)

	payload := domaintoken.TokenPayload{
		PlayerID:   1001,
		IssueTime:  1700000000000,
		ExpireTime: 1700000000000 + 3600000,
		Version:    1,
	}

	tokenStr, err := signer.Sign(context.Background(), payload)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	verifiedPayload, err := signer.Verify(context.Background(), tokenStr)
	require.NoError(t, err)
	assert.Equal(t, payload.PlayerID, verifiedPayload.PlayerID)
	assert.Equal(t, payload.IssueTime, verifiedPayload.IssueTime)
	assert.Equal(t, payload.ExpireTime, verifiedPayload.ExpireTime)
	assert.Equal(t, payload.Version, verifiedPayload.Version)
}

// TestTokenSignerImpl_Vify_Invalid 测试无效令牌校验。
func TestTokenSignerImpl_Verify_Invalid(t *testing.T) {
	logger := zap.NewNop()
	signer, err := NewTokenSignerImpl([]byte("test-signing-key"), logger)
	require.NoError(t, err)

	t.Run("格式错误", func(t *testing.T) {
		_, err := signer.Verify(context.Background(), "invalid-token")
		assert.Error(t, err)
	})

	t.Run("签名不匹配", func(t *testing.T) {
		payload := domaintoken.TokenPayload{PlayerID: 1001, IssueTime: 1700000000000, ExpireTime: 1700000000000 + 3600000, Version: 1}
		tokenStr, _ := signer.Sign(context.Background(), payload)
		tampered := tokenStr[:len(tokenStr)-1] + "x"
		_, err := signer.Verify(context.Background(), tampered)
		assert.Error(t, err)
	})
}

// TestNewTokenSignerImpl_EmptyKey 测试空密钥返回错误。
func TestNewTokenSignerImpl_EmptyKey(t *testing.T) {
	logger := zap.NewNop()
	_, err := NewTokenSignerImpl([]byte{}, logger)
	assert.Error(t, err)
}
