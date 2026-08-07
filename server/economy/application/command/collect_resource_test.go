// Package command Economy服务application层命令，编排domain层聚合根与配置查询。
// 本文件定义CollectResourceHandler的单元测试。
package command

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/economy/domain/wallet"
	"insectworld/server/shared/pkg/config/mock"
)

// mockWalletRepository Wallet仓储mock实现。
type mockWalletRepository struct {
	mu      sync.Mutex
	wallet  *wallet.PlayerWallet
	loadErr error
	saveErr error
}

// LoadWallet 加载钱包的mock实现。
func (r *mockWalletRepository) LoadWallet(ctx context.Context, playerID int64) (*wallet.PlayerWallet, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r.wallet, nil
}

// SaveWallet 保存钱包的mock实现。
func (r *mockWalletRepository) SaveWallet(ctx context.Context, w *wallet.PlayerWallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wallet = w
	if r.saveErr != nil {
		return r.saveErr
	}
	return nil
}

// mockOutbox Outbox mock实现。
type mockOutbox struct {
	events []any
	err    error
}

// Append 写Outbox的mock实现。
func (o *mockOutbox) Append(ctx context.Context, event any) error {
	if o.err != nil {
		return o.err
	}
	o.events = append(o.events, event)
	return nil
}

// TestCollectResourceHandler_Success 测试采集资源成功。
func TestCollectResourceHandler_Success(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()

	w := wallet.NewPlayerWallet(1)
	repo := &mockWalletRepository{wallet: w}
	outbox := &mockOutbox{}

	handler := NewCollectResourceHandler(repo, cfg, outbox, logger)

	err := handler.Handle(context.Background(), CollectResourceCommand{
		PlayerID:        1,
		ResourcePointID: 10,
		ResourceType:    100,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(100), repo.wallet.GetBalance(100))
}

// TestCollectResourceHandler_LoadError 测试加载钱包失败。
func TestCollectResourceHandler_LoadError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()

	repo := &mockWalletRepository{loadErr: assert.AnError}
	outbox := &mockOutbox{}

	handler := NewCollectResourceHandler(repo, cfg, outbox, logger)

	err := handler.Handle(context.Background(), CollectResourceCommand{
		PlayerID:        1,
		ResourcePointID: 10,
		ResourceType:    100,
	})
	assert.Error(t, err)
}

// TestCollectResourceHandler_SaveError 测试保存钱包失败。
func TestCollectResourceHandler_SaveError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()

	w := wallet.NewPlayerWallet(1)
	repo := &mockWalletRepository{wallet: w, saveErr: assert.AnError}
	outbox := &mockOutbox{}

	handler := NewCollectResourceHandler(repo, cfg, outbox, logger)

	err := handler.Handle(context.Background(), CollectResourceCommand{
		PlayerID:        1,
		ResourcePointID: 10,
		ResourceType:    100,
	})
	assert.Error(t, err)
}
