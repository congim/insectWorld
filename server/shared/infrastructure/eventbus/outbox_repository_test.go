// Package eventbus 提供共享事件契约的MySQL基础设施适配。
package eventbus

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainevent "insectworld/server/shared/pkg/eventbus"
	"insectworld/server/shared/schema/tables"
)

func newOutboxMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *OutboxRepository) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock, NewOutboxRepository(db)
}

// TestClaimPendingUsesLeaseTransaction 验证领取事件使用行锁、跳过已锁记录并提交租约。
func TestClaimPendingUsesLeaseTransaction(t *testing.T) {
	t.Parallel()
	_, mock, repository := newOutboxMock(t)
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"event_id", "aggregate_id", "event_type", "event_version", "payload", "status", "retry_count", "create_time", "publish_time", "available_time"}).
		AddRow("event-1", int64(1), "auth.player_registered", 1, []byte(`{}`), domainevent.OutboxStatusPending, 0, int64(1000), int64(0), int64(0))
	mock.ExpectQuery("SELECT .* FROM "+tables.TOutbox).WithArgs("auth.player_registered", domainevent.OutboxStatusPending, domainevent.OutboxStatusFailed, int64(2000), domainevent.OutboxStatusProcessing, int64(2000), 10).WillReturnRows(rows)
	mock.ExpectExec("UPDATE "+tables.TOutbox).WithArgs(domainevent.OutboxStatusProcessing, int64(7000), "event-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	records, err := repository.ClaimPending(context.Background(), 2000, []string{"auth.player_registered"}, 10, 7000)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, domainevent.OutboxStatusProcessing, records[0].Status)
	assert.Equal(t, int64(7000), records[0].AvailableTime)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestClaimPendingRejectsEmptyEventTypes 验证仓储不会无范围领取其他上下文事件。
func TestClaimPendingRejectsEmptyEventTypes(t *testing.T) {
	t.Parallel()
	_, _, repository := newOutboxMock(t)
	_, err := repository.ClaimPending(context.Background(), 2000, nil, 10, 7000)
	require.Error(t, err)
}

// TestMarkFailedSchedulesRetry 验证失败状态递增重试次数并保存下次领取时间。
func TestMarkFailedSchedulesRetry(t *testing.T) {
	t.Parallel()
	_, mock, repository := newOutboxMock(t)
	mock.ExpectExec("UPDATE "+tables.TOutbox).WithArgs(domainevent.OutboxStatusFailed, int64(9000), "consumer failed", "event-1", domainevent.OutboxStatusProcessing).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repository.MarkFailed(context.Background(), "event-1", 9000, "consumer failed"))
	require.NoError(t, mock.ExpectationsWereMet())
}
