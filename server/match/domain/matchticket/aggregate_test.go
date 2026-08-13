// Package matchticket 匹配票聚合根单元测试。
package matchticket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMatchTicket 测试匹配票创建。
func TestNewMatchTicket(t *testing.T) {
	ticket := NewMatchTicket(1, 100, SubjectPlayer, 1000, 5, 3000, 1000)
	assert.Equal(t, int64(1), ticket.TicketID())
	assert.Equal(t, StatusWaiting, ticket.Status())
}

// TestMatchTicket_Match 测试匹配成功。
func TestMatchTicket_Match(t *testing.T) {
	ticket := NewMatchTicket(1, 100, SubjectPlayer, 1000, 5, 3000, 1000)
	event, err := ticket.Match(5001, 2000)
	require.NoError(t, err)
	assert.Equal(t, StatusMatched, ticket.Status())
	assert.Equal(t, int64(5001), event.BattlefieldID)
	assert.Equal(t, int64(2000), event.MatchTime)
}

// TestMatchTicket_Match_NotWaiting 测试非等待状态匹配失败。
func TestMatchTicket_Match_NotWaiting(t *testing.T) {
	ticket := NewMatchTicket(1, 100, SubjectPlayer, 1000, 5, 3000, 1000)
	_, _ = ticket.Match(5001, 2000)

	_, err := ticket.Match(5002, 3000)
	assert.Error(t, err)
}

// TestMatchTicket_Timeout 测试匹配超时。
func TestMatchTicket_Timeout(t *testing.T) {
	ticket := NewMatchTicket(1, 100, SubjectPlayer, 1000, 5, 3000, 1000)
	event := ticket.Timeout(5000)
	assert.Equal(t, StatusTimeout, ticket.Status())
	assert.Equal(t, int64(4000), event.WaitDuration)
}

// TestMatchTicket_Cancel 测试取消匹配。
func TestMatchTicket_Cancel(t *testing.T) {
	ticket := NewMatchTicket(1, 100, SubjectPlayer, 1000, 5, 3000, 1000)
	err := ticket.Cancel()
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, ticket.Status())
}

// TestMatchTicket_Cancel_NotWaiting 测试非等待状态取消失败。
func TestMatchTicket_Cancel_NotWaiting(t *testing.T) {
	ticket := NewMatchTicket(1, 100, SubjectPlayer, 1000, 5, 3000, 1000)
	_, _ = ticket.Match(5001, 2000)

	err := ticket.Cancel()
	assert.Error(t, err)
}
