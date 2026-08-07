// Package errors World服务错误码定义，集中管理便于统一错误处理与排查。
// 错误码区间 10000-10999 为 World 服务保留。
package errors

import "fmt"

// 错误码常量（规范1），使用显式整型赋值，区间10000-10999。
const (
	// ErrCodeInvalidParams 参数非法，请求参数校验未通过
	ErrCodeInvalidParams = 10001
	// ErrCodeRuleViolation 规则违反，业务规则校验未通过
	ErrCodeRuleViolation = 10002
	// ErrCodeCooldownActive 冷却中，操作冷却未过期
	ErrCodeCooldownActive = 10003
	// ErrCodeResourceInsufficient 资源不足，操作所需资源不够
	ErrCodeResourceInsufficient = 10004
	// ErrCodeRegionNotEmpty 区域非空，销毁区域时存在活跃实体
	ErrCodeRegionNotEmpty = 10005
	// ErrCodeOutOfBounds 坐标越界，坐标超出地图范围
	ErrCodeOutOfBounds = 10006
	// ErrCodeEntityNotFound 实体不存在
	ErrCodeEntityNotFound = 10007
	// ErrCodeMovementInProgress 已有移动进行中
	ErrCodeMovementInProgress = 10008
)

// WorldError World服务错误，携带错误码便于业务层判断。
type WorldError struct {
	Code int    // 错误码，对应ErrCodeXxx常量
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *WorldError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// 错误变量定义，业务代码通过errors.Is判断错误类型（规范9）。
var (
	// ErrInvalidParams 参数非法错误
	ErrInvalidParams = &WorldError{ErrCodeInvalidParams, "参数非法"}
	// ErrRuleViolation 规则违反错误
	ErrRuleViolation = &WorldError{ErrCodeRuleViolation, "规则违反"}
	// ErrCooldownActive 冷却中错误
	ErrCooldownActive = &WorldError{ErrCodeCooldownActive, "冷却中"}
	// ErrResourceInsufficient 资源不足错误
	ErrResourceInsufficient = &WorldError{ErrCodeResourceInsufficient, "资源不足"}
	// ErrRegionNotEmpty 区域非空错误
	ErrRegionNotEmpty = &WorldError{ErrCodeRegionNotEmpty, "区域非空"}
	// ErrOutOfBounds 坐标越界错误
	ErrOutOfBounds = &WorldError{ErrCodeOutOfBounds, "坐标越界"}
	// ErrEntityNotFound 实体不存在错误
	ErrEntityNotFound = &WorldError{ErrCodeEntityNotFound, "实体不存在"}
	// ErrMovementInProgress 已有移动进行中错误
	ErrMovementInProgress = &WorldError{ErrCodeMovementInProgress, "已有移动进行中"}
)
