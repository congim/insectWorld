// Package idgen 雪花算法ID生成器实现，infrastructure层技术适配。
//
// 实现domain层IDGenerator接口，生成全局唯一int64 ID（规范8）。
// 雪花算法保证全局唯一（spec 5.1.1 规则4），时钟回拨时返回错误（spec 4.2 可靠性）。
package idgen

import (
	"context"
	"fmt"
	"sync"
	"time"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// 雪花算法位分配常量（规范1就近归属）。
const (
	workerIDBits       = 10                          // 机器ID占用位数，最多1024个节点
	sequenceBits       = 12                          // 序列号占用位数，每毫秒最多4096个ID
	workerIDMax        = -1 ^ (-1 << workerIDBits)   // 机器ID最大值
	sequenceMax        = -1 ^ (-1 << sequenceBits)   // 序列号最大值
	timestampLeftShift = workerIDBits + sequenceBits // 时间戳左移位数
	workerIDLeftShift  = sequenceBits                // 机器ID左移位数
	// twepoch 雪花算法起始纪元，2024-01-01 00:00:00 UTC的毫秒时间戳
	twepoch = 1704067200000
)

// SnowflakeIDGen 雪花算法ID生成器，实现IDGenerator接口。
type SnowflakeIDGen struct {
	workerID      int64      // 机器ID，多实例区分
	sequence      int64      // 当前毫秒内序列号
	lastTimestamp int64      // 上次生成ID的时间戳
	mu            sync.Mutex // 互斥锁，保证并发安全
}

// NewSnowflakeIDGen 创建雪花ID生成器实例。
//
// workerID为机器ID，多实例区分，范围[0, 1023]。
func NewSnowflakeIDGen(workerID int64) (*SnowflakeIDGen, error) {
	if workerID < 0 || workerID > workerIDMax {
		return nil, fmt.Errorf("机器ID超出范围[0,%d]: workerID=%d", workerIDMax, workerID)
	}
	return &SnowflakeIDGen{
		workerID:      workerID,
		lastTimestamp: -1,
	}, nil
}

// NextID 生成全局唯一int64 ID，实现IDGenerator接口。
//
// 时钟回拨时返回ErrIDGenClockBack（spec 4.2 可靠性）。
func (s *SnowflakeIDGen) NextID(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	timestamp := time.Now().UnixMilli() - twepoch
	if timestamp < s.lastTimestamp {
		return 0, fmt.Errorf("时钟回拨，当前=%d 上次=%d: %w", timestamp, s.lastTimestamp, gatewayerr.ErrIDGenClockBack)
	}

	if timestamp == s.lastTimestamp {
		s.sequence = (s.sequence + 1) & sequenceMax
		if s.sequence == 0 {
			timestamp = s.tilNextMillis(s.lastTimestamp)
		}
	} else {
		s.sequence = 0
	}

	s.lastTimestamp = timestamp
	id := (timestamp << timestampLeftShift) | (s.workerID << workerIDLeftShift) | s.sequence
	return id, nil
}

// tilNextMillis 等待到下一毫秒。
func (s *SnowflakeIDGen) tilNextMillis(lastTimestamp int64) int64 {
	timestamp := time.Now().UnixMilli() - twepoch
	for timestamp <= lastTimestamp {
		timestamp = time.Now().UnixMilli() - twepoch
	}
	return timestamp
}
