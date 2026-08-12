// Package e2e 测试端端到端测试执行器，编排注册→登录→心跳→鉴权→登出五步认证流程并执行断言。
package e2e

// 步骤状态枚举常量。
// 取值映射：1=通过 2=失败 3=跳过
const (
	StepPassed  = 1 // 通过
	StepFailed  = 2 // 失败
	StepSkipped = 3 // 跳过
)

// StepResult 单步测试结果。
type StepResult struct {
	StepName   string `json:"step_name"`   // 步骤名称
	Request    string `json:"request"`     // 请求JSON
	Response   string `json:"response"`    // 响应JSON
	Assertion  string `json:"assertion"`   // 断言结论中文描述
	DurationMs int64  `json:"duration_ms"` // 耗时毫秒
	Status     int    `json:"status"`      // 状态：1=通过 2=失败 3=跳过
}

// TestReport 端到端测试报告。
type TestReport struct {
	StartTime     int64        `json:"start_time"`     // 开始时间戳毫秒
	EndTime       int64        `json:"end_time"`       // 结束时间戳毫秒
	TestAccount   string       `json:"test_account"`   // 测试账号
	Steps         []StepResult `json:"steps"`          // 各步骤结果
	PassRate      int          `json:"pass_rate"`      // 通过率0-100
	OverallPassed bool         `json:"overall_passed"` // 是否全部通过
}

// buildReport 构建测试报告，计算通过率与整体结论。
func buildReport(steps []StepResult, startTime, endTime int64, account string) *TestReport {
	passedCount := 0
	executedCount := 0
	for _, step := range steps {
		if step.Status != StepSkipped {
			executedCount++
		}
		if step.Status == StepPassed {
			passedCount++
		}
	}
	passRate := 0
	if executedCount > 0 {
		passRate = passedCount * 100 / executedCount
	}
	return &TestReport{
		StartTime:     startTime,
		EndTime:       endTime,
		TestAccount:   account,
		Steps:         steps,
		PassRate:      passRate,
		OverallPassed: passedCount == executedCount && executedCount > 0,
	}
}
