// Package grpc Config服务接口层gRPC handler，实现ConfigServiceServer接口。
//
// interfaces层依赖application层command/query handler（规范3 DDD），不直接依赖infrastructure。
// handler负责gRPC请求响应转换与错误码映射，业务逻辑由command/query编排。
package grpc

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/config/application/command"
	"insectworld/server/config/application/query"
	commonpb "insectworld/server/shared/proto/common"
	configpb "insectworld/server/shared/proto/config"
)

// ConfigHandler Config服务gRPC handler，实现ConfigServiceServer接口。
// 依赖application层command/query handler，由cmd/main.go注入。
type ConfigHandler struct {
	configpb.UnimplementedConfigServiceServer                                  // 嵌入未实现基类，保证前向兼容
	cmdHandler                                *command.ConfigCommandHandler    // 配置命令handler
	queryHandler                              *query.ConfigVersionQueryHandler // 版本查询handler
	logger                                    *zap.Logger                      // 结构化日志器（规范7）
}

// NewConfigHandler 创建Config服务gRPC handler实例。
// cmdHandler和queryHandler由cmd/main.go注入application层handler。
func NewConfigHandler(
	cmdHandler *command.ConfigCommandHandler,
	queryHandler *query.ConfigVersionQueryHandler,
	logger *zap.Logger,
) *ConfigHandler {
	return &ConfigHandler{
		cmdHandler:   cmdHandler,
		queryHandler: queryHandler,
		logger:       logger,
	}
}

// SubmitConfigPack 提交配置包，调用command.SubmitConfig编排校验→保存版本→热更广播→审计日志。
func (h *ConfigHandler) SubmitConfigPack(ctx context.Context, req *configpb.SubmitConfigRequest) (*configpb.SubmitConfigResponse, error) {
	result, err := h.cmdHandler.SubmitConfig(ctx, command.SubmitConfigCommand{
		ConfigPackID: req.GetConfigPackId(),
		ConfigData:   req.GetConfigData(),
		MD5:          req.GetMd5(),
		Operator:     "admin", // 运营管理面调用，operator从鉴权上下文获取
	})
	if err != nil {
		h.logger.Error("提交配置包失败",
			zap.String("pack_id", req.GetConfigPackId()),
			zap.Error(err),
		)
		return &configpb.SubmitConfigResponse{
			Header: buildErrorResponse(req.GetHeader(), err),
		}, nil
	}

	return &configpb.SubmitConfigResponse{
		Header:        buildSuccessResponse(req.GetHeader()),
		ConfigVersion: result.ConfigVersion,
	}, nil
}

// RollbackConfig 回滚配置，调用command.RollbackConfig编排查询历史→审计日志→触发热更。
func (h *ConfigHandler) RollbackConfig(ctx context.Context, req *configpb.RollbackRequest) (*configpb.RollbackResponse, error) {
	result, err := h.cmdHandler.RollbackConfig(ctx, command.RollbackConfigCommand{
		TargetVersion: req.GetTargetVersion(),
		Operator:      "admin",
	})
	if err != nil {
		h.logger.Error("回滚配置失败",
			zap.Int64("target_version", req.GetTargetVersion()),
			zap.Error(err),
		)
		return &configpb.RollbackResponse{
			Header: buildErrorResponse(req.GetHeader(), err),
		}, nil
	}

	return &configpb.RollbackResponse{
		Header:         buildSuccessResponse(req.GetHeader()),
		CurrentVersion: result.CurrentVersion,
	}, nil
}

// GetConfigVersionHistory 查询配置版本历史，调用query.Handle从读模型查询。
func (h *ConfigHandler) GetConfigVersionHistory(ctx context.Context, req *configpb.VersionHistoryRequest) (*configpb.VersionHistoryResponse, error) {
	limit := 10
	if req.GetPagination() != nil && req.GetPagination().GetPageSize() > 0 {
		limit = int(req.GetPagination().GetPageSize())
	}

	versions, err := h.queryHandler.Handle(ctx, query.VersionHistoryQuery{
		ConfigType: 1, // 全量配置包类型
		Limit:      limit,
	})
	if err != nil {
		h.logger.Error("查询配置版本历史失败", zap.Error(err))
		return &configpb.VersionHistoryResponse{
			Header: buildErrorResponse(req.GetHeader(), err),
		}, nil
	}

	pbVersions := make([]*configpb.ConfigVersion, 0, len(versions))
	for _, v := range versions {
		pbVersions = append(pbVersions, &configpb.ConfigVersion{
			Version:      v.VersionID,
			ConfigPackId: v.Version,
			SubmitTime:   v.CreateTime,
			Submitter:    v.Operator,
		})
	}

	return &configpb.VersionHistoryResponse{
		Header:   buildSuccessResponse(req.GetHeader()),
		Versions: pbVersions,
	}, nil
}

// buildSuccessResponse 构造成功响应头。
func buildSuccessResponse(reqHeader *commonpb.RequestHeader) *commonpb.ResponseHeader {
	return &commonpb.ResponseHeader{
		RequestId: reqHeader.GetRequestId(),
		TraceId:   reqHeader.GetTraceId(),
		ErrorCode: 0,
		ErrorMsg:  "",
	}
}

// buildErrorResponse 构造错误响应头，携带错误信息。
func buildErrorResponse(reqHeader *commonpb.RequestHeader, err error) *commonpb.ResponseHeader {
	return &commonpb.ResponseHeader{
		RequestId: reqHeader.GetRequestId(),
		TraceId:   reqHeader.GetTraceId(),
		ErrorCode: 15000, // Config服务错误码基址
		ErrorMsg:  fmt.Sprintf("%s", err),
	}
}
