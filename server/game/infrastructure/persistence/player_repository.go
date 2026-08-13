// Package persistence 提供Growth上下文MySQL持久化适配器。
package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"

	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/game/domain/player"
	"insectworld/server/shared/schema/tables"
)

// PlayerRepository 是玩家成长档案MySQL仓储。
type PlayerRepository struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志器
}

// NewPlayerRepository 创建玩家成长档案MySQL仓储。
func NewPlayerRepository(db *sql.DB, logger *zap.Logger) *PlayerRepository {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PlayerRepository{db: db, logger: logger}
}

// FindByPlayerID 按玩家ID恢复成长档案。
func (r *PlayerRepository) FindByPlayerID(ctx context.Context, playerID int64) (*player.Profile, error) {
	query := fmt.Sprintf(`SELECT player_id, faction_id, nickname, level, experience, created_at, config_version, command_id FROM %s WHERE player_id = ?`, tables.TPlayerProfile)
	return r.queryOne(r.db.QueryRowContext(ctx, query, playerID))
}

// FindByCommandID 按创建命令幂等键恢复成长档案。
func (r *PlayerRepository) FindByCommandID(ctx context.Context, commandID string) (*player.Profile, error) {
	query := fmt.Sprintf(`SELECT player_id, faction_id, nickname, level, experience, created_at, config_version, command_id FROM %s WHERE command_id = ?`, tables.TPlayerProfile)
	return r.queryOne(r.db.QueryRowContext(ctx, query, commandID))
}

// SaveIfAbsent 原子插入玩家档案；重复命令返回既有档案。
func (r *PlayerRepository) SaveIfAbsent(ctx context.Context, profile *player.Profile) (*player.Profile, bool, error) {
	query := fmt.Sprintf(`INSERT INTO %s (player_id, faction_id, nickname, level, experience, created_at, config_version, command_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, tables.TPlayerProfile)
	_, err := r.db.ExecContext(ctx, query, profile.PlayerID(), profile.FactionID(), profile.Nickname(), profile.Level(), profile.Experience(), profile.CreatedAt(), profile.ConfigVersion(), profile.CommandID())
	if err == nil {
		return profile.Clone(), true, nil
	}
	if !isDuplicateEntry(err) {
		r.logger.Error("玩家档案持久化失败", zap.Int64("player_id", profile.PlayerID()), zap.Error(err))
		return nil, false, unavailable("玩家档案持久化", err)
	}
	existing, findErr := r.FindByCommandID(ctx, profile.CommandID())
	if findErr != nil {
		if errors.Is(findErr, gameerr.ErrPlayerNotFound) {
			return nil, false, fmt.Errorf("玩家ID已存在，playerID=%d: %w", profile.PlayerID(), gameerr.ErrPlayerAlreadyExists)
		}
		return nil, false, findErr
	}
	if existing.PlayerID() != profile.PlayerID() || existing.FactionID() != profile.FactionID() || existing.Nickname() != profile.Nickname() || existing.ConfigVersion() != profile.ConfigVersion() {
		return nil, false, fmt.Errorf("玩家创建命令载荷冲突，commandID=%s: %w", profile.CommandID(), gameerr.ErrStateConflict)
	}
	return existing, false, nil
}

func (r *PlayerRepository) queryOne(row scanner) (*player.Profile, error) {
	var playerID int64
	var factionID string
	var nickname string
	var level int32
	var experience int64
	var createdAt int64
	var configVersion string
	var commandID string
	if err := row.Scan(&playerID, &factionID, &nickname, &level, &experience, &createdAt, &configVersion, &commandID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gameerr.ErrPlayerNotFound
		}
		return nil, unavailable("玩家档案查询", err)
	}
	profile, err := player.RestoreProfile(playerID, factionID, nickname, level, experience, createdAt, configVersion, commandID)
	if err != nil {
		return nil, fmt.Errorf("恢复玩家档案失败，playerID=%d: %w", playerID, err)
	}
	return profile, nil
}
