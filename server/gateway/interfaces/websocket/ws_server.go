package websocket

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// wsUpgrader WebSocket连接升级器。
var wsUpgrader = websocket.Upgrader{
	HandshakeTimeout: 5 * time.Second,                            // 握手超时
	ReadBufferSize:   4096,                                       // 读缓冲区大小
	WriteBufferSize:  4096,                                       // 写缓冲区大小
	CheckOrigin:      func(r *http.Request) bool { return true }, // 允许任意来源（测试环境）
}

// WSServer WebSocket HTTP服务器，监听认证端点并分发消息到WSAuthHandler。
type WSServer struct {
	handler    *WSAuthHandler // 认证消息处理器
	httpServer *http.Server   // HTTP服务器
	logger     *zap.Logger    // 结构化日志
	mu         sync.Mutex     // 互斥锁
}

// NewWSServer 创建WebSocket HTTP服务器实例。
func NewWSServer(handler *WSAuthHandler, logger *zap.Logger) *WSServer {
	return &WSServer{
		handler: handler,
		logger:  logger,
	}
}

// Start 启动WebSocket HTTP服务器。
func (s *WSServer) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", s.handleAuth)

	s.mu.Lock()
	s.httpServer = &http.Server{Addr: addr, Handler: mux}
	s.mu.Unlock()

	go func() {
		s.logger.Info("WebSocket服务启动", zap.String("addr", addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("WebSocket服务异常", zap.Error(err))
		}
	}()

	go func() {
		<-ctx.Done()
		s.logger.Info("WebSocket服务关闭中")
		_ = s.httpServer.Shutdown(context.Background())
	}()

	return nil
}

// handleAuth 处理/auth端点的WebSocket连接。
func (s *WSServer) handleAuth(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Warn("WebSocket升级失败", zap.Error(err))
		return
	}
	defer conn.Close()

	connID := r.RemoteAddr
	sourceIP := r.RemoteAddr

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		resp, err := s.handler.HandleMessage(r.Context(), msg, sourceIP, connID)
		if err != nil {
			s.logger.Warn("消息处理失败", zap.Error(err))
			break
		}

		if err := conn.WriteMessage(websocket.TextMessage, resp); err != nil {
			break
		}
	}
}
