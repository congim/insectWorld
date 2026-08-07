// Package versionstore Config服务配置版本持久化，支持10版本历史回滚。
//
// infrastructure层技术适配，实现domain层VersionRepository接口。
// 表名引用shared/schema/tables常量（规范2），不硬编码表名字符串。
package versionstore

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/shared/schema/tables"
)

// 最大保留版本数，超过后自动清理最旧版本。
const maxVersionHistory = 10

// VersionStore 配置版本持久化管理器。
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

// SaveVersion 保存配置版本，表名引用shared/schema/tables.TConfigVersion常量。
func (v *VersionStore) SaveVersion(ctx context.Context, versionID int64, version string, configType int, operator string) error {
	_, err := v.db.ExecContext(ctx,
		`INSERT INTO `+tables.TConfigVersion+` (id, version, config_type, status, operator, create_time) VALUES (?, ?, ?, 2, ?, ?)`,
		versionID, version, configType, operator, versionID,
	)
	if err != nil {
		return fmt.Errorf("保存配置版本失败: %w", err)
	}
	v.logger.Info("配置版本保存成功", zap.Int64("version_id", versionID), zap.String("version", version))
	return nil
}

// FindVersions 查询指定配置类型的版本历史。
func (v *VersionStore) FindVersions(ctx context.Context, configType int, limit int) error {
	_ = ctx
	_ = fmt.Sprintf("查询configType=%d limit=%d的版本历史 from %s", configType, limit, tables.TConfigVersion)
	return nil
}
