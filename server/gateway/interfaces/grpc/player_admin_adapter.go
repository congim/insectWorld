// Package grpc Gateway服务interfaces层gRPC handler，实现运营管理操作协议。
package grpc

import (
	"context"

	"insectworld/server/gateway/application/command"
)

// PlayerAdminAdapter PlayerAdmin接口适配器，将BanCommand/UnbanCommand适配为PlayerAdmin接口。
//
// design整合步骤7：AdminHandler的playerAdmin字段保留接口定义但实现委托给BanCommand/UnbanCommand。
// BanCommand编排封禁+踢下线，UnbanCommand仅更新账号状态不影响在线会话。
type PlayerAdminAdapter struct {
	banCmd   *command.BanCommand   // 封禁命令
	unbanCmd *command.UnbanCommand // 解封命令
	adminID  string                // 操作管理员ID，用于审计
}

// NewPlayerAdminAdapter 创建PlayerAdmin适配器实例。
//
// banCmd为封禁命令（含踢下线编排），unbanCmd为解封命令，adminID为操作管理员ID。
func NewPlayerAdminAdapter(banCmd *command.BanCommand, unbanCmd *command.UnbanCommand, adminID string) *PlayerAdminAdapter {
	return &PlayerAdminAdapter{
		banCmd:   banCmd,
		unbanCmd: unbanCmd,
		adminID:  adminID,
	}
}

// BanPlayer 封禁玩家，委托给BanCommand.Handle（含踢下线编排）。
//
// 实现PlayerAdmin接口，封禁在线玩家被即时踢下线（design.md 1.1.2节扩展功能）。
func (a *PlayerAdminAdapter) BanPlayer(ctx context.Context, playerID int64, durationMs int64, reason string) error {
	return a.banCmd.Handle(ctx, playerID, durationMs, reason, a.adminID)
}

// UnbanPlayer 解封玩家，委托给UnbanCommand.Handle（仅更新账号状态）。
//
// 实现PlayerAdmin接口，解封不影响在线会话。
func (a *PlayerAdminAdapter) UnbanPlayer(ctx context.Context, playerID int64) error {
	return a.unbanCmd.Handle(ctx, playerID, a.adminID)
}

// 确保 PlayerAdminAdapter 实现 PlayerAdmin 接口（编译期检查）。
var _ PlayerAdmin = (*PlayerAdminAdapter)(nil)
