// Package persistence Economy服务持久化层。
package persistence

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	economyerr "insectworld/server/economy/domain/errors"
	"insectworld/server/economy/domain/resourceaccount"
)

func newResourceRepositoryMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *ResourceAccountRepository) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock, NewResourceAccountRepository(db, zap.NewNop())
}

func duplicateResourceOperation() error {
	return &mysql.MySQLError{Number: mysqlDuplicateEntryCode, Message: "Duplicate entry"}
}

// TestResourceAccountApplyAndRetry 验证首次入账与重复投递只改变一次余额。
func TestResourceAccountApplyAndRetry(t *testing.T) {
	t.Parallel()
	_, mock, repository := newResourceRepositoryMock(t)
	change := resourceaccount.Change{OperationID: "create-1", PlayerID: 1, Amounts: map[string]int64{"food": 100}, CreatedAt: 1000}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO "+TableResourceOperation).WithArgs("create-1", int64(1), []byte(`{"food":100}`), resourceaccount.OperationStatusApplied, int64(1000)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT amount FROM "+TableResourceAccountBalance).WithArgs(int64(1), "food").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO "+TableResourceAccountBalance).WithArgs(int64(1), "food", int64(100), int64(1000)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repository.Apply(context.Background(), change))

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO " + TableResourceOperation).WillReturnError(duplicateResourceOperation())
	rows := sqlmock.NewRows([]string{"player_id", "amounts_json", "status"}).AddRow(int64(1), []byte(`{"food":100}`), resourceaccount.OperationStatusApplied)
	mock.ExpectQuery("SELECT .* FROM " + TableResourceOperation).WithArgs("create-1").WillReturnRows(rows)
	mock.ExpectCommit()
	require.NoError(t, repository.Apply(context.Background(), change))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestResourceAccountRejectsInsufficientBalance 验证扣款不足时回滚操作账本和全部余额更新。
func TestResourceAccountRejectsInsufficientBalance(t *testing.T) {
	t.Parallel()
	_, mock, repository := newResourceRepositoryMock(t)
	change := resourceaccount.Change{OperationID: "build-1", PlayerID: 1, Amounts: map[string]int64{"food": -25}, CreatedAt: 1000}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO " + TableResourceOperation).WillReturnResult(sqlmock.NewResult(0, 1))
	rows := sqlmock.NewRows([]string{"amount"}).AddRow(int64(10))
	mock.ExpectQuery("SELECT amount FROM "+TableResourceAccountBalance).WithArgs(int64(1), "food").WillReturnRows(rows)
	mock.ExpectRollback()
	err := repository.Apply(context.Background(), change)
	require.Error(t, err)
	assert.True(t, errors.Is(err, economyerr.ErrResourceInsufficient))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestResourceAccountReverseAndReapply 验证补偿撤销后相同操作可在重试时重新应用。
func TestResourceAccountReverseAndReapply(t *testing.T) {
	t.Parallel()
	_, mock, repository := newResourceRepositoryMock(t)
	mock.ExpectBegin()
	operationRows := sqlmock.NewRows([]string{"player_id", "amounts_json", "status"}).AddRow(int64(1), []byte(`{"food":100}`), resourceaccount.OperationStatusApplied)
	mock.ExpectQuery("SELECT .* FROM " + TableResourceOperation).WithArgs("create-1").WillReturnRows(operationRows)
	balanceRows := sqlmock.NewRows([]string{"amount"}).AddRow(int64(100))
	mock.ExpectQuery("SELECT amount FROM "+TableResourceAccountBalance).WithArgs(int64(1), "food").WillReturnRows(balanceRows)
	mock.ExpectExec("UPDATE "+TableResourceAccountBalance).WithArgs(int64(0), int64(2000), int64(1), "food").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE "+TableResourceOperation).WithArgs(resourceaccount.OperationStatusReversed, int64(2000), "create-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repository.Reverse(context.Background(), "create-1", 2000))

	change := resourceaccount.Change{OperationID: "create-1", PlayerID: 1, Amounts: map[string]int64{"food": 100}, CreatedAt: 3000}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO " + TableResourceOperation).WillReturnError(duplicateResourceOperation())
	reversedRows := sqlmock.NewRows([]string{"player_id", "amounts_json", "status"}).AddRow(int64(1), []byte(`{"food":100}`), resourceaccount.OperationStatusReversed)
	mock.ExpectQuery("SELECT .* FROM " + TableResourceOperation).WithArgs("create-1").WillReturnRows(reversedRows)
	zeroRows := sqlmock.NewRows([]string{"amount"}).AddRow(int64(0))
	mock.ExpectQuery("SELECT amount FROM "+TableResourceAccountBalance).WithArgs(int64(1), "food").WillReturnRows(zeroRows)
	mock.ExpectExec("UPDATE "+TableResourceAccountBalance).WithArgs(int64(100), int64(3000), int64(1), "food").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE "+TableResourceOperation).WithArgs(resourceaccount.OperationStatusApplied, "create-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repository.Apply(context.Background(), change))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestResourceAccountBalances 验证资源余额查询使用稳定字符串ID。
func TestResourceAccountBalances(t *testing.T) {
	t.Parallel()
	_, mock, repository := newResourceRepositoryMock(t)
	rows := sqlmock.NewRows([]string{"resource_id", "amount"}).AddRow("food", int64(75)).AddRow("wood", int64(20))
	mock.ExpectQuery("SELECT resource_id, amount FROM " + TableResourceAccountBalance).WithArgs(int64(1)).WillReturnRows(rows)
	balances, err := repository.Balances(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"food": 75, "wood": 20}, balances)
	require.NoError(t, mock.ExpectationsWereMet())
}
