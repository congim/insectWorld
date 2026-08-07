// Package audit 审计日志infrastructure层实现，提供MySQL异步落库适配。
//
// AuditLoggerImpl异步INSERT到t_auth_audit_log表，主流程不阻塞（design.md 2.2.2.7节）。
// MySQL故障时记录Error日志但不返回错误（审计日志失败不阻塞主流程）。
package audit

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	domainaudit "insectworld/server/gateway/domain/audit"
	"insectworld/server/shared/schema/tables"
)

// 审计日志异步落库缓冲区大小，超过时丢弃并告警。
const auditLogBufferSize = 1024

// AuditLoggerImpl MySQL审计日志实现，实现AuditLogger接口。
type AuditLoggerImpl struct {
	db     *sql.DB                       // MySQL数据库连接
	logger *zap.Logger                   // 结构化日志
	ch     chan *domainaudit.AuditRecord // 异步落库缓冲channel
	wg     sync.WaitGroup                // 等待goroutine退出
	stopCh chan struct{}                 // 停止信号channel
}

// NewAuditLoggerImpl 创建MySQL审计日志实例。
//
// 启动后台goroutine异步消费channel写入MySQL。
func NewAuditLoggerImpl(db *sql.DB, logger *zap.Logger) *AuditLoggerImpl {
	impl := &AuditLoggerImpl{
		db:     db,
		logger: logger,
		ch:     make(chan *domainaudit.AuditRecord, auditLogBufferSize),
		stopCh: make(chan struct{}),
	}
	impl.wg.Add(1)
	go impl.consumeLoop()
	return impl
}

// LogRecord 异步记录审计日志到t_auth_audit_log表。
//
// 非阻塞写入channel，由后台goroutine消费落库。
// channel满时丢弃并记录Warn日志，MySQL故障不阻塞主流程。
func (l *AuditLoggerImpl) LogRecord(ctx context.Context, record *domainaudit.AuditRecord) error {
	select {
	case l.ch <- record:
		return nil
	default:
		l.logger.Warn("审计日志缓冲区满，丢弃记录",
			zap.Int("op_type", record.OpType),
			zap.String("subject", record.Subject),
		)
		return nil
	}
}

// consumeLoop 后台消费循环，从channel读取记录写入MySQL。
func (l *AuditLoggerImpl) consumeLoop() {
	defer l.wg.Done()
	for {
		select {
		case record := <-l.ch:
			l.writeToDB(record)
		case <-l.stopCh:
			return
		}
	}
}

// writeToDB 将审计记录写入MySQL，失败时记录Error日志不阻塞。
func (l *AuditLoggerImpl) writeToDB(record *domainaudit.AuditRecord) {
	result := 0
	if record.Result {
		result = 1
	}
	query := fmt.Sprintf(
		`INSERT INTO %s (op_type, subject, result, source_ip, op_time, extra) VALUES (?, ?, ?, ?, ?, ?)`,
		tables.TAuthAuditLog,
	)
	_, err := l.db.Exec(query, record.OpType, record.Subject, result, record.SourceIP, record.OpTime, record.Extra)
	if err != nil {
		l.logger.Error("审计日志落库失败",
			zap.Int("op_type", record.OpType),
			zap.String("subject", record.Subject),
			zap.Error(err),
		)
	}
}

// Close 关闭审计日志，停止后台goroutine，等待缓冲区落库完成。
//
// 优雅退出：发送停止信号，等待goroutine退出，超时5秒强制返回。
func (l *AuditLoggerImpl) Close() {
	close(l.stopCh)
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		l.logger.Warn("审计日志关闭超时，强制返回")
	}
}
