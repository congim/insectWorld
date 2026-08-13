// Package query Match服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"context"

	"go.uber.org/zap"
)

// RankListQuery 排行榜查询DTO。
type RankListQuery struct {
	RankType int   // 排行榜类型
	ZoneID   int64 // 跨服区ID
	Season   int64 // 赛季编号
	Offset   int   // 分页偏移
	Limit    int   // 分页条数
}

// RankListResult 排行榜查询结果DTO。
type RankListResult struct {
	Total   int64       // 排行榜总条目数
	Entries []RankEntry // 排名条目列表
}

// RankEntry 排名条目DTO。
type RankEntry struct {
	SubjectID int64 // 排名主体ID
	Score     int64 // 排名分数
	Rank      int   // 排名序号
}

// RankReadModel 排行榜读模型查询接口，在application层声明，infrastructure层实现。
// CQRS读侧通过读模型表t_rank_tier查询，不经过聚合根。
type RankReadModel interface {
	// QueryRankList 分页查询排行榜
	QueryRankList(ctx context.Context, rankType int, zoneID int64, season int64, offset int, limit int) (int64, []RankEntry, error)
}

// RankListQueryHandler 排行榜查询handler，CQRS读侧。
type RankListQueryHandler struct {
	rankReadModel RankReadModel // 排行榜读模型查询接口，infrastructure层注入
	logger        *zap.Logger   // 结构化日志器（规范7）
}

// NewRankListQueryHandler 创建排行榜查询handler实例。
// rankReadModel由infrastructure层实现，cmd/main.go组装时注入。
func NewRankListQueryHandler(rankReadModel RankReadModel, logger *zap.Logger) *RankListQueryHandler {
	return &RankListQueryHandler{rankReadModel: rankReadModel, logger: logger}
}

// Handle 处理排行榜查询。
func (h *RankListQueryHandler) Handle(ctx context.Context, q RankListQuery) (*RankListResult, error) {
	total, entries, err := h.rankReadModel.QueryRankList(ctx, q.RankType, q.ZoneID, q.Season, q.Offset, q.Limit)
	if err != nil {
		return nil, err
	}

	h.logger.Debug("查询排行榜",
		zap.Int("rank_type", q.RankType),
		zap.Int64("zone_id", q.ZoneID),
		zap.Int64("total", total),
	)
	return &RankListResult{Total: total, Entries: entries}, nil
}
