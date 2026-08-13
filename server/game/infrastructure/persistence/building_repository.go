// Package persistence 提供Growth上下文MySQL持久化适配器。
package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/game/domain/building"
	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/shared/schema/tables"
)

// BuildingRepository 是玩家建筑MySQL仓储。
type BuildingRepository struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志器
}

// NewBuildingRepository 创建玩家建筑MySQL仓储。
func NewBuildingRepository(db *sql.DB, logger *zap.Logger) *BuildingRepository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &BuildingRepository{db: db, logger: logger}
}

// FindByID 按建筑实例ID恢复聚合。
func (r *BuildingRepository) FindByID(ctx context.Context, buildingID int64) (*building.Building, error) {
	query := fmt.Sprintf(`SELECT id, player_id, type_id, status, started_at, complete_at, config_version, command_id FROM %s WHERE id = ?`, tables.TPlayerBuilding)
	return r.queryOne(r.db.QueryRowContext(ctx, query, buildingID))
}

// FindByCommandID 按建造命令幂等键恢复聚合。
func (r *BuildingRepository) FindByCommandID(ctx context.Context, commandID string) (*building.Building, error) {
	query := fmt.Sprintf(`SELECT id, player_id, type_id, status, started_at, complete_at, config_version, command_id FROM %s WHERE command_id = ?`, tables.TPlayerBuilding)
	return r.queryOne(r.db.QueryRowContext(ctx, query, commandID))
}

// SaveIfAbsent 原子插入建筑；重复命令返回既有建筑。
func (r *BuildingRepository) SaveIfAbsent(ctx context.Context, aggregate *building.Building) (*building.Building, bool, error) {
	query := fmt.Sprintf(`INSERT INTO %s (id, player_id, type_id, status, started_at, complete_at, config_version, command_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, tables.TPlayerBuilding)
	_, err := r.db.ExecContext(ctx, query, aggregate.ID(), aggregate.PlayerID(), aggregate.TypeID(), aggregate.Status(), aggregate.StartedAt(), aggregate.CompleteAt(), aggregate.ConfigVersion(), aggregate.CommandID())
	if err == nil {
		return aggregate.Clone(), true, nil
	}
	if !isDuplicateEntry(err) {
		r.logger.Error("建筑持久化失败", zap.Int64("building_id", aggregate.ID()), zap.Int64("player_id", aggregate.PlayerID()), zap.Error(err))
		return nil, false, unavailable("建筑持久化", err)
	}
	existing, findErr := r.FindByCommandID(ctx, aggregate.CommandID())
	if findErr != nil {
		if errors.Is(findErr, gameerr.ErrBuildingNotFound) {
			return nil, false, fmt.Errorf("建筑ID冲突，buildingID=%d: %w", aggregate.ID(), gameerr.ErrStateConflict)
		}
		return nil, false, findErr
	}
	if existing.PlayerID() != aggregate.PlayerID() || existing.TypeID() != aggregate.TypeID() || existing.ConfigVersion() != aggregate.ConfigVersion() {
		return nil, false, fmt.Errorf("建造命令载荷冲突，commandID=%s: %w", aggregate.CommandID(), gameerr.ErrStateConflict)
	}
	return existing, false, nil
}

// Save 保存建筑状态；状态更新天然幂等。
func (r *BuildingRepository) Save(ctx context.Context, aggregate *building.Building) error {
	query := fmt.Sprintf(`UPDATE %s SET status = ? WHERE id = ?`, tables.TPlayerBuilding)
	result, err := r.db.ExecContext(ctx, query, aggregate.Status(), aggregate.ID())
	if err != nil {
		return unavailable("建筑状态保存", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return unavailable("读取建筑更新结果", err)
	}
	if affected == 0 {
		existing, findErr := r.FindByID(ctx, aggregate.ID())
		if findErr != nil {
			return findErr
		}
		if existing.Status() != aggregate.Status() {
			return fmt.Errorf("建筑状态并发冲突，buildingID=%d: %w", aggregate.ID(), gameerr.ErrStateConflict)
		}
	}
	return nil
}

func (r *BuildingRepository) queryOne(row scanner) (*building.Building, error) {
	var id int64
	var playerID int64
	var typeID string
	var status building.Status
	var startedAt int64
	var completeAt int64
	var configVersion string
	var commandID string
	if err := row.Scan(&id, &playerID, &typeID, &status, &startedAt, &completeAt, &configVersion, &commandID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gameerr.ErrBuildingNotFound
		}
		return nil, unavailable("建筑查询", err)
	}
	aggregate, err := building.RestoreBuilding(id, playerID, typeID, status, startedAt, completeAt, configVersion, commandID)
	if err != nil {
		return nil, fmt.Errorf("恢复建筑失败，buildingID=%d: %w", id, err)
	}
	return aggregate, nil
}
