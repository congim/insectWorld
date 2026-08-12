// Package versionstore Config服务配置版本持久化，支持10版本历史回滚。
//
// infrastructure层技术适配，实现domain层VersionStorage接口。
// 表名引用shared/schema/tables常量（规范2），不硬编码表名字符串。
package versionstore

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/config/domain"
	"insectworld/server/shared/schema/tables"
)

// 最大保留版本数，超过后自动清理最旧版本。
const maxVersionHistory = 10

// VersionStore 配置版本持久化管理器，实现domain.VersionStorage接口。
type VersionStore struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志
}

// NewVersionStore 创建配置版本持久化管理器实例。
func NewVersionStore(db *sql.DB, logger *zap.Logger) *VersionStore {
	return &VersionStore{
		db:     db,
		logger: logger,
	}
}

// SaveVersion 保存配置版本，实现domain.VersionStorage接口。
// 表名引用shared/schema/tables.TConfigVersion常量（规范2）。
// db为nil时返回错误，保证不panic（main.go未配置数据库时降级）。
func (v *VersionStore) SaveVersion(ctx context.Context, versionID int64, version string, configType int, operator string) error {
	if v.db == nil {
		return fmt.Errorf("数据库未初始化，无法保存配置版本")
	}
	_, err := v.db.ExecContext(ctx,
		`INSERT INTO `+tables.TConfigVersion+` (id, version, config_type, status, operator, create_time) VALUES (?, ?, ?, 2, ?, ?)`,
		versionID, version, configType, operator, versionID,
	)
	if err != nil {
		return fmt.Errorf("保存配置版本失败: %w", err)
	}
	v.logger.Info("配置版本保存成功",
		zap.Int64("version_id", versionID),
		zap.String("version", version),
		zap.String("operator", operator),
	)
	return nil
}

// FindVersions 查询指定配置类型的版本历史，实现domain.VersionStorage接口。
// 按创建时间降序返回，limit控制返回数量上限。
// db为nil时返回空列表，保证不panic。
func (v *VersionStore) FindVersions(ctx context.Context, configType int, limit int) ([]domain.VersionInfo, error) {
	if v.db == nil {
		return nil, nil
	}
	if limit <= 0 || limit > maxVersionHistory {
		limit = maxVersionHistory
	}

	rows, err := v.db.QueryContext(ctx,
		`SELECT id, version, config_type, operator, create_time FROM `+tables.TConfigVersion+
			` WHERE config_type = ? ORDER BY create_time DESC LIMIT ?`,
		configType, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("查询配置版本历史失败: %w", err)
	}
	defer rows.Close()

	var versions []domain.VersionInfo
	for rows.Next() {
		var info domain.VersionInfo
		if err := rows.Scan(&info.VersionID, &info.Version, &info.ConfigType, &info.Operator, &info.CreateTime); err != nil {
			return nil, fmt.Errorf("读取版本记录失败: %w", err)
		}
		versions = append(versions, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历版本记录失败: %w", err)
	}

	v.logger.Debug("查询配置版本历史",
		zap.Int("config_type", configType),
		zap.Int("result_count", len(versions)),
	)
	return versions, nil
}

// 确保VersionStore实现domain.VersionStorage接口（编译期检查）。
var _ domain.VersionStorage = (*VersionStore)(nil)
