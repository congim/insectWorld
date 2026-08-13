// Package errors 定义游戏成长上下文稳定错误，供领域与接口层统一映射。
// 错误码区间19000-19999为Game模块保留。
package errors

import "fmt"

// Game模块错误码常量。
const (
	ErrCodeInvalidCommand       = 19001 // 命令参数非法
	ErrCodePlayerNotFound       = 19002 // 玩家不存在
	ErrCodePlayerAlreadyExists  = 19003 // 玩家已存在
	ErrCodeDefinitionNotFound   = 19004 // 游戏包定义不存在
	ErrCodeFactionMismatch      = 19005 // 阵营不匹配
	ErrCodeResourceInsufficient = 19006 // 资源不足
	ErrCodeBuildingNotReady     = 19007 // 建筑不可用
	ErrCodeUnitNotTrainable     = 19008 // 建筑不能训练目标单位
	ErrCodeTaskNotReady         = 19009 // 定时任务尚未到期
	ErrCodeStateConflict        = 19010 // 状态或幂等载荷冲突
)

// GameError 是携带稳定错误码和中文客户端文案的业务错误。
type GameError struct {
	Code int32  // 稳定协议错误码，范围19000-19999
	Msg  string // 中文客户端错误文案
}

// Error 实现error接口。
func (e *GameError) Error() string { return fmt.Sprintf("[%d] %s", e.Code, e.Msg) }

// 稳定业务错误变量，调用方使用errors.Is判断后映射协议状态。
var (
	ErrInvalidCommand       = &GameError{Code: ErrCodeInvalidCommand, Msg: "命令参数非法"}
	ErrPlayerNotFound       = &GameError{Code: ErrCodePlayerNotFound, Msg: "玩家不存在"}
	ErrPlayerAlreadyExists  = &GameError{Code: ErrCodePlayerAlreadyExists, Msg: "玩家已存在"}
	ErrDefinitionNotFound   = &GameError{Code: ErrCodeDefinitionNotFound, Msg: "游戏内容定义不存在"}
	ErrFactionMismatch      = &GameError{Code: ErrCodeFactionMismatch, Msg: "阵营不匹配"}
	ErrResourceInsufficient = &GameError{Code: ErrCodeResourceInsufficient, Msg: "资源不足"}
	ErrBuildingNotReady     = &GameError{Code: ErrCodeBuildingNotReady, Msg: "建筑不可用"}
	ErrUnitNotTrainable     = &GameError{Code: ErrCodeUnitNotTrainable, Msg: "建筑不能训练目标单位"}
	ErrTaskNotReady         = &GameError{Code: ErrCodeTaskNotReady, Msg: "任务尚未到完成时间"}
	ErrStateConflict        = &GameError{Code: ErrCodeStateConflict, Msg: "状态冲突"}
)
