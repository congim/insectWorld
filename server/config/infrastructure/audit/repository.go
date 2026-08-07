// Package audit Config服务配置变更审计日志持久化，独立存储配置操作审计记录。
//
// infrastructure层技术适配，实现domain层AuditRepository接口。
// 表名引用shared/schema/tables常量（规范2），不硬编码表名字符串。
// 审计日志独立存储，含操作人/操作时间/操作内容/操作结果/操作前后值（规范7）。
package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/shared/schema/tables"
)

// 操作类型枚举常量。
// 取值映射：1=创建 2=发布 3=回滚 4=删除
const (
	ActionCreate   = 1 // 创建配置版本
	ActionPublish  = 2 // 发布配置版本
	ActionRollback = 3 // 回滚配置版本
	ActionDelete   = 4 // 删除配置版本
)

// Repository 审计日志仓储，持久化配置变更操作记录。
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

// AuditRecord 审计日志记录。
type AuditRecord struct {
	VersionID   int64  // 版本ID
	Operator    string // 操作人
	Action      int    // 操作类型：1=创建 2=发布 3=回滚 4=删除
	BeforeValue string // 操作前值
	AfterValue  string // 操作后值
}

// Save 保存审计日志记录，表名引用shared/schema/tables.TConfigAuditLog常量。
func (r *Repository) Save(ctx context.Context, record AuditRecord) error {
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
