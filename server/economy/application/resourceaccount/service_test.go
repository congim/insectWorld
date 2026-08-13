// Package resourceaccount 提供Economy稳定字符串资源账户应用服务。
package resourceaccount

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	economyerr "insectworld/server/economy/domain/errors"
	domainaccount "insectworld/server/economy/domain/resourceaccount"
)

type repositoryStub struct {
	change   domainaccount.Change // 最近一次资源变更
	reversed string               // 最近一次撤销操作ID
	balances map[string]int64     // 查询返回余额
}

func (r *repositoryStub) Apply(_ context.Context, change domainaccount.Change) error {
	r.change = change
	return nil
}

func (r *repositoryStub) Reverse(_ context.Context, operationID string, _ int64) error {
	r.reversed = operationID
	return nil
}

func (r *repositoryStub) Balances(_ context.Context, _ int64) (map[string]int64, error) {
	return r.balances, nil
}

// TestServiceDelegatesValidOperations 验证应用服务校验后向事务仓储传递稳定资源ID。
func TestServiceDelegatesValidOperations(t *testing.T) {
	t.Parallel()
	repository := &repositoryStub{balances: map[string]int64{"food": 80}}
	service := NewService(repository)
	amounts := map[string]int64{"food": 100}
	require.NoError(t, service.Change(context.Background(), 1, amounts, "create-1"))
	amounts["food"] = 999
	assert.Equal(t, int64(100), repository.change.Amounts["food"])
	require.NoError(t, service.Reverse(context.Background(), "create-1"))
	assert.Equal(t, "create-1", repository.reversed)
	balances, err := service.Balances(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(80), balances["food"])
}

// TestServiceRejectsInvalidOperations 验证非法玩家、空操作和零变更不会进入仓储。
func TestServiceRejectsInvalidOperations(t *testing.T) {
	t.Parallel()
	service := NewService(&repositoryStub{})
	testCases := []func() error{
		func() error { return service.Change(context.Background(), 0, map[string]int64{"food": 1}, "op") },
		func() error { return service.Change(context.Background(), 1, map[string]int64{}, "op") },
		func() error { return service.Change(context.Background(), 1, map[string]int64{"": 1}, "op") },
		func() error { return service.Change(context.Background(), 1, map[string]int64{"food": 0}, "op") },
		func() error { return service.Reverse(context.Background(), "") },
		func() error { _, err := service.Balances(context.Background(), 0); return err },
	}
	for _, run := range testCases {
		err := run()
		require.Error(t, err)
		assert.True(t, errors.Is(err, economyerr.ErrInvalidParams))
	}
}
