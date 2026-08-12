// 本文件实现spawn_entity规则动作：按模板在指定位置生成实体。
// 生成的实体ID写回ActionContext.Output供后续动作与审计使用（ADR-002 3.2）。
package rule

import "fmt"

// spawnEntityHandler spawn_entity动作处理器，薄处理器，副作用经ActionAccessor端口调用服务实现。
type spawnEntityHandler struct{}

// Type 返回动作类型标识"spawn_entity"。
func (h *spawnEntityHandler) Type() string { return "spawn_entity" }

// Execute 解析参数并调用上下文访问器生成实体，实体ID写回Output["entity_id"]。
// owner允许为空表示无主实体，template_id与position必填。
func (h *spawnEntityHandler) Execute(ctx ActionContext) error {
	templateID, _ := ctx.Params["template_id"].(string)
	owner, _ := ctx.Params["owner"].(string)
	posX, posY, ok := parsePositionParam(ctx.Params, "position")
	if templateID == "" || !ok {
		return fmt.Errorf("动作spawn_entity参数缺失或类型错误，需要template_id与position{x,y}")
	}
	entityID, err := ctx.Accessor.SpawnEntity(ctx, templateID, posX, posY, owner)
	if err != nil {
		return fmt.Errorf("动作spawn_entity执行失败: %w", err)
	}
	// 结果写入区懒初始化：规则引擎可能未预分配Output，写入前确保非nil避免panic。
	if ctx.Output == nil {
		ctx.Output = make(map[string]any)
	}
	ctx.Output["entity_id"] = entityID
	return nil
}

// init 在rule包加载时向全局注册表注册spawn_entity动作。
func init() { RegisterAction(&spawnEntityHandler{}) }
