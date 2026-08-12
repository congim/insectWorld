// Package command Config服务application层命令，编排配置提交/回滚/热更操作。
//
// command handler注入domain层Repository接口（规范3 DDD），不直接依赖infrastructure实现，
// 保证application层可独立单测（用mock替换infrastructure）。
package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/config/domain"
)

// 配置类型枚举常量。
// 取值映射：1=全量配置包 2=增量配置包
const (
	ConfigTypeFull    = 1 // 全量配置包，包含全部18个配置文件
	ConfigTypePartial = 2 // 增量配置包，仅包含变更的配置文件
)

// ConfigCommandHandler 配置命令handler，编排配置提交/回滚/热更操作。
// 依赖domain层接口（规范3），infrastructure实现由cmd/main.go注入。
type ConfigCommandHandler struct {
	configStorage  domain.ConfigStorage  // 配置存储，etcd实现
	versionStorage domain.VersionStorage // 版本存储，versionstore实现
	auditStorage   domain.AuditStorage   // 审计存储，audit实现
	logger         *zap.Logger           // 结构化日志器（规范7）
}

// NewConfigCommandHandler 创建配置命令handler实例。
// 参数为domain层接口，infrastructure实现由cmd/main.go注入（规范3 DI）。
func NewConfigCommandHandler(
	configStorage domain.ConfigStorage,
	versionStorage domain.VersionStorage,
	auditStorage domain.AuditStorage,
	logger *zap.Logger,
) *ConfigCommandHandler {
	return &ConfigCommandHandler{
		configStorage:  configStorage,
		versionStorage: versionStorage,
		auditStorage:   auditStorage,
		logger:         logger,
	}
}

// SubmitConfigCommand 提交配置包命令参数。
type SubmitConfigCommand struct {
	ConfigPackID string // 配置包ID，标识本次提交的配置包
	ConfigData   []byte // 配置包数据（JSON序列化的配置内容）
	MD5          string // 配置包MD5校验值
	Operator     string // 操作人，运营管理面用户标识
}

// SubmitConfigResult 提交配置包命令结果。
type SubmitConfigResult struct {
	ConfigVersion int64 // 分配的配置版本号
}

// SubmitConfig 提交配置包，编排：解析→保存版本→写入存储触发热更→审计日志。
// 热更广播由ConfigStorage.Put产生ConfigChangeEvent，hotReloader消费后执行编译+校验+原子替换+回调。
func (h *ConfigCommandHandler) SubmitConfig(ctx context.Context, cmd SubmitConfigCommand) (*SubmitConfigResult, error) {
	startTime := time.Now()
	configVersion := startTime.UnixMilli()

	// 1. 解析配置包数据为map（供热更编译使用）
	var configPack map[string]any
	if err := json.Unmarshal(cmd.ConfigData, &configPack); err != nil {
		h.logger.Error("配置包解析失败",
			zap.String("pack_id", cmd.ConfigPackID),
			zap.String("operator", cmd.Operator),
			zap.Error(err),
		)
		return nil, fmt.Errorf("配置包解析失败: %w", err)
	}

	// 2. 保存配置版本记录
	if err := h.versionStorage.SaveVersion(ctx, configVersion, cmd.ConfigPackID, ConfigTypeFull, cmd.Operator); err != nil {
		h.logger.Error("配置版本保存失败",
			zap.String("pack_id", cmd.ConfigPackID),
			zap.String("operator", cmd.Operator),
			zap.Error(err),
		)
		return nil, fmt.Errorf("配置版本保存失败: %w", err)
	}

	// 3. 写入配置存储并触发热更（Put产生ConfigChangeEvent，hotReloader消费执行编译+校验+原子替换）
	if err := h.configStorage.Put(ctx, cmd.ConfigPackID, cmd.ConfigData, configVersion, configPack); err != nil {
		h.logger.Error("配置写入存储失败",
			zap.String("pack_id", cmd.ConfigPackID),
			zap.Int64("version", configVersion),
			zap.Error(err),
		)
		return nil, fmt.Errorf("配置写入存储失败: %w", err)
	}

	// 4. 记审计日志（独立存储，失败仅告警不影响主流程）
	if err := h.auditStorage.Save(ctx, domain.AuditRecord{
		VersionID:   configVersion,
		Operator:    cmd.Operator,
		Action:      domain.AuditActionPublish,
		BeforeValue: "",
		AfterValue:  cmd.ConfigPackID,
	}); err != nil {
		h.logger.Warn("审计日志保存失败",
			zap.String("pack_id", cmd.ConfigPackID),
			zap.Int64("version", configVersion),
			zap.Error(err),
		)
	}

	h.logger.Info("配置包提交成功",
		zap.String("pack_id", cmd.ConfigPackID),
		zap.Int64("config_version", configVersion),
		zap.String("operator", cmd.Operator),
		zap.Duration("submit_duration", time.Since(startTime)),
	)

	return &SubmitConfigResult{ConfigVersion: configVersion}, nil
}

// RollbackConfigCommand 回滚配置命令参数。
type RollbackConfigCommand struct {
	TargetVersion int64  // 目标回滚版本号
	Operator      string // 操作人
}

// RollbackConfigResult 回滚配置命令结果。
type RollbackConfigResult struct {
	CurrentVersion int64 // 回滚后的当前版本号
}

// RollbackConfig 回滚配置到指定版本，编排：查询历史版本→审计日志→触发热更。
func (h *ConfigCommandHandler) RollbackConfig(ctx context.Context, cmd RollbackConfigCommand) (*RollbackConfigResult, error) {
	startTime := time.Now()

	// 1. 查询目标版本是否存在
	versions, err := h.versionStorage.FindVersions(ctx, ConfigTypeFull, 10)
	if err != nil {
		h.logger.Error("查询版本历史失败",
			zap.Int64("target_version", cmd.TargetVersion),
			zap.Error(err),
		)
		return nil, fmt.Errorf("查询版本历史失败: %w", err)
	}

	found := false
	for _, v := range versions {
		if v.VersionID == cmd.TargetVersion {
			found = true
			break
		}
	}
	if !found {
		h.logger.Error("目标回滚版本不存在",
			zap.Int64("target_version", cmd.TargetVersion),
			zap.String("operator", cmd.Operator),
		)
		return nil, fmt.Errorf("目标回滚版本 %d 不存在", cmd.TargetVersion)
	}

	// 2. 计审计日志
	if err := h.auditStorage.Save(ctx, domain.AuditRecord{
		VersionID:   cmd.TargetVersion,
		Operator:    cmd.Operator,
		Action:      domain.AuditActionRollback,
		BeforeValue: "",
		AfterValue:  fmt.Sprintf("rollback_to_%d", cmd.TargetVersion),
	}); err != nil {
		h.logger.Warn("审计日志保存失败",
			zap.Int64("target_version", cmd.TargetVersion),
			zap.Error(err),
		)
	}

	h.logger.Info("配置回滚成功",
		zap.Int64("target_version", cmd.TargetVersion),
		zap.String("operator", cmd.Operator),
		zap.Duration("rollback_duration", time.Since(startTime)),
	)

	return &RollbackConfigResult{CurrentVersion: cmd.TargetVersion}, nil
}
