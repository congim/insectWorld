// 本文件实现send_notify规则动作：按配置参数向目标发送通知消息。
// 经ActionAccessor端口调用服务实现，消息内容模板与推送渠道由宿主服务负责。
package rule

import "fmt"

// sendNotifyHandler send_notify动作处理器，薄处理器，副作用经ActionAccessor端口调用服务实现。
type sendNotifyHandler struct{}

// Type 返回动作类型标识"send_notify"。
func (h *sendNotifyHandler) Type() string { return "send_notify" }

// Execute 解析参数并调用上下文访问器发送通知，msg_id与target必填。
// params为通知附带参数，可选：为空时传空map避免向宿主服务传递nil。
func (h *sendNotifyHandler) Execute(ctx ActionContext) error {
	msgID, _ := ctx.Params["msg_id"].(string)
	target, _ := ctx.Params["target"].(string)
	if msgID == "" || target == "" {
		return fmt.Errorf("动作send_notify参数缺失，需要msg_id与target")
	}
	params, _ := ctx.Params["params"].(map[string]any)
	if params == nil {
		params = make(map[string]any)
	}
	if err := ctx.Accessor.SendNotify(ctx, target, msgID, params); err != nil {
		return fmt.Errorf("动作send_notify执行失败: %w", err)
	}
	return nil
}

// init 在rule包加载时向全局注册表注册send_notify动作。
func init() { RegisterAction(&sendNotifyHandler{}) }
