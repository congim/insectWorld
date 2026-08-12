// Package webserver 测试端HTTP服务器，提供静态文件服务与REST API。
package webserver

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"

	"go.uber.org/zap"

	"insectworld/tests/authtest/internal/config"
	"insectworld/tests/authtest/internal/contract"
	"insectworld/tests/authtest/internal/e2e"
	"insectworld/tests/authtest/internal/envinit"
	"insectworld/tests/authtest/internal/sutmgr"
	"insectworld/tests/authtest/internal/wsclient"
)

// WebServer HTTP服务器，提供静态文件与REST API。
type WebServer struct {
	listenAddr   string        // 监听地址
	envHandlers  *EnvHandlers  // 环境初始化handler
	sutHandlers  *SUTHandlers  // 服务管理handler
	authHandlers *AuthHandlers // 认证测试handler
	e2eHandlers  *E2EHandlers  // 端到端测试handler
	logger       *zap.Logger   // 结构化日志
}

// NewWebServer 创建HTTP服务器实例。
func NewWebServer(listenAddr string, envH *EnvHandlers, sutH *SUTHandlers, authH *AuthHandlers, e2eH *E2EHandlers, logger *zap.Logger) *WebServer {
	return &WebServer{
		listenAddr:   listenAddr,
		envHandlers:  envH,
		sutHandlers:  sutH,
		authHandlers: authH,
		e2eHandlers:  e2eH,
		logger:       logger,
	}
}

// Start 启动HTTP服务器。
func (s *WebServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	webFS, err := fs.Sub(embeddedFS, "web")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	mux.HandleFunc("/api/env/init", s.envHandlers.HandleInit)
	mux.HandleFunc("/api/env/status", s.envHandlers.HandleStatus)
	mux.HandleFunc("/api/env/cleanup", s.envHandlers.HandleCleanup)
	mux.HandleFunc("/api/env/destroy", s.envHandlers.HandleDestroy)

	mux.HandleFunc("/api/sut/start", s.sutHandlers.HandleStart)
	mux.HandleFunc("/api/sut/stop", s.sutHandlers.HandleStop)
	mux.HandleFunc("/api/sut/status", s.sutHandlers.HandleStatus)

	mux.HandleFunc("/api/auth/send", s.authHandlers.HandleSend)

	mux.HandleFunc("/api/e2e/run", s.e2eHandlers.HandleRun)
	mux.HandleFunc("/api/e2e/failure", s.e2eHandlers.HandleFailure)
	mux.HandleFunc("/api/e2e/report", s.e2eHandlers.HandleReport)

	server := &http.Server{Addr: s.listenAddr, Handler: mux}

	go func() {
		s.logger.Info("Web服务器启动", zap.String("addr", s.listenAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Web服务器异常", zap.Error(err))
		}
	}()

	go func() {
		<-ctx.Done()
		s.logger.Info("Web服务器关闭中")
		_ = server.Shutdown(context.Background())
	}()

	return nil
}

// writeJSON 统一JSON响应辅助函数。
func writeJSON(w http.ResponseWriter, success bool, data interface{}, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"success":   success,
		"data":      data,
		"error_msg": errMsg,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "响应序列化失败", http.StatusInternalServerError)
	}
}

// EnvHandlers 环境初始化API handler。
type EnvHandlers struct {
	initializer *envinit.DatabaseInitializer // 数据库初始化器
	logger      *zap.Logger                  // 结构化日志
}

// NewEnvHandlers 创建环境初始化handler实例。
func NewEnvHandlers(initializer *envinit.DatabaseInitializer, logger *zap.Logger) *EnvHandlers {
	return &EnvHandlers{initializer: initializer, logger: logger}
}

// HandleInit 环境初始化API。
func (h *EnvHandlers) HandleInit(w http.ResponseWriter, r *http.Request) {
	result, err := h.initializer.Init(r.Context())
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, result, "")
}

// HandleStatus 环境状态查询API。
func (h *EnvHandlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	result, err := h.initializer.Status(r.Context())
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, result, "")
}

