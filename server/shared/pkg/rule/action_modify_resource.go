// 本文件实现modify_resource规则动作：按配置参数修改目标资源余额。
// amount为正增加、为负扣减，int64整型（AGENTS.md规范8），经ActionAccessor端口调用服务实现。
package rule

import "fmt"

// modifyResourceHandler modify_resource动作处理器，薄处理器，副作用经ActionAccessor端口调用服务实现。
type modifyResourceHandler struct{}

// Type 返回动作类型标识"modify_resource"。
func (h *modifyResourceHandler) Type() string { return "modify_resource" }

// Execute 解析参数并调用上下文访问器修改资源余额，参数缺失或类型错误返回错误。
func (h *modifyResourceHandler) Execute(ctx ActionContext) error {
	resourceID, _ := ctx.Params["resource_id"].(string)
	target, _ := ctx.Params["target"].(string)
	amount, ok := parseInt64Param(ctx.Params, "amount")
	if resourceID == "" || target == "" || !ok {
		return fmt.Errorf("动作modify_resource参数缺失或类型错误，需要resource_id、amount与target")
	}
	if err := ctx.Accessor.ModifyResource(ctx, target, resourceID, amount); err != nil {
		return fmt.Errorf("动作modify_resource执行失败: %w", err)
	}
	return nil
}

// init 在rule包加载时向全局注册表注册modify_resource动作。
func init() { RegisterAction(&modifyResourceHandler{}) }
