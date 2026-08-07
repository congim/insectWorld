// Package migration 迁移聚合根，维护DDL迁移脚本的执行记录与版本状态。
//
// 迁移聚合根管理数据库Schema迁移的生命周期，读取server/shared/schema/migrations/目录的
// SQL脚本，按版本号顺序执行，保证数据库Schema的版本演进可追溯、可回滚。
package migration

import (
	"context"
	"fmt"
)

// 迁移状态枚举常量。
// 取值映射：1=待执行 2=已执行 3=执行失败
const (
	MigrationStatusPending  = 1 // 待执行状态，迁移脚本尚未执行
	MigrationStatusExecuted = 2 // 已执行状态，迁移脚本执行成功
	MigrationStatusFailed   = 3 // 执行失败状态，迁移脚本执行出错
)

// MigrationRecord 迁移执行记录，存储单个迁移脚本的执行信息。
type MigrationRecord struct {
	version     int64  // 迁移版本号，对应脚本文件名的V<3位版本号>
	scriptName  string // 迁移脚本文件名，如V001__add_player_last_login_ip.sql
	status      int    // 执行状态：1=待执行 2=已执行 3=执行失败
	executeTime int64  // 执行时间戳，毫秒级
	errorMsg    string // 错误信息，执行失败时记录错误详情
}

// NewMigrationRecord 创建迁移执行记录实例。
func NewMigrationRecord(version int64, scriptName string) *MigrationRecord {
	return &MigrationRecord{
		version:    version,
		scriptName: scriptName,
		status:     MigrationStatusPending,
	}
}

// Version 返回迁移版本号。
func (m *MigrationRecord) Version() int64 { return m.version }

// ScriptName 返回迁移脚本文件名。
func (m *MigrationRecord) ScriptName() string { return m.scriptName }

// Status 返回执行状态。
func (m *MigrationRecord) Status() int { return m.status }

// MarkExecuted 标记迁移已执行成功。
func (m *MigrationRecord) MarkExecuted(executeTime int64) error {
	if m.status == MigrationStatusExecuted {
		return fmt.Errorf("迁移版本 %d 已执行，禁止重复执行", m.version)
	}
	m.status = MigrationStatusExecuted
	m.executeTime = executeTime
	return nil
}

// MarkFailed 标记迁移执行失败。
func (m *MigrationRecord) MarkFailed(executeTime int64, errMsg string) {
	m.status = MigrationStatusFailed
	m.executeTime = executeTime
	m.errorMsg = errMsg
}

// Migration 迁移聚合根，管理一批迁移脚本的执行。
type Migration struct {
	records []*MigrationRecord // 迁移记录列表，按版本号排序
}

// NewMigration 创建迁移聚合根实例。
func NewMigration() *Migration {
	return &Migration{
		records: make([]*MigrationRecord, 0),
	}
}

// AddRecord 添加迁移记录。
func (m *Migration) AddRecord(record *MigrationRecord) {
	m.records = append(m.records, record)
}

// Records 返回所有迁移记录。
func (m *Migration) Records() []*MigrationRecord { return m.records }

// MigrationRepository 迁移仓储接口，在domain层声明，infrastructure层实现。
type MigrationRepository interface {
	// SaveRecord 保存迁移执行记录。
	SaveRecord(ctx context.Context, record *MigrationRecord) error
	// FindExecutedVersions 查询已执行的迁移版本号列表。
	FindExecutedVersions(ctx context.Context) ([]int64, error)
}