// HandleCleanup 环境清理API。
func (h *EnvHandlers) HandleCleanup(w http.ResponseWriter, r *http.Request) {
	result, err := h.initializer.Cleanup(r.Context())
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, result, "")
}

// HandleDestroy 环境销毁API。
func (h *EnvHandlers) HandleDestroy(w http.ResponseWriter, r *http.Request) {
	if err := h.initializer.Destroy(r.Context()); err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, nil, "")
}

// SUTHandlers 服务管理API handler。
type SUTHandlers struct {
	sutMgr *sutmgr.SUTManager // 被测服务管理器
	cfg    *config.TestConfig // 测试配置
	logger *zap.Logger        // 结构化日志
}

// NewSUTHandlers 创建服务管理handler实例。
func NewSUTHandlers(sutMgr *sutmgr.SUTManager, cfg *config.TestConfig, logger *zap.Logger) *SUTHandlers {
	return &SUTHandlers{sutMgr: sutMgr, cfg: cfg, logger: logger}
}

// HandleStart 启动被测服务API。
func (h *SUTHandlers) HandleStart(w http.ResponseWriter, r *http.Request) {
	pid, err := h.sutMgr.Start(r.Context(), h.cfg)
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, map[string]int{"pid": pid}, "")
}

// HandleStop 停止被测服务API。
func (h *SUTHandlers) HandleStop(w http.ResponseWriter, r *http.Request) {
	if err := h.sutMgr.Stop(); err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, nil, "")
}

// HandleStatus 服务状态查询API。
func (h *SUTHandlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, true, h.sutMgr.Status(), "")
}

// AuthHandlers 认证测试API handler。
type AuthHandlers struct {
	wsClient *wsclient.AuthWSClient // WebSocket客户端
	logger   *zap.Logger            // 结构化日志
}

// NewAuthHandlers 创建认证测试handler实例。
func NewAuthHandlers(wsClient *wsclient.AuthWSClient, logger *zap.Logger) *AuthHandlers {
	return &AuthHandlers{wsClient: wsClient, logger: logger}
}

// HandleSend 发送认证消息API。
func (h *AuthHandlers) HandleSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type     string `json:"type"`
		Username string `json:"username"`
		Password string `json:"password"`
		Token    string `json:"token"`
		PlayerID int64  `json:"player_id"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, false, nil, "请求解析失败: "+err.Error())
		return
	}

	msg := contract.AuthMessage{
		Type:     req.Type,
		Username: req.Username,
		Password: req.Password,
		Token:    req.Token,
		PlayerID: req.PlayerID,
		DeviceID: req.DeviceID,
	}
	result, err := h.wsClient.Send(r.Context(), msg)
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, result, "")
}

// E2EHandlers 端到端测试API handler。
type E2EHandlers struct {
	e2eRunner     *e2e.E2ERunner         // 端到端编排器
	failureRunner *e2e.FailureCaseRunner // 失败场景执行器
	logger        *zap.Logger            // 结构化日志
}

// NewE2EHandlers 创建端到端测试handler实例。
func NewE2EHandlers(e2eRunner *e2e.E2ERunner, failureRunner *e2e.FailureCaseRunner, logger *zap.Logger) *E2EHandlers {
	return &E2EHandlers{e2eRunner: e2eRunner, failureRunner: failureRunner, logger: logger}
}

// HandleRun 执行端到端测试API。
func (h *E2EHandlers) HandleRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		DeviceID string `json:"device_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	report, err := h.e2eRunner.Run(r.Context(), req.Username, req.Password, req.DeviceID)
	if err != nil {
		writeJSON(w, false, nil, err.Error())
		return
	}
	writeJSON(w, true, report, "")
}

// HandleFailure 执行失败场景测试API。
func (h *E2EHandlers) HandleFailure(w http.ResponseWriter, r *http.Request) {
	results := h.failureRunner.RunAll(r.Context())
	writeJSON(w, true, results, "")
}

// HandleReport 查询最近测试报告API。
func (h *E2EHandlers) HandleReport(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, true, nil, "请通过/api/e2e/run触发测试获取报告")
}
