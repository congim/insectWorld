// 本文件实现trigger_combat规则动作：按配置参数触发攻击者与防御者之间的一场战斗。
// 经ActionAccessor端口调用服务实现，战斗结算细节由宿主服务（Combat服务）负责。
package rule

import "fmt"

// triggerCombatHandler trigger_combat动作处理器，薄处理器，副作用经ActionAccessor端口调用服务实现。
type triggerCombatHandler struct{}

// Type 返回动作类型标识"trigger_combat"。
func (h *triggerCombatHandler) Type() string { return "trigger_combat" }

// Execute 解析参数并调用上下文访问器触发战斗，参数缺失返回错误。
func (h *triggerCombatHandler) Execute(ctx ActionContext) error {
	attackerID, _ := ctx.Params["attacker_id"].(string)
	defenderID, _ := ctx.Params["defender_id"].(string)
	if attackerID == "" || defenderID == "" {
		return fmt.Errorf("动作trigger_combat参数缺失，需要attacker_id与defender_id")
	}
	if err := ctx.Accessor.TriggerCombat(ctx, attackerID, defenderID); err != nil {
		return fmt.Errorf("动作trigger_combat执行失败: %w", err)
	}
	return nil
}

// init 在rule包加载时向全局注册表注册trigger_combat动作。
func init() { RegisterAction(&triggerCombatHandler{}) }
