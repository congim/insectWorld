// Package player 定义玩家档案聚合及其持久化契约。
package player

import (
	"context"
	"fmt"
	"strings"

	gameerr "insectworld/server/game/domain/errors"
)

// Profile 是玩家成长档案聚合根，不持有资源余额等其他上下文数据。
type Profile struct {
	playerID      int64  // 玩家ID，由注册链路分配且全局唯一
	factionID     string // 阵营稳定ID，来源于绑定的游戏包版本
	nickname      string // 玩家昵称，去除首尾空白后保存
	level         int32  // 玩家等级，初始为1
	experience    int64  // 玩家经验值，初始为0
	createdAt     int64  // 创建时间戳，Unix毫秒
	configVersion string // 创建玩家时绑定的游戏包语义版本
	commandID     string // 创建命令幂等键，不对客户端展示
}

// NewProfile 创建玩家档案并校验聚合不变量。
func NewProfile(playerID int64, factionID string, nickname string, createdAt int64, configVersion string, commandID string) (*Profile, error) {
	nickname = strings.TrimSpace(nickname)
	if playerID <= 0 || factionID == "" || nickname == "" || createdAt <= 0 || configVersion == "" || commandID == "" {
		return nil, fmt.Errorf("玩家档案参数非法，playerID=%d: %w", playerID, gameerr.ErrInvalidCommand)
	}
	return &Profile{playerID: playerID, factionID: factionID, nickname: nickname, level: 1, createdAt: createdAt, configVersion: configVersion, commandID: commandID}, nil
}

// RestoreProfile 从可信持久化数据恢复玩家档案，并重新校验不变量。
func RestoreProfile(playerID int64, factionID string, nickname string, level int32, experience int64, createdAt int64, configVersion string, commandID string) (*Profile, error) {
	profile, err := NewProfile(playerID, factionID, nickname, createdAt, configVersion, commandID)
	if err != nil {
		return nil, err
	}
	if level <= 0 || experience < 0 {
		return nil, fmt.Errorf("持久化玩家成长数据非法，playerID=%d: %w", playerID, gameerr.ErrStateConflict)
	}
	profile.level = level
	profile.experience = experience
	return profile, nil
}

// PlayerID 返回玩家ID。
func (p *Profile) PlayerID() int64 { return p.playerID }

// FactionID 返回玩家阵营稳定ID。
func (p *Profile) FactionID() string { return p.factionID }

// Nickname 返回玩家昵称。
func (p *Profile) Nickname() string { return p.nickname }

// Level 返回玩家等级。
func (p *Profile) Level() int32 { return p.level }

// Experience 返回玩家经验值。
func (p *Profile) Experience() int64 { return p.experience }

// CreatedAt 返回创建时间戳，单位毫秒。
func (p *Profile) CreatedAt() int64 { return p.createdAt }

// ConfigVersion 返回玩家档案绑定的游戏包语义版本。
func (p *Profile) ConfigVersion() string { return p.configVersion }

// CommandID 返回创建命令幂等键，仅供应用与仓储协作。
func (p *Profile) CommandID() string { return p.commandID }

// Clone 返回与仓储内部状态隔离的玩家档案副本。
func (p *Profile) Clone() *Profile {
	copyValue := *p
	return &copyValue
}

// Repository 是玩家档案仓储，由Growth上下文拥有写权限。
type Repository interface {
	FindByPlayerID(ctx context.Context, playerID int64) (*Profile, error)
	FindByCommandID(ctx context.Context, commandID string) (*Profile, error)
	SaveIfAbsent(ctx context.Context, profile *Profile) (*Profile, bool, error)
}
