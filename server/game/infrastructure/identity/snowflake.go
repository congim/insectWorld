// Package identity 提供Growth聚合实例ID的基础设施实现。
package identity

import (
	"fmt"
	"sync"
	"time"
)

const (
	workerIDBits       = 10                          // 机器ID位数，最多支持1024个并发节点
	sequenceBits       = 12                          // 毫秒内序列位数，每节点每毫秒最多4096个ID
	workerIDMax        = int64(1<<workerIDBits - 1)  // 机器ID最大值
	sequenceMask       = int64(1<<sequenceBits - 1)  // 毫秒内序列掩码
	workerIDLeftShift  = sequenceBits                // 机器ID左移位数
	timestampLeftShift = workerIDBits + sequenceBits // 时间戳左移位数
	epochMs            = int64(1704067200000)        // 自定义纪元：2024-01-01 00:00:00 UTC
)

// Snowflake 是进程内并发安全的全局ID生成器。
// 时钟短暂回拨和单毫秒序列耗尽时使用逻辑毫秒继续前进，避免重复ID。
type Snowflake struct {
	workerID      int64        // 部署节点唯一编号，范围0到1023
	sequence      int64        // 当前逻辑毫秒内序列号
	lastTimestamp int64        // 上次使用的逻辑时间戳，单位毫秒
	mu            sync.Mutex   // 保护时间戳和序列号
	now           func() int64 // 当前Unix毫秒时钟，测试可替换
}

// NewSnowflake 创建指定节点的ID生成器。
func NewSnowflake(workerID int64) (*Snowflake, error) {
	if workerID < 0 || workerID > workerIDMax {
		return nil, fmt.Errorf("机器ID超出范围[0,%d]，workerID=%d", workerIDMax, workerID)
	}
	return &Snowflake{workerID: workerID, lastTimestamp: -1, now: func() int64 { return time.Now().UnixMilli() }}, nil
}

// Next 返回下一个全局ID。
func (s *Snowflake) Next() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	timestamp := s.now() - epochMs
	if timestamp < s.lastTimestamp {
		timestamp = s.lastTimestamp
	}
	if timestamp == s.lastTimestamp {
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			timestamp = s.lastTimestamp + 1
		}
	} else {
		s.sequence = 0
	}
	s.lastTimestamp = timestamp
	return timestamp<<timestampLeftShift | s.workerID<<workerIDLeftShift | s.sequence
}
