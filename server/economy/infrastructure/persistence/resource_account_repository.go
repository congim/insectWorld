// Package persistence Economy服务持久化层。
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"

	economyerr "insectworld/server/economy/domain/errors"
	"insectworld/server/economy/domain/resourceaccount"
)

const mysqlDuplicateEntryCode uint16 = 1062

// ResourceAccountRepository 是稳定字符串资源账户MySQL事务仓储。
type ResourceAccountRepository struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志器
}

// NewResourceAccountRepository 创建资源账户MySQL事务仓储。
func NewResourceAccountRepository(db *sql.DB, logger *zap.Logger) *ResourceAccountRepository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ResourceAccountRepository{db: db, logger: logger}
}

// Apply 在一个本地事务内写入操作账本并更新全部资源余额。
func (r *ResourceAccountRepository) Apply(ctx context.Context, change resourceaccount.Change) error {
	payload, err := json.Marshal(change.Amounts)
	if err != nil {
		return fmt.Errorf("资源操作序列化失败: %w", economyerr.ErrInvalidParams)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return unavailable("开启资源变更事务", err)
	}
	defer tx.Rollback()

	insertOperation := fmt.Sprintf(`INSERT INTO %s (operation_id, player_id, amounts_json, status, created_at, reversed_at) VALUES (?, ?, ?, ?, ?, 0)`, TableResourceOperation)
	_, err = tx.ExecContext(ctx, insertOperation, change.OperationID, change.PlayerID, payload, resourceaccount.OperationStatusApplied, change.CreatedAt)
	if err != nil {
		if !isDuplicateEntry(err) {
			return unavailable("写入资源操作账本", err)
		}
		return r.handleExisting(ctx, tx, change, payload)
	}
	if err := applyAmounts(ctx, tx, change.PlayerID, change.Amounts, change.CreatedAt); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return unavailable("提交资源变更事务", err)
	}
	r.logger.Info("资源账户变更完成", zap.String("operation_id", change.OperationID), zap.Int64("player_id", change.PlayerID), zap.String("result", "success"))
	return nil
}

// Reverse 在一个本地事务内撤销已应用操作；不存在或已撤销时保持幂等。
func (r *ResourceAccountRepository) Reverse(ctx context.Context, operationID string, reversedAt int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return unavailable("开启资源撤销事务", err)
	}
	defer tx.Rollback()
	playerID, amounts, status, err := loadOperation(ctx, tx, operationID)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit()
	}
	if err != nil {
		return unavailable("查询资源撤销操作", err)
	}
	if status == resourceaccount.OperationStatusReversed {
		return tx.Commit()
	}
	inverse := make(map[string]int64, len(amounts))
	for resourceID, amount := range amounts {
		inverse[resourceID] = -amount
	}
	if err := applyAmounts(ctx, tx, playerID, inverse, reversedAt); err != nil {
		return err
	}
	update := fmt.Sprintf(`UPDATE %s SET status = ?, reversed_at = ? WHERE operation_id = ?`, TableResourceOperation)
	if _, err := tx.ExecContext(ctx, update, resourceaccount.OperationStatusReversed, reversedAt, operationID); err != nil {
		return unavailable("更新资源撤销状态", err)
	}
	if err := tx.Commit(); err != nil {
		return unavailable("提交资源撤销事务", err)
	}
	return nil
}

// Balances 返回玩家全部资源余额。
func (r *ResourceAccountRepository) Balances(ctx context.Context, playerID int64) (map[string]int64, error) {
	query := fmt.Sprintf(`SELECT resource_id, amount FROM %s WHERE player_id = ?`, TableResourceAccountBalance)
	rows, err := r.db.QueryContext(ctx, query, playerID)
	if err != nil {
		return nil, unavailable("查询资源余额", err)
	}
	defer rows.Close()
	result := make(map[string]int64)
	for rows.Next() {
		var resourceID string
		var amount int64
		if err := rows.Scan(&resourceID, &amount); err != nil {
			return nil, unavailable("扫描资源余额", err)
		}
		result[resourceID] = amount
	}
	if err := rows.Err(); err != nil {
		return nil, unavailable("遍历资源余额", err)
	}
	return result, nil
}

