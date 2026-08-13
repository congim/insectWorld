// Package persistence 提供Growth上下文MySQL持久化适配器。
package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"

	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/shared/schema/tables"
)

// UnitRoster 是玩家已训练单位名册MySQL适配器。
type UnitRoster struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志器
}

// NewUnitRoster 创建玩家单位名册MySQL适配器。
func NewUnitRoster(db *sql.DB, logger *zap.Logger) *UnitRoster {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &UnitRoster{db: db, logger: logger}
}

// Grant 在一个本地事务内记录幂等操作并增加单位数量。
func (r *UnitRoster) Grant(ctx context.Context, playerID int64, unitTypeID string, count int64, operationID string) error {
	if playerID <= 0 || unitTypeID == "" || count <= 0 || operationID == "" {
		return fmt.Errorf("单位入账参数非法，playerID=%d: %w", playerID, gameerr.ErrInvalidCommand)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return unavailable("开启单位入账事务", err)
	}
	defer tx.Rollback()
	operationQuery := fmt.Sprintf(`INSERT INTO %s (operation_id, player_id, unit_type_id, count) VALUES (?, ?, ?, ?)`, tables.TUnitGrantOperation)
	if _, err := tx.ExecContext(ctx, operationQuery, operationID, playerID, unitTypeID, count); err != nil {
		if !isDuplicateEntry(err) {
			return unavailable("记录单位入账操作", err)
		}
		var existingPlayerID int64
		var existingUnitTypeID string
		var existingCount int64
		lookupQuery := fmt.Sprintf(`SELECT player_id, unit_type_id, count FROM %s WHERE operation_id = ?`, tables.TUnitGrantOperation)
		if scanErr := tx.QueryRowContext(ctx, lookupQuery, operationID).Scan(&existingPlayerID, &existingUnitTypeID, &existingCount); scanErr != nil {
			return unavailable("查询单位入账操作", scanErr)
		}
		if existingPlayerID != playerID || existingUnitTypeID != unitTypeID || existingCount != count {
			return fmt.Errorf("单位入账幂等键载荷冲突，operationID=%s: %w", operationID, gameerr.ErrStateConflict)
		}
		if err := tx.Commit(); err != nil {
			return unavailable("提交重复单位入账事务", err)
		}
		return nil
	}
	rosterQuery := fmt.Sprintf(`INSERT INTO %s (player_id, unit_type_id, count) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE count = count + VALUES(count)`, tables.TUnitRoster)
	if _, err := tx.ExecContext(ctx, rosterQuery, playerID, unitTypeID, count); err != nil {
		return unavailable("增加单位名册数量", err)
	}
	if err := tx.Commit(); err != nil {
		return unavailable("提交单位入账事务", err)
	}
	r.logger.Info("单位名册入账完成", zap.String("operation_id", operationID), zap.Int64("player_id", playerID), zap.String("unit_type_id", unitTypeID), zap.Int64("count", count))
	return nil
}

// Count 返回玩家指定类型的已训练单位数量；不存在时返回0。
func (r *UnitRoster) Count(ctx context.Context, playerID int64, unitTypeID string) (int64, error) {
	query := fmt.Sprintf(`SELECT count FROM %s WHERE player_id = ? AND unit_type_id = ?`, tables.TUnitRoster)
	var count int64
	if err := r.db.QueryRowContext(ctx, query, playerID, unitTypeID).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, unavailable("单位名册查询", err)
	}
	return count, nil
}
