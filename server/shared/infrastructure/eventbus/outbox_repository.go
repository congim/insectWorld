// Package eventbus 提供共享事件契约的MySQL基础设施适配。
package eventbus

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"insectworld/server/shared/pkg/eventbus"
	"insectworld/server/shared/schema/tables"
)

// OutboxRepository 是支持多实例租约领取的MySQL Outbox仓储。
type OutboxRepository struct {
	db *sql.DB // MySQL连接池
}

// NewOutboxRepository 创建MySQL Outbox仓储。
func NewOutboxRepository(db *sql.DB) *OutboxRepository { return &OutboxRepository{db: db} }

// Save 保存一条待投递Outbox记录。
func (r *OutboxRepository) Save(ctx context.Context, record eventbus.OutboxRecord) error {
	query := fmt.Sprintf(`INSERT INTO %s (event_id, aggregate_id, event_type, event_version, payload, status, retry_count, create_time, publish_time, available_time, last_error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '')`, tables.TOutbox)
	_, err := r.db.ExecContext(ctx, query, record.EventID, record.AggregateID, record.EventType, record.Version, record.Payload, record.Status, record.RetryCount, record.CreateTime, record.PublishTime, record.AvailableTime)
	if err != nil {
		return fmt.Errorf("保存Outbox记录失败，eventID=%s: %w", record.EventID, err)
	}
	return nil
}

// MarkPublished 标记事件已成功交付给全部同步订阅者。
func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID string, publishTime int64) error {
	query := fmt.Sprintf(`UPDATE %s SET status = ?, publish_time = ?, available_time = 0, last_error = '' WHERE event_id = ? AND status = ?`, tables.TOutbox)
	result, err := r.db.ExecContext(ctx, query, eventbus.OutboxStatusPublished, publishTime, eventID, eventbus.OutboxStatusProcessing)
	if err != nil {
		return fmt.Errorf("标记Outbox已发布失败，eventID=%s: %w", eventID, err)
	}
	return requireOneAffected(result, eventID, "标记已发布")
}

// MarkFailed 标记投递失败并设置下一次可领取时间。
func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID string, nextAttemptMs int64, failure string) error {
	query := fmt.Sprintf(`UPDATE %s SET status = ?, retry_count = retry_count + 1, available_time = ?, last_error = ? WHERE event_id = ? AND status = ?`, tables.TOutbox)
	result, err := r.db.ExecContext(ctx, query, eventbus.OutboxStatusFailed, nextAttemptMs, failure, eventID, eventbus.OutboxStatusProcessing)
	if err != nil {
		return fmt.Errorf("标记Outbox失败状态异常，eventID=%s: %w", eventID, err)
	}
	return requireOneAffected(result, eventID, "标记失败")
}

// GetPending 查询当前待投递记录，不领取租约，仅用于管理查询和兼容旧调用方。
func (r *OutboxRepository) GetPending(ctx context.Context, limit int) ([]eventbus.OutboxRecord, error) {
	query := fmt.Sprintf(`SELECT event_id, aggregate_id, event_type, event_version, payload, status, retry_count, create_time, publish_time, available_time FROM %s WHERE status IN (?, ?) ORDER BY create_time, id LIMIT ?`, tables.TOutbox)
	rows, err := r.db.QueryContext(ctx, query, eventbus.OutboxStatusPending, eventbus.OutboxStatusFailed, limit)
	if err != nil {
		return nil, fmt.Errorf("查询待投递Outbox失败: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

// ClaimPending 原子领取指定类型的到期事件，并把租约截止时间写入available_time。
func (r *OutboxRepository) ClaimPending(ctx context.Context, nowMs int64, eventTypes []string, limit int, leaseUntilMs int64) ([]eventbus.OutboxRecord, error) {
	if len(eventTypes) == 0 {
		return nil, fmt.Errorf("领取Outbox事件类型不能为空")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开启Outbox领取事务失败: %w", err)
	}
	defer tx.Rollback()
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(eventTypes)), ",")
	query := fmt.Sprintf(`SELECT event_id, aggregate_id, event_type, event_version, payload, status, retry_count, create_time, publish_time, available_time FROM %s WHERE event_type IN (%s) AND ((status IN (?, ?) AND available_time <= ?) OR (status = ? AND available_time <= ?)) ORDER BY create_time, id LIMIT ? FOR UPDATE SKIP LOCKED`, tables.TOutbox, placeholders)
	args := make([]any, 0, len(eventTypes)+6)
	for _, eventType := range eventTypes {
		if eventType == "" {
			return nil, fmt.Errorf("领取Outbox事件类型不能包含空值")
		}
		args = append(args, eventType)
	}
	args = append(args, eventbus.OutboxStatusPending, eventbus.OutboxStatusFailed, nowMs, eventbus.OutboxStatusProcessing, nowMs, limit)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("锁定待投递Outbox失败: %w", err)
	}
	records, err := scanRecords(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	update := fmt.Sprintf(`UPDATE %s SET status = ?, available_time = ? WHERE event_id = ?`, tables.TOutbox)
	for index := range records {
		if _, err := tx.ExecContext(ctx, update, eventbus.OutboxStatusProcessing, leaseUntilMs, records[index].EventID); err != nil {
			return nil, fmt.Errorf("设置Outbox投递租约失败，eventID=%s: %w", records[index].EventID, err)
		}
		records[index].Status = eventbus.OutboxStatusProcessing
		records[index].AvailableTime = leaseUntilMs
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交Outbox领取事务失败: %w", err)
	}
	return records, nil
}

type rowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanRecords(rows rowScanner) ([]eventbus.OutboxRecord, error) {
	var records []eventbus.OutboxRecord
	for rows.Next() {
		var record eventbus.OutboxRecord
		if err := rows.Scan(&record.EventID, &record.AggregateID, &record.EventType, &record.Version, &record.Payload, &record.Status, &record.RetryCount, &record.CreateTime, &record.PublishTime, &record.AvailableTime); err != nil {
			return nil, fmt.Errorf("扫描Outbox记录失败: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历Outbox记录失败: %w", err)
	}
	return records, nil
}

func requireOneAffected(result sql.Result, eventID string, operation string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取Outbox更新结果失败，eventID=%s: %w", eventID, err)
	}
	if affected != 1 {
		return fmt.Errorf("Outbox状态已被其他实例修改，operation=%s，eventID=%s", operation, eventID)
	}
	return nil
}

// 确保OutboxRepository实现共享Outbox契约。
var _ eventbus.OutboxRepository = (*OutboxRepository)(nil)
