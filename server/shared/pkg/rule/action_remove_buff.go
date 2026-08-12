// 本文件实现remove_buff规则动作：按BuffID或来源标记移除目标实体上的Buff。
// 支持两种参数形态：{buff_id,target}走RemoveBuff，{source_tag,target}走RemoveBuffByTag。
package rule

import "fmt"

// removeBuffHandler remove_buff动作处理器，薄处理器，副作用经ActionAccessor端口调用服务实现。
type removeBuffHandler struct{}

// Type 返回动作类型标识"remove_buff"。
func (h *removeBuffHandler) Type() string { return "remove_buff" }

// Execute 解析参数并调用上下文访问器移除Buff。
// buff_id与source_tag二选一：优先buff_id精确移除，否则按source_tag批量移除。
func (h *removeBuffHandler) Execute(ctx ActionContext) error {
	target, _ := ctx.Params["target"].(string)
	if target == "" {
		return fmt.Errorf("动作remove_buff参数缺失，需要target")
	}
	if buffID, _ := ctx.Params["buff_id"].(string); buffID != "" {
		if err := ctx.Accessor.RemoveBuff(ctx, target, buffID); err != nil {
			return fmt.Errorf("动作remove_buff执行失败: %w", err)
		}
		return nil
	}
	if sourceTag, _ := ctx.Params["source_tag"].(string); sourceTag != "" {
		if err := ctx.Accessor.RemoveBuffByTag(ctx, target, sourceTag); err != nil {
			return fmt.Errorf("动作remove_buff执行失败: %w", err)
		}
		return nil
	}
	return fmt.Errorf("动作remove_buff参数缺失，需要buff_id或source_tag")
}

// init 在rule包加载时向全局注册表注册remove_buff动作。
func init() { RegisterAction(&removeBuffHandler{}) }
