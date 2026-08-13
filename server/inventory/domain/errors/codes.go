// Package errors Inventory服务错误码定义，集中管理便于统一错误处理与排查。
// 错误码区间 15000-15999 为 Inventory 服务保留。
package errors

import "fmt"

// 错误码常量（规范1），使用显式整型赋值，区间15000-15999。
const (
	// ErrCodeInvalidParams 参数非法，请求参数校验未通过
	ErrCodeInvalidParams = 15001
	// ErrCodeItemNotFound 道具不存在，查询的道具在背包中不存在
	ErrCodeItemNotFound = 15002
	// ErrCodeInventoryFull 背包已满，新增道具时背包容量不足
	ErrCodeInventoryFull = 15003
	// ErrCodeItemNotUsable 道具不可使用，道具使用条件不满足
	ErrCodeItemNotUsable = 15004
	// ErrCodeItemInsufficient 道具数量不足，使用/出售时数量不够
	ErrCodeItemInsufficient = 15005
	// ErrCodeItemExpired 道具已过期，过期道具不可使用
	ErrCodeItemExpired = 15006
	// ErrCodeStackLimitExceeded 堆叠上限超出，可堆叠道具超过最大堆叠数
	ErrCodeStackLimitExceeded = 15007
)

// InventoryError Inventory服务错误，携带错误码便于业务层判断。
type InventoryError struct {
	Code int    // 错误码，对应ErrCodeXxx常量
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *InventoryError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// 错误变量定义，业务代码通过errors.Is判断错误类型（规范9）。
var (
	// ErrInvalidParams 参数非法错误
	ErrInvalidParams = &InventoryError{ErrCodeInvalidParams, "参数非法"}
	// ErrItemNotFound 道具不存在错误
	ErrItemNotFound = &InventoryError{ErrCodeItemNotFound, "道具不存在"}
	// ErrInventoryFull 背包已满错误
	ErrInventoryFull = &InventoryError{ErrCodeInventoryFull, "背包已满"}
	// ErrItemNotUsable 道具不可使用错误
	ErrItemNotUsable = &InventoryError{ErrCodeItemNotUsable, "道具不可使用"}
	// ErrItemInsufficient 道具数量不足错误
	ErrItemInsufficient = &InventoryError{ErrCodeItemInsufficient, "道具数量不足"}
	// ErrItemExpired 道具已过期错误
	ErrItemExpired = &InventoryError{ErrCodeItemExpired, "道具已过期"}
	// ErrStackLimitExceeded 堆叠上限超出错误
	ErrStackLimitExceeded = &InventoryError{ErrCodeStackLimitExceeded, "堆叠上限超出"}
)
