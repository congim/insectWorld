// Package e2e 测试端端到端测试执行器，编排注册→登录→心跳→鉴权→登出五步认证流程并执行断言。
package e2e

import (
	"fmt"

	"go.uber.org/zap"

	"insectworld/tests/authtest/internal/contract"
)

// Assertion 响应断言器。
type Assertion struct {
	logger *zap.Logger // 结构化日志
}

// NewAssertion 创建断言器实例。
func NewAssertion(logger *zap.Logger) *Assertion {
	return &Assertion{logger: logger}
}

// AssertRegister 断言注册响应。
func (a *Assertion) AssertRegister(resp contract.AuthResponse) (bool, string) {
	if resp.Success && resp.PlayerID > 0 {
		return true, fmt.Sprintf("注册成功，返回玩家ID:%d", resp.PlayerID)
	}
	return false, fmt.Sprintf("注册失败，错误码:%d，错误消息:%s", resp.ErrorCode, resp.ErrorMsg)
}

// AssertLogin 断言登录响应。
func (a *Assertion) AssertLogin(resp contract.AuthResponse, expectedPlayerID int64) (bool, string) {
	if resp.Success && resp.Token != "" && resp.PlayerID == expectedPlayerID {
		return true, fmt.Sprintf("登录成功，玩家ID:%d，令牌已签发", resp.PlayerID)
	}
	return false, fmt.Sprintf("登录失败，错误码:%d，错误消息:%s", resp.ErrorCode, resp.ErrorMsg)
}

// AssertHeartbeat 断言心跳响应。
func (a *Assertion) AssertHeartbeat(resp contract.AuthResponse) (bool, string) {
	if resp.Success {
		return true, "心跳更新成功"
	}
	return false, fmt.Sprintf("心跳失败，错误码:%d，错误消息:%s", resp.ErrorCode, resp.ErrorMsg)
}

// AssertAuthenticate 断言鉴权响应。
func (a *Assertion) AssertAuthenticate(resp contract.AuthResponse, expectedPlayerID int64) (bool, string) {
	if resp.Success && resp.PlayerID == expectedPlayerID {
		return true, fmt.Sprintf("鉴权通过，玩家ID:%d", resp.PlayerID)
	}
	return false, fmt.Sprintf("鉴权失败，错误码:%d，错误消息:%s", resp.ErrorCode, resp.ErrorMsg)
}

// AssertLogout 断言登出响应。
func (a *Assertion) AssertLogout(resp contract.AuthResponse) (bool, string) {
	if resp.Success {
		return true, "登出成功"
	}
	return false, fmt.Sprintf("登出失败，错误码:%d，错误消息:%s", resp.ErrorCode, resp.ErrorMsg)
}

// AssertFailure 断言失败场景响应。
func (a *Assertion) AssertFailure(resp contract.AuthResponse, expectedCode int) (bool, string) {
	if !resp.Success && resp.ErrorCode == expectedCode {
		return true, fmt.Sprintf("符合预期失败，错误码:%d", resp.ErrorCode)
	}
	return false, fmt.Sprintf("不符合预期，期望错误码:%d，实际错误码:%d", expectedCode, resp.ErrorCode)
}