func (r *ResourceAccountRepository) handleExisting(ctx context.Context, tx *sql.Tx, change resourceaccount.Change, payload []byte) error {
	playerID, existingAmounts, status, err := loadOperation(ctx, tx, change.OperationID)
	if err != nil {
		return unavailable("查询既有资源操作", err)
	}
	existingPayload, err := json.Marshal(existingAmounts)
	if err != nil || playerID != change.PlayerID || string(existingPayload) != string(payload) {
		return fmt.Errorf("资源操作幂等键载荷冲突，operationID=%s: %w", change.OperationID, economyerr.ErrOperationConflict)
	}
	if status == resourceaccount.OperationStatusApplied {
		if err := tx.Commit(); err != nil {
			return unavailable("提交重复资源操作事务", err)
		}
		return nil
	}
	if err := applyAmounts(ctx, tx, change.PlayerID, change.Amounts, change.CreatedAt); err != nil {
		return err
	}
	update := fmt.Sprintf(`UPDATE %s SET status = ?, reversed_at = 0 WHERE operation_id = ?`, TableResourceOperation)
	if _, err := tx.ExecContext(ctx, update, resourceaccount.OperationStatusApplied, change.OperationID); err != nil {
		return unavailable("重新应用资源操作", err)
	}
	if err := tx.Commit(); err != nil {
		return unavailable("提交重新应用资源事务", err)
	}
	return nil
}

func loadOperation(ctx context.Context, tx *sql.Tx, operationID string) (int64, map[string]int64, resourceaccount.OperationStatus, error) {
	query := fmt.Sprintf(`SELECT player_id, amounts_json, status FROM %s WHERE operation_id = ? FOR UPDATE`, TableResourceOperation)
	var playerID int64
	var payload []byte
	var status resourceaccount.OperationStatus
	if err := tx.QueryRowContext(ctx, query, operationID).Scan(&playerID, &payload, &status); err != nil {
		return 0, nil, 0, err
	}
	amounts := make(map[string]int64)
	if err := json.Unmarshal(payload, &amounts); err != nil {
		return 0, nil, 0, err
	}
	return playerID, amounts, status, nil
}

func applyAmounts(ctx context.Context, tx *sql.Tx, playerID int64, amounts map[string]int64, nowMs int64) error {
	resourceIDs := make([]string, 0, len(amounts))
	for resourceID := range amounts {
		resourceIDs = append(resourceIDs, resourceID)
	}
	sort.Strings(resourceIDs)
	for _, resourceID := range resourceIDs {
		current, exists, err := lockBalance(ctx, tx, playerID, resourceID)
		if err != nil {
			return err
		}
		delta := amounts[resourceID]
		if delta < 0 && current < -delta {
			return fmt.Errorf("资源余额不足，playerID=%d，resourceID=%s，balance=%d，required=%d: %w", playerID, resourceID, current, -delta, economyerr.ErrResourceInsufficient)
		}
		if delta > 0 && current > math.MaxInt64-delta {
			return fmt.Errorf("资源余额溢出，playerID=%d，resourceID=%s: %w", playerID, resourceID, economyerr.ErrOperationConflict)
		}
		next := current + delta
		if exists {
			update := fmt.Sprintf(`UPDATE %s SET amount = ?, updated_at = ? WHERE player_id = ? AND resource_id = ?`, TableResourceAccountBalance)
			if _, err := tx.ExecContext(ctx, update, next, nowMs, playerID, resourceID); err != nil {
				return unavailable("更新资源余额", err)
			}
		} else {
			insert := fmt.Sprintf(`INSERT INTO %s (player_id, resource_id, amount, updated_at) VALUES (?, ?, ?, ?)`, TableResourceAccountBalance)
			if _, err := tx.ExecContext(ctx, insert, playerID, resourceID, next, nowMs); err != nil {
				return unavailable("创建资源余额", err)
			}
		}
	}
	return nil
}

func lockBalance(ctx context.Context, tx *sql.Tx, playerID int64, resourceID string) (int64, bool, error) {
	query := fmt.Sprintf(`SELECT amount FROM %s WHERE player_id = ? AND resource_id = ? FOR UPDATE`, TableResourceAccountBalance)
	var amount int64
	if err := tx.QueryRowContext(ctx, query, playerID, resourceID).Scan(&amount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, unavailable("锁定资源余额", err)
	}
	return amount, true, nil
}

func isDuplicateEntry(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == mysqlDuplicateEntryCode
}

func unavailable(operation string, err error) error {
	return fmt.Errorf("%s失败: %v: %w", operation, err, economyerr.ErrRepositoryUnavailable)
}
