// Package persistence 提供Growth上下文MySQL持久化适配器。
package persistence

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/game/domain/building"
	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/game/domain/player"
	"insectworld/server/game/domain/training"
	"insectworld/server/shared/schema/tables"
)

func newDatabaseMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func duplicateEntryError() error {
	return &mysql.MySQLError{Number: mysqlDuplicateEntryCode, Message: "Duplicate entry"}
}

// TestPlayerRepositorySaveAndRestore 验证玩家档案写入与重启恢复保留配置版本。
func TestPlayerRepositorySaveAndRestore(t *testing.T) {
	t.Parallel()
	db, mock := newDatabaseMock(t)
	repository := NewPlayerRepository(db, zap.NewNop())
	profile, err := player.NewProfile(1, "faction", "玩家", 1000, "0.1.0", "create-1")
	require.NoError(t, err)
	insert := regexp.QuoteMeta("INSERT INTO " + tables.TPlayerProfile)
	mock.ExpectExec(insert).WithArgs(int64(1), "faction", "玩家", int32(1), int64(0), int64(1000), "0.1.0", "create-1").WillReturnResult(sqlmock.NewResult(0, 1))
	saved, created, err := repository.SaveIfAbsent(context.Background(), profile)
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "0.1.0", saved.ConfigVersion())

	rows := sqlmock.NewRows([]string{"player_id", "faction_id", "nickname", "level", "experience", "created_at", "config_version", "command_id"}).AddRow(int64(1), "faction", "玩家", int32(2), int64(30), int64(1000), "0.1.0", "create-1")
	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerProfile).WithArgs(int64(1)).WillReturnRows(rows)
	restored, err := repository.FindByPlayerID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int32(2), restored.Level())
	assert.Equal(t, int64(30), restored.Experience())
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestPlayerRepositoryDuplicateCommand 验证数据库唯一键竞争后返回同一命令结果。
func TestPlayerRepositoryDuplicateCommand(t *testing.T) {
	t.Parallel()
	db, mock := newDatabaseMock(t)
	repository := NewPlayerRepository(db, nil)
	profile, err := player.NewProfile(1, "faction", "玩家", 1000, "0.1.0", "create-1")
	require.NoError(t, err)
	mock.ExpectExec("INSERT INTO " + tables.TPlayerProfile).WillReturnError(duplicateEntryError())
	rows := sqlmock.NewRows([]string{"player_id", "faction_id", "nickname", "level", "experience", "created_at", "config_version", "command_id"}).AddRow(int64(1), "faction", "玩家", int32(1), int64(0), int64(1000), "0.1.0", "create-1")
	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerProfile).WithArgs("create-1").WillReturnRows(rows)
	saved, created, err := repository.SaveIfAbsent(context.Background(), profile)
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, int64(1), saved.PlayerID())
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestRepositoriesReturnStableNotFoundErrors 验证三类仓储区分内容定义与实例不存在。
func TestRepositoriesReturnStableNotFoundErrors(t *testing.T) {
	t.Parallel()
	db, mock := newDatabaseMock(t)
	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerProfile).WithArgs(int64(9)).WillReturnError(sql.ErrNoRows)
	_, err := NewPlayerRepository(db, nil).FindByPlayerID(context.Background(), 9)
	assert.True(t, errors.Is(err, gameerr.ErrPlayerNotFound))
	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerBuilding).WithArgs(int64(9)).WillReturnError(sql.ErrNoRows)
	_, err = NewBuildingRepository(db, nil).FindByID(context.Background(), 9)
	assert.True(t, errors.Is(err, gameerr.ErrBuildingNotFound))
	mock.ExpectQuery("SELECT .* FROM " + tables.TTrainingTask).WithArgs(int64(9)).WillReturnError(sql.ErrNoRows)
	_, err = NewTrainingRepository(db, nil).FindByID(context.Background(), 9)
	assert.True(t, errors.Is(err, gameerr.ErrTrainingNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestBuildingRepositorySaveAndIdempotentUpdate 验证建筑写入及重复状态保存。
func TestBuildingRepositorySaveAndIdempotentUpdate(t *testing.T) {
	t.Parallel()
	db, mock := newDatabaseMock(t)
	repository := NewBuildingRepository(db, zap.NewNop())
	aggregate, err := building.NewConstruction(10, 1, "farm", 1000, 2000, "0.1.0", "build-1")
	require.NoError(t, err)
	mock.ExpectExec("INSERT INTO "+tables.TPlayerBuilding).WithArgs(int64(10), int64(1), "farm", building.StatusConstructing, int64(1000), int64(2000), "0.1.0", "build-1").WillReturnResult(sqlmock.NewResult(0, 1))
	_, created, err := repository.SaveIfAbsent(context.Background(), aggregate)
	require.NoError(t, err)
	assert.True(t, created)
	require.NoError(t, aggregate.Complete(2000))
	mock.ExpectExec("UPDATE "+tables.TPlayerBuilding).WithArgs(building.StatusOperational, int64(10)).WillReturnResult(sqlmock.NewResult(0, 0))
	rows := sqlmock.NewRows([]string{"id", "player_id", "type_id", "status", "started_at", "complete_at", "config_version", "command_id"}).AddRow(int64(10), int64(1), "farm", building.StatusOperational, int64(1000), int64(2000), "0.1.0", "build-1")
	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerBuilding).WithArgs(int64(10)).WillReturnRows(rows)
	require.NoError(t, repository.Save(context.Background(), aggregate))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTrainingRepositorySaveAndRestore 验证训练任务持久化并恢复配置版本。
func TestTrainingRepositorySaveAndRestore(t *testing.T) {
	t.Parallel()
	db, mock := newDatabaseMock(t)
	repository := NewTrainingRepository(db, zap.NewNop())
	aggregate, err := training.NewTask(20, 1, 10, "worker", 2, 1000, 3000, "0.1.0", "train-1")
	require.NoError(t, err)
	mock.ExpectExec("INSERT INTO "+tables.TTrainingTask).WithArgs(int64(20), int64(1), int64(10), "worker", int64(2), training.StatusTraining, int64(1000), int64(3000), "0.1.0", "train-1").WillReturnResult(sqlmock.NewResult(0, 1))
	_, created, err := repository.SaveIfAbsent(context.Background(), aggregate)
	require.NoError(t, err)
	assert.True(t, created)
	rows := sqlmock.NewRows([]string{"id", "player_id", "building_id", "unit_type_id", "count", "status", "started_at", "complete_at", "config_version", "command_id"}).AddRow(int64(20), int64(1), int64(10), "worker", int64(2), training.StatusTraining, int64(1000), int64(3000), "0.1.0", "train-1")
	mock.ExpectQuery("SELECT .* FROM " + tables.TTrainingTask).WithArgs(int64(20)).WillReturnRows(rows)
	restored, err := repository.FindByID(context.Background(), 20)
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", restored.ConfigVersion())
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUnitRosterGrantIsTransactionalAndIdempotent 验证单位操作账本与名册在同一事务内提交。
func TestUnitRosterGrantIsTransactionalAndIdempotent(t *testing.T) {
	t.Parallel()
	db, mock := newDatabaseMock(t)
	roster := NewUnitRoster(db, zap.NewNop())
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO "+tables.TUnitGrantOperation).WithArgs("grant-1", int64(1), "worker", int64(2)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO "+tables.TUnitRoster).WithArgs(int64(1), "worker", int64(2)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, roster.Grant(context.Background(), 1, "worker", 2, "grant-1"))

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO "+tables.TUnitGrantOperation).WithArgs("grant-1", int64(1), "worker", int64(2)).WillReturnError(duplicateEntryError())
	rows := sqlmock.NewRows([]string{"player_id", "unit_type_id", "count"}).AddRow(int64(1), "worker", int64(2))
	mock.ExpectQuery("SELECT .* FROM " + tables.TUnitGrantOperation).WithArgs("grant-1").WillReturnRows(rows)
	mock.ExpectCommit()
	require.NoError(t, roster.Grant(context.Background(), 1, "worker", 2, "grant-1"))

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(2))
	mock.ExpectQuery("SELECT count FROM "+tables.TUnitRoster).WithArgs(int64(1), "worker").WillReturnRows(countRows)
	count, err := roster.Count(context.Background(), 1, "worker")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	require.NoError(t, mock.ExpectationsWereMet())
}
