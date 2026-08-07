// Package migration Persist服务迁移执行器，读取shared/schema/migrations/脚本并按版本号顺序执行。
//
// infrastructure层技术适配，实现DDL迁移脚本的版本化管理与顺序执行。
// 迁移脚本从server/shared/schema/migrations/目录读取，按V<3位版本号>排序执行，
// 每个脚本事务包裹，成功后写入t_schema_migration记录（规范7日志埋点）。
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"

	"go.uber.org/zap"
)

// 迁移脚本文件名正则，匹配V<3位版本号>__<蛇形描述>.sql
var migrationFileRegex = regexp.MustCompile(`^V(\d{3})__[a-z][a-z0-9_]*\.sql$`)

// Executor 迁移脚本执行器，管理迁移脚本的发现与执行。
type Executor struct {
	db            *sql.DB     // MySQL连接池
	migrationsDir string      // 迁移脚本目录路径，指向server/shared/schema/migrations/
	logger        *zap.Logger // 结构化日志
	mu            sync.Mutex  // 互斥锁，保证迁移执行串行
}

// NewExecutor 创建迁移执行器实例。
func NewExecutor(db *sql.DB, migrationsDir string, logger *zap.Logger) *Executor {
	return &Executor{
		db:            db,
		migrationsDir: migrationsDir,
		logger:        logger,
	}
}

// MigrationFile 迁移脚本文件信息。
type MigrationFile struct {
	Version    int64  // 迁移版本号
	ScriptName string // 脚本文件名
	FilePath   string // 脚本文件完整路径
	Content    string // 脚本SQL内容
}

// DiscoverScripts 发现迁移脚本目录中的所有SQL脚本，按版本号排序。
func (e *Executor) DiscoverScripts() ([]MigrationFile, error) {
	var files []MigrationFile

	err := filepath.WalkDir(e.migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		matches := migrationFileRegex.FindStringSubmatch(d.Name())
		if matches == nil {
			return nil
		}

		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return fmt.Errorf("解析迁移版本号失败: %w", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取迁移脚本失败: %w", err)
		}

		files = append(files, MigrationFile{
			Version:    version,
			ScriptName: d.Name(),
			FilePath:   path,
			Content:    string(content),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("发现迁移脚本失败: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Version < files[j].Version
	})

	return files, nil
}

// Execute 执行单个迁移脚本，事务包裹。
func (e *Executor) Execute(ctx context.Context, file MigrationFile) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启迁移事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, file.Content); err != nil {
		e.logger.Error("迁移脚本执行失败",
			zap.Int64("version", file.Version),
			zap.String("script", file.ScriptName),
			zap.Error(err),
		)
		return fmt.Errorf("迁移脚本 %s 执行失败: %w", file.ScriptName, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交迁移事务失败: %w", err)
	}

	e.logger.Info("迁移脚本执行成功",
		zap.Int64("version", file.Version),
		zap.String("script", file.ScriptName),
	)
	return nil
}

// ExecutePending 执行所有未执行的迁移脚本。
func (e *Executor) ExecutePending(ctx context.Context, executedVersions []int64) error {
	files, err := e.DiscoverScripts()
	if err != nil {
		return err
	}

	executedSet := make(map[int64]bool)
	for _, v := range executedVersions {
		executedSet[v] = true
	}

	for _, file := range files {
		if executedSet[file.Version] {
			continue
		}
		if err := e.Execute(ctx, file); err != nil {
			return err
		}
	}
	return nil
}
