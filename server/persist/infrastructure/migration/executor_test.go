// Package migration Persist服务迁移执行器，读取并执行版本化SQL脚本。
package migration

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newExecutorMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *Executor) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock, NewExecutor(db, t.TempDir(), zap.NewNop())
}

// TestExecuteRunsStatementsSequentially 验证多语句迁移按声明顺序逐条执行。
func TestExecuteRunsStatementsSequentially(t *testing.T) {
	_, mock, executor := newExecutorMock(t)
	mock.ExpectExec("CREATE TABLE t_first").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE t_second").WillReturnResult(sqlmock.NewResult(0, 0))
	file := MigrationFile{Version: 4, ScriptName: "V004__growth.sql", Content: "-- 说明\nCREATE TABLE t_first (id BIGINT);\n\nCREATE TABLE t_second (id BIGINT);"}
	require.NoError(t, executor.Execute(context.Background(), file))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestExecuteStopsOnStatementFailure 验证任一语句失败后不再执行后续语句。
func TestExecuteStopsOnStatementFailure(t *testing.T) {
	_, mock, executor := newExecutorMock(t)
	mock.ExpectExec("CREATE TABLE t_first").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE t_second").WillReturnError(errors.New("ddl failed"))
	file := MigrationFile{Version: 4, ScriptName: "V004__growth.sql", Content: "CREATE TABLE t_first (id BIGINT); CREATE TABLE t_second (id BIGINT);"}
	err := executor.Execute(context.Background(), file)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "第2条语句")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestExecuteRejectsEmptyScript 验证空迁移脚本不会被误标记为成功。
func TestExecuteRejectsEmptyScript(t *testing.T) {
	_, mock, executor := newExecutorMock(t)
	err := executor.Execute(context.Background(), MigrationFile{ScriptName: "V004__empty.sql", Content: "-- 只有注释"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不包含可执行语句")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSplitStatementsFiltersComments 验证迁移切分忽略空行和独立注释行。
func TestSplitStatementsFiltersComments(t *testing.T) {
	t.Parallel()
	statements := splitStatements("-- 注释\nALTER TABLE t_a ADD COLUMN value INT;\n\n-- 注释\nCREATE INDEX idx_a_value ON t_a(value);")
	assert.Equal(t, []string{"ALTER TABLE t_a ADD COLUMN value INT", "CREATE INDEX idx_a_value ON t_a(value)"}, statements)
}
