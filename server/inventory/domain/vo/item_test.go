// Package vo Inventory服务值对象单元测试。
package vo

import "testing"

// TestItem_IsExpired 测试道具过期判定。
func TestItem_IsExpired(t *testing.T) {
	tests := []struct {
		name        string
		item        Item
		now         int64
		wantExpired bool
	}{
		{"永不过期", Item{ExpireTime: 0}, 9999, false},
		{"未过期", Item{ExpireTime: 2000}, 1000, false},
		{"已过期", Item{ExpireTime: 1000}, 2000, true},
		{"刚好过期", Item{ExpireTime: 1000}, 1000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.IsExpired(tt.now)
			if got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}
