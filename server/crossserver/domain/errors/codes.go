// Package errors CrossServer服务错误码定义，集中管理便于统一错误处理与排查。
// 错误码区间 18000-18999 为 CrossServer 服务保留。
package errors

import "fmt"

// 错误码常量（规范1），使用显式整型赋值，区间18000-18999。
const (
	// ErrCodeInvalidParams 参数非法，请求参数校验未通过
	ErrCodeInvalidParams = 18001
	// ErrCodeNodeNotFound 节点不存在
	ErrCodeNodeNotFound = 18002
	// ErrCodeNodeOffline 节点离线
	ErrCodeNodeOffline = 18003
	// ErrCodeActivityNotFound 跨服活动不存在
	ErrCodeActivityNotFound = 18004
	// ErrCodeActivityEnded 跨服活动已结束
	ErrCodeActivityEnded = 18005
	// ErrCodeMergeTaskNotFound 合服任务不存在
	ErrCodeMergeTaskNotFound = 18006
	// ErrCodeMergeTaskRunning 合服任务进行中，不可重复执行
	ErrCodeMergeTaskRunning = 18007
)

// CrossServerError CrossServer服务错误，携带错误码便于业务层判断。
type CrossServerError struct {
	Code int    // 错误码，对应ErrCodeXxx常量
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *CrossServerError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// 错误变量定义，业务代码通过errors.Is判断错误类型（规范9）。
var (
	// ErrInvalidParams 参数非法错误
	ErrInvalidParams = &CrossServerError{ErrCodeInvalidParams, "参数非法"}
	// ErrNodeNotFound 节点不存在错误
	ErrNodeNotFound = &CrossServerError{ErrCodeNodeNotFound, "节点不存在"}
	// ErrNodeOffline 节点离线错误
	ErrNodeOffline = &CrossServerError{ErrCodeNodeOffline, "节点离线"}
	// ErrActivityNotFound 跨服活动不存在错误
	ErrActivityNotFound = &CrossServerError{ErrCodeActivityNotFound, "跨服活动不存在"}
	// ErrActivityEnded 跨服活动已结束错误
	ErrActivityEnded = &CrossServerError{ErrCodeActivityEnded, "跨服活动已结束"}
	// ErrMergeTaskNotFound 合服任务不存在错误
	ErrMergeTaskNotFound = &CrossServerError{ErrCodeMergeTaskNotFound, "合服任务不存在"}
	// ErrMergeTaskRunning 合服任务进行中错误
	ErrMergeTaskRunning = &CrossServerError{ErrCodeMergeTaskRunning, "合服任务进行中"}
)
