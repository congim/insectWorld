// Package eventbus 事件总线共享内核，提供领域事件总线契约与Outbox通用接口。
package eventbus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOutboxStatusConstants 测试Outbox投递状态枚举常量值。
func TestOutboxStatusConstants(t *testing.T) {
	assert.Equal(t, 1, OutboxStatusPending)
	assert.Equal(t, 2, OutboxStatusPublished)
	assert.Equal(t, 3, OutboxStatusFailed)
	assert.Equal(t, 4, OutboxStatusProcessing)
}

// TestDomainEventConstruction 测试领域事件结构构造。
func TestDomainEventConstruction(t *testing.T) {
	event := DomainEvent{
		EventID:     "evt-001",
		EventType:   "combat.ended",
		AggregateID: 1001,
		Version:     3,
		Timestamp:   1700000000000,
		Payload:     []byte(`{"winner":1}`),
	}
	assert.Equal(t, "evt-001", event.EventID)
	assert.Equal(t, "combat.ended", event.EventType)
	assert.Equal(t, int64(1001), event.AggregateID)
	assert.Equal(t, 3, event.Version)
	assert.Equal(t, int64(1700000000000), event.Timestamp)
	assert.Equal(t, []byte(`{"winner":1}`), event.Payload)
}

// TestOutboxRecordStatusTransition 测试Outbox记录状态流转。
func TestOutboxRecordStatusTransition(t *testing.T) {
	record := OutboxRecord{
		EventID:     "evt-001",
		AggregateID: 1001,
		EventType:   "combat.ended",
		Payload:     []byte(`{}`),
		Status:      OutboxStatusPending,
		RetryCount:  0,
		CreateTime:  1700000000000,
		PublishTime: 0,
	}
	assert.Equal(t, OutboxStatusPending, record.Status)
	assert.Equal(t, 0, record.RetryCount)
	assert.Equal(t, int64(0), record.PublishTime)

	record.Status = OutboxStatusPublished
	record.PublishTime = 1700000001000
	assert.Equal(t, OutboxStatusPublished, record.Status)
	assert.NotZero(t, record.PublishTime)

	record.Status = OutboxStatusFailed
	record.RetryCount = 3
	assert.Equal(t, OutboxStatusFailed, record.Status)
	assert.Equal(t, 3, record.RetryCount)
}
