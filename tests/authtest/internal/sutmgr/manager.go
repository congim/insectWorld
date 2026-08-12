// Package sutmgr 测试端被测服务管理器，以子进程方式启动/停止Gateway服务并注入环境变量。
package sutmgr

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"

	"go.uber.org/zap"

	"insectworld/tests/authtest/internal/config"
)

// 进程状态枚举常量。
// 取值映射：1=未启动 2=启动中 3=运行中 4=已停止 5=异常退出
const (
	StatusNotStarted = 1 // 未启动
	StatusStarting   = 2 // 启动中
	StatusRunning    = 3 // 运行中
	StatusStopped    = 4 // 已停止
	StatusCrashed    = 5 // 异常退出
)

// 日志环形缓冲容量。
const logBufferSize = 500

// SUTManager 被测服务子进程管理器。
type SUTManager struct {
	cmd        *exec.Cmd           // 子进程命令
	status     int                 // 当前状态
	exitCode   int                 // 退出码
	logs       []string            // 日志环形缓冲
	mu         sync.RWMutex        // 读写锁
	envBuilder *EnvironmentBuilder // 环境变量构建器
	gatewayDir string              // Gateway服务目录
	logger     *zap.Logger         // 结构化日志
}

// SUTStatus 服务状态快照。
type SUTStatus struct {
	Status   int      // 当前状态：1=未启动 2=启动中 3=运行中 4=已停止 5=异常退出
	PID      int      // 进程ID
	ExitCode int      // 退出码
	Logs     []string // 最近日志行
}

// NewSUTManager 创建被测服务管理器实例。
func NewSUTManager(envBuilder *EnvironmentBuilder, gatewayDir string, logger *zap.Logger) *SUTManager {
	return &SUTManager{
		status:     StatusNotStarted,
		logs:       make([]string, 0, logBufferSize),
		envBuilder: envBuilder,
		gatewayDir: gatewayDir,
		logger:     logger,
	}
}

// Start 启动被测Gateway服务子进程。
func (m *SUTManager) Start(ctx context.Context, cfg *config.TestConfig) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == StatusRunning {
		return 0, fmt.Errorf("服务已在运行中")
	}

	env := m.envBuilder.Build(cfg)
	cmd := exec.Command("go", "run", "./cmd")
	cmd.Dir = m.gatewayDir
	cmd.Env = env

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("创建stdout管道失败: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("创建stderr管道失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("启动子进程失败: %w", err)
	}

	m.cmd = cmd
	m.status = StatusRunning
	m.exitCode = 0
	pid := cmd.Process.Pid

	go m.readLogs(stdoutPipe)
	go m.readLogs(stderrPipe)
	go m.waitProcess()

	m.logger.Info("被测服务启动成功", zap.Int("pid", pid))
	return pid, nil
}

// Stop 停止被测服务。
func (m *SUTManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil || m.status != StatusRunning {
		return fmt.Errorf("服务未运行")
	}

	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("发送终止信号失败: %w", err)
	}

	m.status = StatusStopped
	m.logger.Info("被测服务已停止")
	return nil
}

// Status 返回服务状态快照。
func (m *SUTManager) Status() SUTStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pid := 0
	if m.cmd != nil && m.cmd.Process != nil {
		pid = m.cmd.Process.Pid
	}
	logsCopy := make([]string, len(m.logs))
	copy(logsCopy, m.logs)

	return SUTStatus{
		Status:   m.status,
		PID:      pid,
		ExitCode: m.exitCode,
		Logs:     logsCopy,
	}
}

// waitProcess 等待子进程退出并更新状态。
func (m *SUTManager) waitProcess() {
	if m.cmd == nil {
		return
	}
	err := m.cmd.Wait()
	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			m.exitCode = exitErr.ExitCode()
		}
		m.status = StatusCrashed
		m.logger.Error("被测服务异常退出", zap.Int("exit_code", m.exitCode), zap.Error(err))
	} else {
		m.status = StatusStopped
	}
}

// readLogs 读取管道日志到环形缓冲。
func (m *SUTManager) readLogs(pipe interface{ Read([]byte) (int, error) }) {
	buf := make([]byte, 1024)
	for {
		n, err := pipe.Read(buf)
		if err != nil {
			return
		}
		if n > 0 {
			m.mu.Lock()
			line := string(buf[:n])
			m.logs = append(m.logs, line)
			if len(m.logs) > logBufferSize {
				m.logs = m.logs[len(m.logs)-logBufferSize:]
			}
			m.mu.Unlock()
		}
	}
}
