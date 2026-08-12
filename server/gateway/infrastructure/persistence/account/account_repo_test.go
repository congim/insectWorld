package account

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainaccount "insectworld/server/gateway/domain/account"
	gatewayerr "insectworld/server/gateway/domain/errors"
	"insectworld/server/shared/schema/tables"

	"go.uber.org/zap"
)

// newAccountRepoMock 创建sqlmock与账号仓储实例。
func newAccountRepoMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *AccountRepoMySQL) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	logger := zap.NewNop()
	repo := NewAccountRepoMySQL(db, logger)
	return db, mock, repo
}

// TestAccountRepoMySQL_Save 测试账号保存（UPSERT）。
func TestAccountRepoMySQL_Save(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	account := domainaccount.NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)

	mock.ExpectExec("INSERT INTO "+tables.TPlayerAccount).
		WithArgs(int64(1001), "testuser", "hash", "salt", domainaccount.AccountStatusNormal, "", int64(0), int64(1700000000000), "127.0.0.1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Save(context.Background(), account)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAccountRepoMySQL_SaveError 测试账号保存故障返回ErrAccountRepoUnavailable。
func TestAccountRepoMySQL_SaveError(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	account := domainaccount.NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)

	mock.ExpectExec("INSERT INTO " + tables.TPlayerAccount).
		WillReturnError(errors.New("db connection lost"))

	err := repo.Save(context.Background(), account)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountRepoUnavailable))
}

// TestAccountRepoMySQL_FindByID 测试按玩家ID查询账号。
func TestAccountRepoMySQL_FindByID(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"player_id", "username", "password_hash", "salt", "status", "ban_reason", "ban_expire_time", "register_time", "register_ip"}).
		AddRow(int64(1001), "testuser", "hash", "salt", domainaccount.AccountStatusNormal, "", int64(0), int64(1700000000000), "127.0.0.1")

	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerAccount).WithArgs(int64(1001)).WillReturnRows(rows)

	account, err := repo.FindByID(context.Background(), 1001)
	require.NoError(t, err)
	assert.Equal(t, int64(1001), account.PlayerID())
	assert.Equal(t, "testuser", account.Username())
	assert.Equal(t, "hash", account.PasswordHash())
	assert.Equal(t, domainaccount.AccountStatusNormal, account.Status())
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAccountRepoMySQL_FindByIDBanned 测试查询已封禁账号恢复封禁状态。
func TestAccountRepoMySQL_FindByIDBanned(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"player_id", "username", "password_hash", "salt", "status", "ban_reason", "ban_expire_time", "register_time", "register_ip"}).
		AddRow(int64(1001), "testuser", "hash", "salt", domainaccount.AccountStatusBanned, "违规", int64(0), int64(1700000000000), "127.0.0.1")

	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerAccount).WithArgs(int64(1001)).WillReturnRows(rows)

	account, err := repo.FindByID(context.Background(), 1001)
	require.NoError(t, err)
	assert.Equal(t, domainaccount.AccountStatusBanned, account.Status())
	assert.Equal(t, "违规", account.BanReason())
	assert.True(t, account.IsBanned(1700000000000))
}

// TestAccountRepoMySQL_FindByIDNotFound 测试查询不存在的账号返回ErrAccountNotFoundSentinel。
func TestAccountRepoMySQL_FindByIDNotFound(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerAccount).WithArgs(int64(9999)).WillReturnError(sql.ErrNoRows)

	_, err := repo.FindByID(context.Background(), 9999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountNotFoundSentinel))
}

// TestAccountRepoMySQL_FindByIDError 测试查询故障返回ErrAccountRepoUnavailable。
func TestAccountRepoMySQL_FindByIDError(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerAccount).WillReturnError(errors.New("db error"))

	_, err := repo.FindByID(context.Background(), 1001)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountRepoUnavailable))
}

// TestAccountRepoMySQL_FindByUsername 测试按用户名查询账号。
func TestAccountRepoMySQL_FindByUsername(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"player_id", "username", "password_hash", "salt", "status", "ban_reason", "ban_expire_time", "register_time", "register_ip"}).
		AddRow(int64(1001), "testuser", "hash", "salt", domainaccount.AccountStatusNormal, "", int64(0), int64(1700000000000), "127.0.0.1")

	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerAccount).WithArgs("testuser").WillReturnRows(rows)

	account, err := repo.FindByUsername(context.Background(), "testuser")
	require.NoError(t, err)
	assert.Equal(t, int64(1001), account.PlayerID())
	assert.Equal(t, "testuser", account.Username())
}

// TestAccountRepoMySQL_FindByUsernameNotFound 测试用户名查询不存在返回哨兵错误。
func TestAccountRepoMySQL_FindByUsernameNotFound(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT .* FROM " + tables.TPlayerAccount).WillReturnError(sql.ErrNoRows)

	_, err := repo.FindByUsername(context.Background(), "nouser")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountNotFoundSentinel))
}

// TestAccountRepoMySQL_ExistsByUsername 测试用户名存在性查询。
func TestAccountRepoMySQL_ExistsByUsername(t *testing.T) {
	t.Run("已存在", func(t *testing.T) {
		db, mock, repo := newAccountRepoMock(t)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery("SELECT COUNT\\(1\\) FROM " + tables.TPlayerAccount).WithArgs("testuser").WillReturnRows(rows)

		exists, err := repo.ExistsByUsername(context.Background(), "testuser")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("不存在", func(t *testing.T) {
		db, mock, repo := newAccountRepoMock(t)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery("SELECT COUNT\\(1\\) FROM " + tables.TPlayerAccount).WithArgs("nouser").WillReturnRows(rows)

		exists, err := repo.ExistsByUsername(context.Background(), "nouser")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// TestAccountRepoMySQL_ExistsByUsernameError 测试存在性查询故障。
func TestAccountRepoMySQL_ExistsByUsernameError(t *testing.T) {
	db, mock, repo := newAccountRepoMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT COUNT\\(1\\) FROM " + tables.TPlayerAccount).WillReturnError(errors.New("db error"))

	_, err := repo.ExistsByUsername(context.Background(), "testuser")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gatewayerr.ErrAccountRepoUnavailable))
}
