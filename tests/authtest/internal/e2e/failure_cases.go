// Package e2e 测试端端到端测试执行器，编排注册→登录→心跳→鉴权→登出五步认证流程并执行断言。
package e2e

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/tests/authtest/internal/contract"
	"insectworld/tests/authtest/internal/wsclient"
)

// FailureCase 失败场景用例。
type FailureCase struct {
	CaseName string                                                                // 用例名称
	Run      func(ctx context.Context, wsClient *wsclient.AuthWSClient) StepResult // 执行函数
}

// FailureCaseRunner 失败场景执行器。
type FailureCaseRunner struct {
	wsClient  *wsclient.AuthWSClient // WebSocket客户端
	assertion *Assertion             // 断言器
	logger    *zap.Logger            // 结构化日志
}

// NewFailureCaseRunner 创建失败场景执行器实例。
func NewFailureCaseRunner(wsClient *wsclient.AuthWSClient, assertion *Assertion, logger *zap.Logger) *FailureCaseRunner {
	return &FailureCaseRunner{
		wsClient:  wsClient,
		assertion: assertion,
		logger:    logger,
	}
}

// BuiltInCases 返回预置失败用例列表。
func (r *FailureCaseRunner) BuiltInCases() []FailureCase {
	return []FailureCase{
		{
			CaseName: "重复注册",
			Run: func(ctx context.Context, wsClient *wsclient.AuthWSClient) StepResult {
				username := fmt.Sprintf("dup_%d", time.Now().UnixMilli())
				msg1 := contract.AuthMessage{Type: contract.MsgTypeRegister, Username: username, Password: "Test12345"}
				_, _ = wsClient.Send(ctx, msg1)
				msg2 := contract.AuthMessage{Type: contract.MsgTypeRegister, Username: username, Password: "Test12345"}
				result, err := wsClient.Send(ctx, msg2)
				if err != nil {
					return StepResult{StepName: "重复注册", Assertion: err.Error(), Status: StepFailed}
				}
				passed, desc := r.assertion.AssertFailure(result.Response, contract.ErrCodeUsernameAlreadyExists)
				return buildFailureStep("重复注册", result, passed, desc)
			},
		},
		{
			CaseName: "登录错误密码",
			Run: func(ctx context.Context, wsClient *wsclient.AuthWSClient) StepResult {
				username := fmt.Sprintf("wp_%d", time.Now().UnixMilli())
				regMsg := contract.AuthMessage{Type: contract.MsgTypeRegister, Username: username, Password: "Correct123"}
				_, _ = wsClient.Send(ctx, regMsg)
				loginMsg := contract.AuthMessage{Type: contract.MsgTypeLogin, Username: username, Password: "WrongPassword1"}
				result, err := wsClient.Send(ctx, loginMsg)
				if err != nil {
					return StepResult{StepName: "登录错误密码", Assertion: err.Error(), Status: StepFailed}
				}
				passed, desc := r.assertion.AssertFailure(result.Response, contract.ErrCodePasswordIncorrect)
				return buildFailureStep("登录错误密码", result, passed, desc)
			},
		},
		{
			CaseName: "登录不存在账号",
			Run: func(ctx context.Context, wsClient *wsclient.AuthWSClient) StepResult {
				username := fmt.Sprintf("ne_%d", time.Now().UnixMilli())
				loginMsg := contract.AuthMessage{Type: contract.MsgTypeLogin, Username: username, Password: "Test12345"}
				result, err := wsClient.Send(ctx, loginMsg)
				if err != nil {
					return StepResult{StepName: "登录不存在账号", Assertion: err.Error(), Status: StepFailed}
				}
				passed, desc := r.assertion.AssertFailure(result.Response, contract.ErrCodeAccountNotFound)
				return buildFailureStep("登录不存在账号", result, passed, desc)
			},
		},
		{
			CaseName: "心跳无效令牌",
			Run: func(ctx context.Context, wsClient *wsclient.AuthWSClient) StepResult {
				msg := contract.AuthMessage{Type: contract.MsgTypeHeartbeat, Token: "invalid_token", PlayerID: 99999}
				result, err := wsClient.Send(ctx, msg)
				if err != nil {
					return StepResult{StepName: "心跳无效令牌", Assertion: err.Error(), Status: StepFailed}
				}
				passed := !result.Response.Success
				desc := "心跳被拒绝"
				if !passed {
					desc = "心跳未被拒绝，预期失败"
				}
				return buildFailureStep("心跳无效令牌", result, passed, desc)
			},
		},
		{
			CaseName: "登出无效令牌",
			Run: func(ctx context.Context, wsClient *wsclient.AuthWSClient) StepResult {
				msg := contract.AuthMessage{Type: contract.MsgTypeLogout, Token: "invalid_token", PlayerID: 99999}
				result, err := wsClient.Send(ctx, msg)
				if err != nil {
					return StepResult{StepName: "登出无效令牌", Assertion: err.Error(), Status: StepFailed}
				}
				passed := result.Response.Success
				desc := "登出幂等返回成功"
				if !passed {
					desc = fmt.Sprintf("登出未幂等返回，错误码:%d", result.Response.ErrorCode)
				}
				return buildFailureStep("登出无效令牌", result, passed, desc)
			},
		},
	}
}

// RunAll 执行全部预置失败用例。
func (r *FailureCaseRunner) RunAll(ctx context.Context) []StepResult {
	cases := r.BuiltInCases()
	results := make([]StepResult, 0, len(cases))
	for _, c := range cases {
		result := c.Run(ctx, r.wsClient)
		results = append(results, result)
		r.logger.Info("失败用例执行完成",
			zap.String("case", c.CaseName),
			zap.Int("status", result.Status),
		)
	}
	return results
}

// buildFailureStep 构建失败用例步骤结果。
func buildFailureStep(name string, result *wsclient.RequestResult, passed bool, desc string) StepResult {
	status := StepPassed
	if !passed {
		status = StepFailed
	}
	return StepResult{
		StepName:   name,
		Response:   fmt.Sprintf(`{"success":%v,"error_code":%d,"error_msg":"%s"}`, result.Response.Success, result.Response.ErrorCode, result.Response.ErrorMsg),
		Assertion:  desc,
		DurationMs: result.DurationMs,
		Status:     status,
	}
}
