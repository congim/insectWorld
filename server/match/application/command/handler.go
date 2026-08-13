// Package command Match服务application层写侧命令，编排聚合根调用、事务边界与领域事件Outbox投递。
// 对应DDD application层，不直接import infrastructure，通过依赖注入的接口操作。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/match/domain/matchticket"
)

// CreateTicketCommand 创建匹配票命令DTO。
type CreateTicketCommand struct {
	PoolID      int64 // 匹配池ID
	SubjectType int   // 匹配主体类型：1=玩家 2=联盟
	SubjectID   int64 // 匹配主体ID
	Tier        int32 // 段位
	Score       int64 // 匹配评分
	Now         int64 // 当前时间戳（毫秒）
}

// CancelTicketCommand 取消匹配命令DTO。
type CancelTicketCommand struct {
	TicketID int64 // 匹配票ID
}

// CreateTicketHandler 创建匹配票命令handler。
type CreateTicketHandler struct {
	matchTicketRepo matchticket.MatchTicketRepository // 匹配票仓储接口，infrastructure层注入
	logger          *zap.Logger                       // 结构化日志器（规范7）
}

// NewCreateTicketHandler 创建匹配票命令handler实例。
func NewCreateTicketHandler(matchTicketRepo matchticket.MatchTicketRepository, logger *zap.Logger) *CreateTicketHandler {
	return &CreateTicketHandler{matchTicketRepo: matchTicketRepo, logger: logger}
}

// Handle 处理创建匹配票命令。
// 编排：创建匹配票聚合根→加入匹配池→保存+Outbox。
func (h *CreateTicketHandler) Handle(ctx context.Context, cmd CreateTicketCommand) (*matchticket.MatchTicket, error) {
	// 1. 构造匹配票ID（TODO 后续接入雪花算法ID生成器）
	var ticketID int64 = 0

	// 2. 创建匹配票聚合根
	ticket := matchticket.NewMatchTicket(ticketID, cmd.PoolID, cmd.SubjectType, cmd.SubjectID, cmd.Tier, cmd.Score, cmd.Now)

	// 3. 保存聚合根
	if err := h.matchTicketRepo.SaveMatchTicket(ctx, ticket); err != nil {
		return nil, fmt.Errorf("保存匹配票失败: %w", err)
	}

	h.logger.Info("创建匹配票",
		zap.Int64("pool_id", cmd.PoolID),
		zap.Int("subject_type", cmd.SubjectType),
		zap.Int64("subject_id", cmd.SubjectID),
	)
	// TODO 后续接入匹配池引擎加入队列、Outbox投递事件
	return ticket, nil
}

// CancelTicketHandler 取消匹配命令handler。
type CancelTicketHandler struct {
	matchTicketRepo matchticket.MatchTicketRepository // 匹配票仓储接口，infrastructure层注入
	logger          *zap.Logger                       // 结构化日志器（规范7）
}

// NewCancelTicketHandler 创建取消匹配命令handler实例。
func NewCancelTicketHandler(matchTicketRepo matchticket.MatchTicketRepository, logger *zap.Logger) *CancelTicketHandler {
	return &CancelTicketHandler{matchTicketRepo: matchTicketRepo, logger: logger}
}

// Handle 处理取消匹配命令。
// 编排：加载匹配票聚合根→调用聚合根Cancel→保存+Outbox。
func (h *CancelTicketHandler) Handle(ctx context.Context, cmd CancelTicketCommand) error {
	// 1. 加载匹配票聚合根
	ticket, err := h.matchTicketRepo.LoadMatchTicket(ctx, cmd.TicketID)
	if err != nil {
		return fmt.Errorf("加载匹配票失败，ticketID=%d: %w", cmd.TicketID, err)
	}

	// 2. 调用聚合根Cancel
	if err := ticket.Cancel(); err != nil {
		h.logger.Warn("取消匹配失败",
			zap.Int64("ticket_id", cmd.TicketID),
			zap.Error(err),
		)
		return err
	}

	// 3. 保存聚合根
	if err := h.matchTicketRepo.SaveMatchTicket(ctx, ticket); err != nil {
		return fmt.Errorf("保存匹配票失败，ticketID=%d: %w", cmd.TicketID, err)
	}

	h.logger.Info("取消匹配", zap.Int64("ticket_id", cmd.TicketID))
	// TODO 后续接入匹配池引擎移除、Outbox投递事件
	return nil
}
