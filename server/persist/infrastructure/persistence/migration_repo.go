// Package persistence Persist服务持久化层，实现domain层Repository接口。
// 表名引用shared/schema/tables常量（规范2），不硬编码表名字符串。
package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"insectworld/server/persist/domain/migration"
	"insectworld/server/shared/schema/tables"
)

// MigrationRepoImpl MigrationRepository接口实现，读写t_schema_migration表。
type MigrationRepoImpl struct {
	db *sql.DB // MySQL连接池
}

// NewMigrationRepoImpl 创建迁移仓储实现实例。
func NewMigrationRepoImpl(db *sql.DB) *MigrationRepoImpl {
	return &MigrationRepoImpl{db: db}
}

// SaveRecord 保存迁移执行记录到t_schema_migration表。
func (r *MigrationRepoImpl) SaveRecord(ctx context.Context, record *migration.MigrationRecord) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO `+tables.TSchemaMigration+` (id, version, description, status, execute_time, create_time) VALUES (?, ?, ?, ?, ?, ?)`,
		now, record.Version(), record.ScriptName(), record.Status(), record.Version(), now,
	)
	if err != nil {
		return fmt.Errorf("保存迁移记录失败: %w", err)
	}
	return nil
}

// FindExecutedVersions 查询已执行的迁移版本号列表。
func (r *MigrationRepoImpl) FindExecutedVersions(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT version FROM `+tables.TSchemaMigration+` WHERE status = ? ORDER BY version ASC`,
		migration.MigrationStatusExecuted,
	)
	if err != nil {
		return nil, fmt.Errorf("查询已执行迁移版本失败: %w", err)
	}
	defer rows.Close()

	var versions []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("扫描迁移版本号失败: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, nil
}