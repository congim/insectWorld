// Package ranking 排行榜聚合根，维护跨服排行榜的排名数据与快照。
// Ranking聚合根按排行榜类型分片，提供排名更新与TopN查询能力。
// 对应spec.md 5.1.7.4节Match上下文功能4"排行榜"。
package ranking

import (
	"context"
	"fmt"

	matcherr "insectworld/server/match/domain/errors"
)

// 排行榜类型常量（规范1）。
const (
	RankTypePower       = 1 // 战力榜
	RankTypeKill        = 2 // 击杀榜
	RankTypeResource    = 3 // 资源榜
	RankTypeAlliance    = 4 // 联盟榜
	RankTypeBattlefield = 5 // 战场积分榜
)

// 排行榜快照状态常量（规范1）。
const (
	SnapshotActive   = 1 // 当前有效快照
	SnapshotArchived = 2 // 已归档快照
)

// Ranking 排行榜聚合根，维护某个排行榜类型的排名数据。
type Ranking struct {
	rankID     int64       // 排行榜ID，全局唯一
	rankType   int         // 排行榜类型：1=战力 2=杀伤 3=资源 4=联盟 5=战场积分
	zoneID     int64       // 跨服区ID，跨服排行榜所属的跨服区
	season     int64       // 赛季编号，赛季制排行榜有效
	entries    []RankEntry // 排名条目列表，按排名升序排列
	maxSize    int         // 排行榜最大容量（TopN），从配置查询
	snapshot   int         // 快照状态：1=当前有效 2=已归档
	updateTime int64       // 最后更新时间戳（毫秒）
}

// RankEntry 排名条目值对象。
type RankEntry struct {
	SubjectID int64 // 排名主体ID（玩家ID或联盟ID）
	Score     int64 // 排名分数（规范8用int64）
	Rank      int   // 排名序号，从1开始
}

// NewRanking 创建排行榜聚合根实例。
// rankType为排行榜类型，maxSize为TopN容量从配置查询注入。
func NewRanking(rankID int64, rankType int, zoneID int64, season int64, maxSize int) *Ranking {
	return &Ranking{
		rankID:   rankID,
		rankType: rankType,
		zoneID:   zoneID,
		season:   season,
		maxSize:  maxSize,
		entries:  make([]RankEntry, 0, maxSize),
		snapshot: SnapshotActive,
	}
}

// RankID 返回排行榜ID。
func (r *Ranking) RankID() int64 { return r.rankID }

// RankType 返回排行榜类型。
func (r *Ranking) RankType() int { return r.rankType }

// Entries 返回排行榜条目列表（按排名升序）。
func (r *Ranking) Entries() []RankEntry { return r.entries }

// UpdateScore 更新主体分数，重新计算排名并截断至TopN。
// subjectID为排名主体ID，score为最新分数，now为当前时间戳（毫秒）。
func (r *Ranking) UpdateScore(subjectID int64, score int64, now int64) (*RankUpdatedEvent, error) {
	if r.snapshot != SnapshotActive {
		return nil, fmt.Errorf("排行榜已归档，rankID=%d: %w", r.rankID, matcherr.ErrInvalidParams)
	}

	// 移除已有条目（更新场景）
	for i, e := range r.entries {
		if e.SubjectID == subjectID {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			break
		}
	}

	// 插入新条目（按分数降序插入）
	entry := RankEntry{SubjectID: subjectID, Score: score}
	inserted := false
	for i, e := range r.entries {
		if score > e.Score {
			r.entries = append(r.entries[:i], append([]RankEntry{entry}, r.entries[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		r.entries = append(r.entries, entry)
	}

	// 截断至TopN
	if len(r.entries) > r.maxSize {
		r.entries = r.entries[:r.maxSize]
	}

	// 重算排名
	oldRank := 0
	for i := range r.entries {
		r.entries[i].Rank = i + 1
		if r.entries[i].SubjectID == subjectID {
			oldRank = r.entries[i].Rank
		}
	}

	r.updateTime = now

	return &RankUpdatedEvent{
		RankID:    r.rankID,
		RankType:  r.rankType,
		SubjectID: subjectID,
		Score:     score,
		Rank:      oldRank,
	}, nil
}

// Archive 归档排行榜快照，用于赛季结束时生成历史快照。
func (r *Ranking) Archive() {
	r.snapshot = SnapshotArchived
}

// RankUpdatedEvent 排名更新事件。
type RankUpdatedEvent struct {
	RankID    int64 // 排行榜ID
	RankType  int   // 排行榜类型
	SubjectID int64 // 排名主体ID
	Score     int64 // 更新后分数
	Rank      int   // 更新后排名
}

// RankingRepository 排行榜聚合根仓储接口，在domain层声明（规范3）。
type RankingRepository interface {
	// LoadRanking 加载排行榜聚合根
	LoadRanking(ctx context.Context, rankID int64) (*Ranking, error)
	// SaveRanking 保存排行榜聚合根
	SaveRanking(ctx context.Context, r *Ranking) error
}
