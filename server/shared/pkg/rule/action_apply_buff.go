// 本文件实现apply_buff规则动作：按配置参数给目标实体挂载Buff。
// 拆分为独立文件+init()注册，新增内置动作只需在rule包内新增同类文件。
package rule

import "fmt"

// applyBuffHandler apply_buff动作处理器，薄处理器，副作用经ActionAccessor端口调用服务实现。
type applyBuffHandler struct{}

// Type 返回动作类型标识"apply_buff"。
func (h *applyBuffHandler) Type() string { return "apply_buff" }

// Execute 解析参数并调用上下文访问器给目标挂载Buff，参数缺失返回错误。
func (h *applyBuffHandler) Execute(ctx ActionContext) error {
	buffID, _ := ctx.Params["buff_id"].(string)
	target, _ := ctx.Params["target"].(string)
	if buffID == "" || target == "" {
		return fmt.Errorf("动作apply_buff参数缺失，需要buff_id与target")
	}
	if err := ctx.Accessor.ApplyBuff(ctx, target, buffID); err != nil {
		return fmt.Errorf("动作apply_buff执行失败: %w", err)
	}
	return nil
}

// init 在rule包加载时向全局注册表注册apply_buff动作。
func init() { RegisterAction(&applyBuffHandler{}) }
