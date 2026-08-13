// Package gamepack 通用游戏包模块，负责加载、校验并编译题材内容配置。
package gamepack

import "fmt"

// 游戏包错误码区间 15000-15999，供离线工具和运行时统一识别失败类型。
const (
	ErrCodeManifestInvalid = 15001 // manifest结构、版本或模块声明非法
	ErrCodeFileInvalid     = 15002 // 配置文件路径、格式或schema非法
	ErrCodeIDInvalid       = 15003 // 稳定ID格式非法或重复
	ErrCodeReferenceBroken = 15004 // 配置间引用目标不存在
	ErrCodeEngineMismatch  = 15005 // 游戏包与当前引擎版本不兼容
)

// Error 游戏包错误，携带稳定错误码和中文消息。
type Error struct {
	Code int    // 错误码，对应ErrCodeXxx常量
	Msg  string // 中文错误消息，包含文件和字段定位信息
}

// Error 实现error接口。
func (e *Error) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

var (
	// ErrManifestInvalid manifest内容非法。
	ErrManifestInvalid = &Error{Code: ErrCodeManifestInvalid, Msg: "游戏包manifest非法"}
	// ErrFileInvalid 配置文件内容非法。
	ErrFileInvalid = &Error{Code: ErrCodeFileInvalid, Msg: "游戏包配置文件非法"}
	// ErrIDInvalid 稳定ID非法或重复。
	ErrIDInvalid = &Error{Code: ErrCodeIDInvalid, Msg: "游戏包稳定ID非法"}
	// ErrReferenceBroken 配置引用断裂。
	ErrReferenceBroken = &Error{Code: ErrCodeReferenceBroken, Msg: "游戏包配置引用断裂"}
	// ErrEngineMismatch 引擎版本不在游戏包兼容范围内。
	ErrEngineMismatch = &Error{Code: ErrCodeEngineMismatch, Msg: "游戏包与引擎版本不兼容"}
)
