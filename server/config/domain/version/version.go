// Package version 配置版本聚合根，封装版本状态变更边界。
//
// ConfigVersion 聚合根封装配置版本的创建/发布/回滚/删除状态流转，
// 方法对应 domain 层 AuditActionCreate/Publish/Rollback/Delete 枚举，
// 不引入新业务功能（不过度设计，规范4）。
// 状态流转见 design.md 2.1.3.2：
//   Created -> Published -> RolledBack -> Published（允许重新发布）
//   Created/Published/RolledBack -> Deleted（终态）
//
// domain 层零外部依赖（规范3），仅 import context 与本服务 domain/errors。
package version

import (
	"context"
	"fmt"
	"time"

	configerr "insectworld/server/config/domain/errors"
)

// 配置版本状态常量（规范1，就近归属）。
// 取值映射：1=已创建 2=已发布 3=已回滚 4=已删除
const (
	StatusCreated    = 1 // 已创建，版本已生成但未发布生效
	StatusPublished  = 2 // 已发布，版本生效中
	StatusRolledBack = 3 // 已回滚，退回历史版本
	StatusDeleted    = 4 // 已删除，终态
)

// ConfigVersion 配置版本聚合根，封装版本状态变更边界。
type ConfigVersion struct {
	versionID  int64  // 版本ID，全局唯一，由时间戳生成
	version    string // 版本号字符串，对应配置包ID
	configType int    // 配置类型：1=全量配置包 2=增量配置包
	operator   string // 操作人，运营管理面用户标识
	status     int    // 版本状态：1=已创建 2=已发布 3=已回滚 4=已删除
	createTime int64  // 创建时间戳，int64毫秒级（规范8）
}

// NewConfigVersion 创建配置版本聚合根实例，初始状态置为 StatusCreated。
// 构造即创建，调用方在持久化后通过 Create 方法产出领域事件用于 Outbox 投递。
func NewConfigVersion(versionID int64, version string, configType int, operator string) *ConfigVersion {
	return &ConfigVersion{
		versionID:  versionID,
		version:    version,
		configType: configType,
		operator:   operator,
		status:     StatusCreated,
		createTime: time.Now().UnixMilli(),
	}
}

// Create 产出配置版本创建领域事件。
// 前置状态：StatusCreated（构造时已进入此状态），非法状态返回错误。
func (v *ConfigVersion) Create(ctx context.Context) (*VersionCreatedEvent, error) {
	if v.status != StatusCreated {
		return nil, fmt.Errorf("配置版本创建失败，状态非法，当前状态=%d，期望状态=%d: %w",
			v.status, StatusCreated, configerr.ErrVersionConflict)
	}
	return &VersionCreatedEvent{
		VersionID:  v.versionID,
		Version:    v.version,
		ConfigType: v.configType,
		Operator:   v.operator,
		CreateTime: v.createTime,
	}, nil
}

// Publish 发布配置版本，状态从 Created 或 RolledBack 转为 Published。
// 支持回滚后重新发布（对应 design.md 2.1.3.2 状态图）。
func (v *ConfigVersion) Publish(ctx context.Context) (*VersionPublishedEvent, error) {
	if v.status != StatusCreated && v.status != StatusRolledBack {
		return nil, fmt.Errorf("配置版本发布失败，状态非法，当前状态=%d，期望状态=%d或%d: %w",
			v.status, StatusCreated, StatusRolledBack, configerr.ErrVersionConflict)
	}
	v.status = StatusPublished
	return &VersionPublishedEvent{
		VersionID:  v.versionID,
		Version:    v.version,
		ConfigType: v.configType,
		Operator:   v.operator,
	}, nil
}

// Rollback 回滚配置版本，状态从 Published 转为 RolledBack。
// 仅已发布版本可回滚，回滚后允许重新发布。
func (v *ConfigVersion) Rollback(ctx context.Context) (*VersionRolledBackEvent, error) {
	if v.status != StatusPublished {
		return nil, fmt.Errorf("配置版本回滚失败，状态非法，当前状态=%d，期望状态=%d: %w",
			v.status, StatusPublished, configerr.ErrVersionConflict)
	}
	v.status = StatusRolledBack
	return &VersionRolledBackEvent{
		VersionID:  v.versionID,
		Version:    v.version,
		ConfigType: v.configType,
		Operator:   v.operator,
	}, nil
}

// Delete 删除配置版本，状态转为 Deleted 终态。
// Created/Published/RolledBack 状态均可删除，Deleted 为终态不可逆。
func (v *ConfigVersion) Delete(ctx context.Context) error {
	if v.status == StatusDeleted {
		return fmt.Errorf("配置版本删除失败，版本已删除，versionID=%d: %w",
			v.versionID, configerr.ErrVersionConflict)
	}
	v.status = StatusDeleted
	return nil
}

// VersionID 返回版本ID。
func (v *ConfigVersion) VersionID() int64 { return v.versionID }

// Status 返回版本状态：1=已创建 2=已发布 3=已回滚 4=已删除。
func (v *ConfigVersion) Status() int { return v.status }

// ConfigType 返回配置类型：1=全量配置包 2=增量配置包。
func (v *ConfigVersion) ConfigType() int { return v.configType }

// VersionCreatedEvent 配置版本创建领域事件。
type VersionCreatedEvent struct {
	VersionID  int64  // 版本ID
	Version    string // 版本号字符串
	ConfigType int    // 配置类型：1=全量 2=增量
	Operator   string // 操作人
	CreateTime int64  // 创建时间戳毫秒
}

// VersionPublishedEvent 配置版本发布领域事件。
type VersionPublishedEvent struct {
	VersionID  int64  // 版本ID
	Version    string // 版本号字符串
	ConfigType int    // 配置类型：1=全量 2=增量
	Operator   string // 操作人
}

// VersionRolledBackEvent 配置版本回滚领域事件。
type VersionRolledBackEvent struct {
	VersionID  int64  // 版本ID
	Version    string // 版本号字符串
	ConfigType int    // 配置类型：1=全量 2=增量
	Operator   string // 操作人
}
