// Package persistence 提供Growth上下文MySQL持久化适配器。
package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"

	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/game/domain/training"
	"insectworld/server/shared/schema/tables"
)

// TrainingRepository 是单位训练任务MySQL仓储。
type TrainingRepository struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志器
}

// NewTrainingRepository 创建单位训练任务MySQL仓储。
func NewTrainingRepository(db *sql.DB, logger *zap.Logger) *TrainingRepository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TrainingRepository{db: db, logger: logger}
}

// FindByID 按训练任务ID恢复聚合。
func (r *TrainingRepository) FindByID(ctx context.Context, taskID int64) (*training.Task, error) {
	query := fmt.Sprintf(`SELECT id, player_id, building_id, unit_type_id, count, status, started_at, complete_at, config_version, command_id FROM %s WHERE id = ?`, tables.TTrainingTask)
	return r.queryOne(r.db.QueryRowContext(ctx, query, taskID))
}

// FindByCommandID 按训练命令幂等键恢复聚合。
func (r *TrainingRepository) FindByCommandID(ctx context.Context, commandID string) (*training.Task, error) {
	query := fmt.Sprintf(`SELECT id, player_id, building_id, unit_type_id, count, status, started_at, complete_at, config_version, command_id FROM %s WHERE command_id = ?`, tables.TTrainingTask)
	return r.queryOne(r.db.QueryRowContext(ctx, query, commandID))
}

// SaveIfAbsent 原子插入训练任务；重复命令返回既有任务。
func (r *TrainingRepository) SaveIfAbsent(ctx context.Context, aggregate *training.Task) (*training.Task, bool, error) {
	query := fmt.Sprintf(`INSERT INTO %s (id, player_id, building_id, unit_type_id, count, status, started_at, complete_at, config_version, command_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, tables.TTrainingTask)
	_, err := r.db.ExecContext(ctx, query, aggregate.ID(), aggregate.PlayerID(), aggregate.BuildingID(), aggregate.UnitTypeID(), aggregate.Count(), aggregate.Status(), aggregate.StartedAt(), aggregate.CompleteAt(), aggregate.ConfigVersion(), aggregate.CommandID())
	if err == nil {
		return aggregate.Clone(), true, nil
	}
	if !isDuplicateEntry(err) {
		r.logger.Error("训练任务持久化失败", zap.Int64("training_id", aggregate.ID()), zap.Int64("player_id", aggregate.PlayerID()), zap.Error(err))
		return nil, false, unavailable("训练任务持久化", err)
	}
	existing, findErr := r.FindByCommandID(ctx, aggregate.CommandID())
	if findErr != nil {
		if errors.Is(findErr, gameerr.ErrTrainingNotFound) {
			return nil, false, fmt.Errorf("训练任务ID冲突，taskID=%d: %w", aggregate.ID(), gameerr.ErrStateConflict)
		}
		return nil, false, findErr
	}
	if existing.PlayerID() != aggregate.PlayerID() || existing.BuildingID() != aggregate.BuildingID() || existing.UnitTypeID() != aggregate.UnitTypeID() || existing.Count() != aggregate.Count() || existing.ConfigVersion() != aggregate.ConfigVersion() {
		return nil, false, fmt.Errorf("训练命令载荷冲突，commandID=%s: %w", aggregate.CommandID(), gameerr.ErrStateConflict)
	}
	return existing, false, nil
}

// Save 保存训练任务状态；状态更新天然幂等。
func (r *TrainingRepository) Save(ctx context.Context, aggregate *training.Task) error {
	query := fmt.Sprintf(`UPDATE %s SET status = ? WHERE id = ?`, tables.TTrainingTask)
	result, err := r.db.ExecContext(ctx, query, aggregate.Status(), aggregate.ID())
	if err != nil {
		return unavailable("训练任务状态保存", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return unavailable("读取训练任务更新结果", err)
	}
	if affected == 0 {
		existing, findErr := r.FindByID(ctx, aggregate.ID())
		if findErr != nil {
			return findErr
		}
		if existing.Status() != aggregate.Status() {
			return fmt.Errorf("训练任务状态并发冲突，taskID=%d: %w", aggregate.ID(), gameerr.ErrStateConflict)
		}
	}
	return nil
}

func (r *TrainingRepository) queryOne(row scanner) (*training.Task, error) {
	var id int64
	var playerID int64
	var buildingID int64
	var unitTypeID string
	var count int64
	var status training.Status
	var startedAt int64
	var completeAt int64
	var configVersion string
	var commandID string
	if err := row.Scan(&id, &playerID, &buildingID, &unitTypeID, &count, &status, &startedAt, &completeAt, &configVersion, &commandID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gameerr.ErrTrainingNotFound
		}
		return nil, unavailable("训练任务查询", err)
	}
	aggregate, err := training.RestoreTask(id, playerID, buildingID, unitTypeID, count, status, startedAt, completeAt, configVersion, commandID)
	if err != nil {
		return nil, fmt.Errorf("恢复训练任务失败，taskID=%d: %w", id, err)
	}
	return aggregate, nil
}
