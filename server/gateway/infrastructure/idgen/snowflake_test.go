// Package idgen 雪花算法ID生成器实现，infrastructure层技术适配。
package idgen

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSnowflakeIDGen_NextID 测试雪花算法生成全局唯一ID。
func TestSnowflakeIDGen_NextID(t *testing.T) {
	t.Run("单线程生成唯一ID", func(t *testing.T) {
		gen, err := NewSnowflakeIDGen(1)
		require.NoError(t, err)

		ids := make(map[int64]bool)
		for i := 0; i < 10000; i++ {
			id, err := gen.NextID(context.Background())
			require.NoError(t, err)
			assert.False(t, ids[id], "生成重复ID: %d", id)
			ids[id] = true
		}
	})

	t.Run("多线程生成唯一ID", func(t *testing.T) {
		gen, err := NewSnowflakeIDGen(1)
		require.NoError(t, err)

		var wg sync.WaitGroup
		mu := sync.Mutex{}
		ids := make(map[int64]bool)
		duplicateCount := 0

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					id, err := gen.NextID(context.Background())
					require.NoError(t, err)
					mu.Lock()
					if ids[id] {
						duplicateCount++
					}
					ids[id] = true
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, 0, duplicateCount, "多线程生成重复ID")
	})
}

// TestNewSnowflakeIDGen_InvalidWorkerID 测试无效机器ID。
func TestNewSnowflakeIDGen_InvalidWorkerID(t *testing.T) {
	t.Run("负数机器ID", func(t *testing.T) {
		_, err := NewSnowflakeIDGen(-1)
		assert.Error(t, err)
	})

	t.Run("过大机器ID", func(t *testing.T) {
		_, err := NewSnowflakeIDGen(1024)
		assert.Error(t, err)
	})
}
