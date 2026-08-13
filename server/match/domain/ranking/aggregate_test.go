// Package ranking 排行榜聚合根单元测试。
package ranking

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRanking 测试排行榜创建。
func TestNewRanking(t *testing.T) {
	r := NewRanking(1, RankTypePower, 100, 1, 100)
	assert.Equal(t, int64(1), r.RankID())
	assert.Equal(t, RankTypePower, r.RankType())
	assert.Equal(t, 0, len(r.Entries()))
}

// TestRanking_UpdateScore 测试排名更新。
func TestRanking_UpdateScore(t *testing.T) {
	r := NewRanking(1, RankTypePower, 100, 1, 10)

	_, err := r.UpdateScore(1001, 5000, 1000)
	require.NoError(t, err)
	_, err = r.UpdateScore(1002, 3000, 2000)
	require.NoError(t, err)

	entries := r.Entries()
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, int64(1001), entries[0].SubjectID)
	assert.Equal(t, 1, entries[0].Rank)
	assert.Equal(t, int64(1002), entries[1].SubjectID)
	assert.Equal(t, 2, entries[1].Rank)
}

// TestRanking_UpdateScore_Existing 测试更新已有主体分数。
func TestRanking_UpdateScore_Existing(t *testing.T) {
	r := NewRanking(1, RankTypePower, 100, 1, 10)

	_, _ = r.UpdateScore(1001, 5000, 1000)
	_, _ = r.UpdateScore(1002, 3000, 2000)
	event, err := r.UpdateScore(1001, 6000, 3000)
	require.NoError(t, err)
	assert.Equal(t, int64(6000), event.Score)
	assert.Equal(t, 1, event.Rank)
}

// TestRanking_UpdateScore_TopN 测试TopN截断。
func TestRanking_UpdateScore_TopN(t *testing.T) {
	r := NewRanking(1, RankTypePower, 100, 1, 2)

	_, _ = r.UpdateScore(1001, 5000, 1000)
	_, _ = r.UpdateScore(1002, 4000, 2000)
	_, _ = r.UpdateScore(1003, 3000, 3000)

	entries := r.Entries()
	assert.Equal(t, 2, len(entries))
}

// TestRanking_Archive 测试排行榜归档。
func TestRanking_Archive(t *testing.T) {
	r := NewRanking(1, RankTypePower, 100, 1, 10)
	r.Archive()

	_, err := r.UpdateScore(1001, 5000, 1000)
	assert.Error(t, err)
}
