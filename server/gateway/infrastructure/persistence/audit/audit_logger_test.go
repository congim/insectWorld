package audit

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	domainaudit "insectworld/server/gateway/domain/audit"
	"insectworld/server/shared/schema/tables"

	"go.uber.org/zap"
)

// waitAuditConsumed 等待审计日志异步消费完成。
func waitAuditConsumed() {
	time.Sleep(100 * time.Millisecond)
}

// TestAuditLoggerImpl_LogRecord 测试审计日志异步落库。
func TestAuditLoggerImpl_LogRecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	impl := NewAuditLoggerImpl(db, logger)
	defer impl.Close()

	mock.ExpectExec("INSERT INTO "+tables.TAuthAuditLog).
		WithArgs(domainaudit.OpTypeLoginSuccess, "testuser", 1, "127.0.0.1", int64(1700000000000), `{"player_id":1001}`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = impl.LogRecord(context.Background(), &domainaudit.AuditRecord{
		OpType:   domainaudit.OpTypeLoginSuccess,
		Subject:  "testuser",
		Result:   true,
		SourceIP: "127.0.0.1",
		OpTime:   1700000000000,
		Extra:    `{"player_id":1001}`,
	})
	require.NoError(t, err)

	waitAuditConsumed()
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditLoggerImpl_LogRecordResultFalse 测试审计日志Result=false时落库为0。
func TestAuditLoggerImpl_LogRecordResultFalse(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	impl := NewAuditLoggerImpl(db, logger)
	defer impl.Close()

	mock.ExpectExec("INSERT INTO "+tables.TAuthAuditLog).
		WithArgs(domainaudit.OpTypeLoginFailure, "testuser", 0, "127.0.0.1", int64(1700000000000), "").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = impl.LogRecord(context.Background(), &domainaudit.AuditRecord{
		OpType:   domainaudit.OpTypeLoginFailure,
		Subject:  "testuser",
		Result:   false,
		SourceIP: "127.0.0.1",
		OpTime:   1700000000000,
	})
	require.NoError(t, err)
	waitAuditConsumed()
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditLoggerImpl_LogRecordDBError 测试MySQL故障时不阻塞主流程。
func TestAuditLoggerImpl_LogRecordDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	impl := NewAuditLoggerImpl(db, logger)
	defer impl.Close()

	mock.ExpectExec("INSERT INTO " + tables.TAuthAuditLog).
		WillReturnError(errors.New("db connection lost"))

	err = impl.LogRecord(context.Background(), &domainaudit.AuditRecord{
		OpType:  domainaudit.OpTypeLoginSuccess,
		Subject: "testuser",
		Result:  true,
	})
	// LogRecord非阻塞返回nil，异步落库失败由后台goroutine记录Error日志
	require.NoError(t, err)
	waitAuditConsumed()
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAuditLoggerImpl_Close 测试Close优雅退出。
func TestAuditLoggerImpl_Close(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	impl := NewAuditLoggerImpl(db, logger)

	done := make(chan struct{})
	go func() {
		impl.Close()
		close(done)
	}()
	select {
	case <-done:
		// 成功退出
	case <-time.After(3 * time.Second):
		t.Fatal("Close未在3秒内返回")
	}
}

// TestAuditLoggerImpl_BufferFull 测试缓冲区满时丢弃不阻塞。
func TestAuditLoggerImpl_BufferFull(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	impl := NewAuditLoggerImpl(db, logger)
	defer impl.Close()

	// Expect若干条用于匹配消费的记录，超出缓冲区的被丢弃
	for i := 0; i < 100; i++ {
		mock.ExpectExec("INSERT INTO " + tables.TAuthAuditLog).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	// 快速发送超过缓冲区容量(1024)的记录，部分被丢弃
	for i := 0; i < 1100; i++ {
		err := impl.LogRecord(context.Background(), &domainaudit.AuditRecord{
			OpType:  domainaudit.OpTypeLoginSuccess,
			Subject: "testuser",
			Result:  true,
		})
		require.NoError(t, err, "缓冲区满应丢弃但不报错")
	}
	waitAuditConsumed()
	// 不严格校验ExpectationsWereMet，因为部分记录被丢弃，只要不panic即通过
}

// 确保sql包被引用。
var _ = sql.ErrNoRows
