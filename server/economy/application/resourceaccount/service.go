// Package resourceaccount 提供Economy稳定字符串资源账户应用服务。
package resourceaccount

import (
	"context"
	"fmt"
	"math"
	"time"

	economyerr "insectworld/server/economy/domain/errors"
	domainaccount "insectworld/server/economy/domain/resourceaccount"
)

// Service 是模块化单体内供其他上下文调用的资源账户API。
type Service struct {
	repository domainaccount.Repository // Economy资源账户事务仓储
}

// NewService 创建资源账户应用服务。
func NewService(repository domainaccount.Repository) *Service {
	return &Service{repository: repository}
}

// Change 幂等应用资源变更，同一operationID不得携带不同载荷。
func (s *Service) Change(ctx context.Context, playerID int64, amounts map[string]int64, operationID string) error {
	if err := validateChange(playerID, amounts, operationID); err != nil {
		return err
	}
	return s.repository.Apply(ctx, domainaccount.Change{OperationID: operationID, PlayerID: playerID, Amounts: cloneAmounts(amounts), CreatedAt: time.Now().UnixMilli()})
}

// Reverse 幂等撤销已经应用的资源操作；不存在的操作视为无需补偿。
func (s *Service) Reverse(ctx context.Context, operationID string) error {
	if operationID == "" {
		return fmt.Errorf("资源撤销缺少操作ID: %w", economyerr.ErrInvalidParams)
	}
	return s.repository.Reverse(ctx, operationID, time.Now().UnixMilli())
}

// Balances 返回玩家全部稳定资源ID余额。
func (s *Service) Balances(ctx context.Context, playerID int64) (map[string]int64, error) {
	if playerID <= 0 {
		return nil, fmt.Errorf("资源账户玩家ID非法，playerID=%d: %w", playerID, economyerr.ErrInvalidParams)
	}
	return s.repository.Balances(ctx, playerID)
}

func validateChange(playerID int64, amounts map[string]int64, operationID string) error {
	if playerID <= 0 || operationID == "" || len(amounts) == 0 {
		return fmt.Errorf("资源变更参数非法，playerID=%d: %w", playerID, economyerr.ErrInvalidParams)
	}
	for resourceID, amount := range amounts {
		if resourceID == "" || amount == 0 || amount == math.MinInt64 {
			return fmt.Errorf("资源变更项非法，resourceID=%s，amount=%d: %w", resourceID, amount, economyerr.ErrInvalidParams)
		}
	}
	return nil
}

func cloneAmounts(source map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(source))
	for resourceID, amount := range source {
		result[resourceID] = amount
	}
	return result
}
