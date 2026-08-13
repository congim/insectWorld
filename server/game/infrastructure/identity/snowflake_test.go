package identity

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSnowflakeGeneratesUniqueIDs 验证固定时钟和并发调用仍不会生成重复ID。
func TestSnowflakeGeneratesUniqueIDs(t *testing.T) {
	t.Parallel()
	generator, err := NewSnowflake(2)
	require.NoError(t, err)
	generator.now = func() int64 { return epochMs + 1 }
	ids := make(chan int64, 5000)
	var waitGroup sync.WaitGroup
	for range 10 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for range 500 {
				ids <- generator.Next()
			}
		}()
	}
	waitGroup.Wait()
	close(ids)
	seen := make(map[int64]struct{}, 5000)
	for id := range ids {
		assert.Positive(t, id)
		_, exists := seen[id]
		assert.False(t, exists, "ID不应重复")
		seen[id] = struct{}{}
	}
}

// TestNewSnowflakeRejectsInvalidWorker 验证部署节点编号范围。
func TestNewSnowflakeRejectsInvalidWorker(t *testing.T) {
	t.Parallel()
	_, err := NewSnowflake(workerIDMax + 1)
	require.Error(t, err)
}
