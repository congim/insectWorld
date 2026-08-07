// Package e2e 测试端端到端测试执行器，编排注册→登录→心跳→鉴权→登出五步认证流程并执行断言。
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/authtest/internal/contract"
	"insectworld/server/authtest/internal/wsclient"
)

// 默认测试账号常量（规范1就近归属）。
const (
	DefaultTestUsername = "e2e_test_user"   // 默认测试用户名
	DefaultTestPassword = "E2eTest123"      // 默认测试密码
	DefaultTestDeviceID = "e2e_test_device" // 默认测试设备ID
)

// E2ERunner 端到端五步流程编排器。
type E2ERunner struct {
	wsClient  *wsclient.AuthWSClient // WebSocket客户端
	assertion *Assertion             // 断言器
	logger    *zap.Logger            // 结构化日志
}

// NewE2ERunner 创建端到端编排器实例。
func NewE2ERunner(wsClient *wsclient.AuthWSClient, assertion *Assertion, logger *zap.Logger) *E2ERunner {
	return &E2ERunner{
		wsClient:  wsClient,
		assertion: assertion,
		logger:    logger,
	}
}

// Run 执行端到端五步流程。
func (r *E2ERunner) Run(ctx context.Context, username, password, deviceID string) (*TestReport, error) {
	if username == "" {
		username = DefaultTestUsername
	}
	if password == "" {
		password = DefaultTestPassword
	}
	if deviceID == "" {
		deviceID = DefaultTestDeviceID
	}

	startTime := time.Now().UnixMilli()
	steps := make([]StepResult, 0, 5)

	step1 := r.stepRegister(ctx, username, password)
	steps = append(steps, step1)
	if step1.Status != StepPassed {
		steps = append(steps, skipRemaining(4)...)
		return buildReport(steps, startTime, time.Now().UnixMilli(), username), nil
	}
	registeredPlayerID := step1.extractPlayerID()

	step2 := r.stepLogin(ctx, username, password, deviceID, registeredPlayerID)
	steps = append(steps, step2)
	if step2.Status != StepPassed {
		steps = append(steps, skipRemaining(3)...)
		return buildReport(steps, startTime, time.Now().UnixMilli(), username), nil
	}
	token := step2.extractToken()
	playerID := step2.extractPlayerID()

	step3 := r.stepHeartbeat(ctx, token, playerID)
	steps = append(steps, step3)

	step4 := r.stepAuthenticate(ctx, token, playerID)
	steps = append(steps, step4)

	step5 := r.stepLogout(ctx, token, playerID)
	steps = append(steps, step5)

	return buildReport(steps, startTime, time.Now().UnixMilli(), username), nil
}

// stepRegister 执行注册步骤。
func (r *E2ERunner) stepRegister(ctx context.Context, username, password string) StepResult {
	msg := contract.AuthMessage{Type: contract.MsgTypeRegister, Username: username, Password: password}
	return r.executeStep(ctx, "注册", msg, func(resp contract.AuthResponse) (bool, string) {
		return r.assertion.AssertRegister(resp)
	})
}

// stepLogin 执行登录步骤。
func (r *E2ERunner) stepLogin(ctx context.Context, username, password, deviceID string, expectedPlayerID int64) StepResult {
	msg := contract.AuthMessage{Type: contract.MsgTypeLogin, Username: username, Password: password, DeviceID: deviceID}
	return r.executeStep(ctx, "登录", msg, func(resp contract.AuthResponse) (bool, string) {
		return r.assertion.AssertLogin(resp, expectedPlayerID)
	})
}

// stepHeartbeat 执行心跳步骤。
func (r *E2ERunner) stepHeartbeat(ctx context.Context, token string, playerID int64) StepResult {
	msg := contract.AuthMessage{Type: contract.MsgTypeHeartbeat, Token: token, PlayerID: playerID}
	return r.executeStep(ctx, "心跳", msg, func(resp contract.AuthResponse) (bool, string) {
		return r.assertion.AssertHeartbeat(resp)
	})
}

// stepAuthenticate 执行鉴权步骤。
func (r *E2ERunner) stepAuthenticate(ctx context.Context, token string, expectedPlayerID int64) StepResult {
	msg := contract.AuthMessage{Type: contract.MsgTypeAuthenticate, Token: token}
	return r.executeStep(ctx, "鉴权", msg, func(resp contract.AuthResponse) (bool, string) {
		return r.assertion.AssertAuthenticate(resp, expectedPlayerID)
	})
}

// stepLogout 执行登出步骤。
func (r *E2ERunner) stepLogout(ctx context.Context, token string, playerID int64) StepResult {
	msg := contract.AuthMessage{Type: contract.MsgTypeLogout, Token: token, PlayerID: playerID}
	return r.executeStep(ctx, "登出", msg, func(resp contract.AuthResponse) (bool, string) {
		return r.assertion.AssertLogout(resp)
	})
}

// executeStep 执行单步请求与断言。
func (r *E2ERunner) executeStep(ctx context.Context, name string, msg contract.AuthMessage, assertFn func(contract.AuthResponse) (bool, string)) StepResult {
	reqJSON, _ := json.Marshal(msg)
	result, err := r.wsClient.Send(ctx, msg)
	if err != nil {
		return StepResult{
			StepName:  name,
			Request:   string(reqJSON),
			Assertion: fmt.Sprintf("请求失败: %v", err),
			Status:    StepFailed,
		}
	}
	respJSON, _ := json.Marshal(result.Response)
	passed, assertionDesc := assertFn(result.Response)
	status := StepPassed
	if !passed {
		status = StepFailed
	}
	return StepResult{
		StepName:   name,
		Request:    string(reqJSON),
		Response:   string(respJSON),
		Assertion:  assertionDesc,
		DurationMs: result.DurationMs,
		Status:     status,
	}
}

// skipRemaining 生成跳过的步骤。
func skipRemaining(count int) []StepResult {
	names := []string{"登录", "心跳", "鉴权", "登出"}
	result := make([]StepResult, 0, count)
	for i := 0; i < count && i < len(names); i++ {
		result = append(result, StepResult{StepName: names[i], Status: StepSkipped, Assertion: "前置步骤失败，跳过"})
	}
	return result
}

// extractPlayerID 从步骤响应中提取玩家ID。
func (s StepResult) extractPlayerID() int64 {
	var resp contract.AuthResponse
	_ = json.Unmarshal([]byte(s.Response), &resp)
	return resp.PlayerID
}

// extractToken 从步骤响应中提取令牌。
func (s StepResult) extractToken() string {
	var resp contract.AuthResponse
	_ = json.Unmarshal([]byte(s.Response), &resp)
	return resp.Token
}
