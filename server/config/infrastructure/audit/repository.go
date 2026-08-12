// Package audit Config服务配置变更审计日志持久化，独立存储配置操作审计记录。
//
// infrastructure层技术适配，实现domain层AuditStorage接口。
// 表名引用shared/schema/tables常量（规范2），不硬编码表名字符串。
// 审计日志独立存储，含操作人/操作时间/操作内容/操作结果/操作前后值（规范7）。
package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/config/domain"
	"insectworld/server/shared/schema/tables"
)

// Repository 审计日志仓储，实现domain.AuditStorage接口。
type Repository struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志
}

// NewRepository 创建审计日志仓储实例。
func NewRepository(db *sql.DB, logger *zap.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Save 保存审计日志记录，实现domain.AuditStorage接口。
// 表名引用shared/schema/tables.TConfigAuditLog常量（规范2）。
// db为nil时返回错误，保证不panic（main.go未配置数据库时降级）。
func (r *Repository) Save(ctx context.Context, record domain.AuditRecord) error {
	if r.db == nil {
		return fmt.Errorf("数据库未初始化，无法保存审计日志")
	}
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO `+tables.TConfigAuditLog+` (id, version_id, operator, action, before_value, after_value, create_time) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		now, record.VersionID, record.Operator, record.Action, record.BeforeValue, record.AfterValue, now,
	)
	if err != nil {
		return fmt.Errorf("保存审计日志失败: %w", err)
	}
	r.logger.Info("审计日志保存成功",
		zap.Int64("version_id", record.VersionID),
		zap.String("operator", record.Operator),
		zap.Int("action", record.Action),
	)
	return nil
}

// 确保Repository实现domain.AuditStorage接口（编译期检查）。
var _ domain.AuditStorage = (*Repository)(nil)
