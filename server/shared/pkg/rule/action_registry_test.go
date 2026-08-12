// 本文件覆盖ADR-002规则动作注册表（action_registry.go）与7个内置动作的单元测试：
// 注册表增删查、重复注册报错、全局注册表全量校验、7个动作Execute参数透传（mock Accessor）、
// 参数缺失不panic、题材层blank import扩展模式模拟（框架零改动）。
package rule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// builtinActionTypes 7个内置动作类型，与ADR-002 3.5参数契约对齐，用于启动校验与注册断言。
var builtinActionTypes = []string{
	"apply_buff",
	"remove_buff",
	"modify_resource",
	"spawn_entity",
	"trigger_combat",
	"send_notify",
	"change_terrain",
}

// accessorCall 记录一次ActionAccessor端口调用，供断言参数透传。
type accessorCall struct {
	method string         // 端口方法名，如"ApplyBuff"
	args   map[string]any // 参数快照：参数名到参数值，断言时按具体类型读取
}

// mockActionAccessor 测试用ActionAccessor mock，按调用顺序记录全部端口调用并模拟成功返回。
type mockActionAccessor struct {
	calls       []accessorCall // 端口调用记录，按时间顺序追加
	entityIDSeq int64          // 实体ID自增种子，模拟SpawnEntity生成ID
}

// call 追加一条端口调用记录。
func (m *mockActionAccessor) call(method string, args map[string]any) {
	m.calls = append(m.calls, accessorCall{method: method, args: args})
}

// findCall 查找指定端口方法的首次调用记录，未找到返回false。
func (m *mockActionAccessor) findCall(method string) (accessorCall, bool) {
	for _, c := range m.calls {
		if c.method == method {
			return c, true
		}
	}
	return accessorCall{}, false
}

// ApplyBuff mock：记录调用参数，模拟挂载Buff成功。
func (m *mockActionAccessor) ApplyBuff(ctx ActionContext, target string, buffID string) error {
	m.call("ApplyBuff", map[string]any{"target": target, "buffID": buffID})
	return nil
}

// RemoveBuff mock：记录调用参数，模拟按BuffID移除成功。
func (m *mockActionAccessor) RemoveBuff(ctx ActionContext, target string, buffID string) error {
	m.call("RemoveBuff", map[string]any{"target": target, "buffID": buffID})
	return nil
}

// RemoveBuffByTag mock：记录调用参数，模拟按来源标记批量移除成功。
func (m *mockActionAccessor) RemoveBuffByTag(ctx ActionContext, target string, sourceTag string) error {
	m.call("RemoveBuffByTag", map[string]any{"target": target, "sourceTag": sourceTag})
	return nil
}

// ModifyResource mock：记录调用参数，模拟修改资源余额成功。
func (m *mockActionAccessor) ModifyResource(ctx ActionContext, target string, resourceID string, amount int64) error {
	m.call("ModifyResource", map[string]any{"target": target, "resourceID": resourceID, "amount": amount})
	return nil
}

// SpawnEntity mock：记录调用参数，返回自增实体ID模拟生成成功。
func (m *mockActionAccessor) SpawnEntity(ctx ActionContext, templateID string, posX int32, posY int32, owner string) (int64, error) {
	m.entityIDSeq++
	m.call("SpawnEntity", map[string]any{"templateID": templateID, "posX": posX, "posY": posY, "owner": owner})
	return m.entityIDSeq, nil
}

// TriggerCombat mock：记录调用参数，模拟触发战斗成功。
func (m *mockActionAccessor) TriggerCombat(ctx ActionContext, attackerID string, defenderID string) error {
	m.call("TriggerCombat", map[string]any{"attackerID": attackerID, "defenderID": defenderID})
	return nil
}

// SendNotify mock：记录调用参数，模拟发送通知成功。
func (m *mockActionAccessor) SendNotify(ctx ActionContext, target string, msgID string, params map[string]any) error {
	m.call("SendNotify", map[string]any{"target": target, "msgID": msgID, "params": params})
	return nil
}

// ChangeTerrain mock：记录调用参数，模拟修改地形成功。
func (m *mockActionAccessor) ChangeTerrain(ctx ActionContext, posX int32, posY int32, terrainID string) error {
	m.call("ChangeTerrain", map[string]any{"posX": posX, "posY": posY, "terrainID": terrainID})
	return nil
}

// TestActionRegistry_Register 测试动作注册表的注册路径。
func TestActionRegistry_Register(t *testing.T) {
	registry := NewActionRegistry()
	handler := &applyBuffHandler{}

	t.Run("正常注册", func(t *testing.T) {
		err := registry.Register(handler)
		assert.NoError(t, err)
	})

	t.Run("重复注册报错且错误含类型名", func(t *testing.T) {
		err := registry.Register(handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "apply_buff")
	})

	t.Run("空类型报错", func(t *testing.T) {
		err := registry.Register(&emptyTypeHandler{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "动作类型不能为空")
	})

	t.Run("nil处理器报错", func(t *testing.T) {
		err := registry.Register(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "动作处理器不能为空")
	})
}

// emptyTypeHandler 测试用空类型动作处理器，验证Type()为空时的注册拦截。
type emptyTypeHandler struct{}

// Type 返回空字符串，用于测试空类型注册拦截。
func (h *emptyTypeHandler) Type() string { return "" }

// Execute 正常执行空实现，注册阶段即被拦截不走到执行。
func (h *emptyTypeHandler) Execute(ctx ActionContext) error { return nil }

// TestActionRegistry_Get 测试动作注册表的查询路径。
func TestActionRegistry_Get(t *testing.T) {
	registry := NewActionRegistry()
	require.NoError(t, registry.Register(&applyBuffHandler{}))

	t.Run("已注册的动作类型", func(t *testing.T) {
		handler, err := registry.Get("apply_buff")
		require.NoError(t, err)
		assert.Equal(t, "apply_buff", handler.Type())
	})

	t.Run("未注册的动作类型返回错误", func(t *testing.T) {
		handler, err := registry.Get("not_registered")
		assert.Error(t, err)
		assert.Nil(t, handler)
		assert.Contains(t, err.Error(), "not_registered")
	})
}

// TestActionRegistry_Types 测试动作注册表的类型枚举。
func TestActionRegistry_Types(t *testing.T) {
	registry := NewActionRegistry()
	require.NoError(t, registry.Register(&applyBuffHandler{}))
	require.NoError(t, registry.Register(&modifyResourceHandler{}))

	types := registry.Types()
	assert.Len(t, types, 2)
	assert.Contains(t, types, "apply_buff")
	assert.Contains(t, types, "modify_resource")
}

// TestBuiltinActions_Registered 验证7个内置动作经init()全部注册到全局注册表。
// 与ADR-002验收标准"Types()返回7个"对齐。
func TestBuiltinActions_Registered(t *testing.T) {
	types := GlobalActionRegistry().Types()
	require.Len(t, types, 7, "全局注册表应恰好包含7个内置动作")

	for _, actionType := range builtinActionTypes {
		t.Run(actionType, func(t *testing.T) {
			handler, err := GlobalActionRegistry().Get(actionType)
			require.NoError(t, err)
			assert.Equal(t, actionType, handler.Type())
		})
	}
}

// TestValidateTypes 测试启动全量校验：全部注册通过、缺失返回缺失列表。
func TestValidateTypes(t *testing.T) {
	t.Run("配置引用的动作全部已注册返回空", func(t *testing.T) {
		missing := ValidateTypes(builtinActionTypes)
		assert.Empty(t, missing)
	})

	t.Run("配置引用未注册动作返回缺失列表", func(t *testing.T) {
		missing := ValidateTypes([]string{"apply_buff", "release_pheromone", "unknown_action"})
		assert.Equal(t, []string{"release_pheromone", "unknown_action"}, missing)
	})
}

// TestBuiltinActions_Execute 遍历7个内置动作的Execute路径：
// 用mock Accessor验证参数透传与副作用端口调用，spawn_entity额外断言Output写入。
func TestBuiltinActions_Execute(t *testing.T) {
	t.Run("apply_buff透传buff_id与target", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params:   map[string]any{"buff_id": "buff_1", "target": "player_1"},
			Accessor: mock,
		}
		require.NoError(t, (&applyBuffHandler{}).Execute(ctx))

		call, ok := mock.findCall("ApplyBuff")
		require.True(t, ok, "应调用ApplyBuff端口")
		assert.Equal(t, "player_1", call.args["target"])
		assert.Equal(t, "buff_1", call.args["buffID"])
	})

	t.Run("remove_buff按buff_id走RemoveBuff", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params:   map[string]any{"buff_id": "buff_1", "target": "player_1"},
			Accessor: mock,
		}
		require.NoError(t, (&removeBuffHandler{}).Execute(ctx))

		call, ok := mock.findCall("RemoveBuff")
		require.True(t, ok, "应调用RemoveBuff端口")
		assert.Equal(t, "buff_1", call.args["buffID"])
	})

	t.Run("remove_buff按source_tag走RemoveBuffByTag", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params:   map[string]any{"source_tag": "skill_1", "target": "player_1"},
			Accessor: mock,
		}
		require.NoError(t, (&removeBuffHandler{}).Execute(ctx))

		call, ok := mock.findCall("RemoveBuffByTag")
		require.True(t, ok, "应调用RemoveBuffByTag端口")
		assert.Equal(t, "skill_1", call.args["sourceTag"])
	})

	t.Run("modify_resource透传resource_id与int64金额", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params:   map[string]any{"resource_id": "wood", "amount": int64(100), "target": "player_1"},
			Accessor: mock,
		}
		require.NoError(t, (&modifyResourceHandler{}).Execute(ctx))

		call, ok := mock.findCall("ModifyResource")
		require.True(t, ok, "应调用ModifyResource端口")
		assert.Equal(t, "wood", call.args["resourceID"])
		assert.Equal(t, int64(100), call.args["amount"])
	})

	t.Run("modify_resource兼容JSON解析的float64金额", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params:   map[string]any{"resource_id": "wood", "amount": float64(50), "target": "player_1"},
			Accessor: mock,
		}
		require.NoError(t, (&modifyResourceHandler{}).Execute(ctx))

		call, ok := mock.findCall("ModifyResource")
		require.True(t, ok)
		assert.Equal(t, int64(50), call.args["amount"])
	})

	t.Run("spawn_entity透传模板位置归属并写入Output", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params: map[string]any{
				"template_id": "ant_worker",
				"position":    map[string]any{"x": int32(3), "y": int32(7)},
				"owner":       "player_1",
			},
			Output:   map[string]any{},
			Accessor: mock,
		}
		require.NoError(t, (&spawnEntityHandler{}).Execute(ctx))

		call, ok := mock.findCall("SpawnEntity")
		require.True(t, ok, "应调用SpawnEntity端口")
		assert.Equal(t, "ant_worker", call.args["templateID"])
		assert.Equal(t, int32(3), call.args["posX"])
		assert.Equal(t, int32(7), call.args["posY"])
		assert.Equal(t, "player_1", call.args["owner"])
		assert.Equal(t, int64(1), ctx.Output["entity_id"], "生成的实体ID应写回Output供后续动作使用")
	})

	t.Run("trigger_combat透传攻防双方", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params:   map[string]any{"attacker_id": "entity_1", "defender_id": "entity_2"},
			Accessor: mock,
		}
		require.NoError(t, (&triggerCombatHandler{}).Execute(ctx))

		call, ok := mock.findCall("TriggerCombat")
		require.True(t, ok, "应调用TriggerCombat端口")
		assert.Equal(t, "entity_1", call.args["attackerID"])
		assert.Equal(t, "entity_2", call.args["defenderID"])
	})

	t.Run("send_notify透传消息模板与附带参数", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params: map[string]any{
				"msg_id": "msg_battle_report",
				"target": "player_1",
				"params": map[string]any{"round": 3},
			},
			Accessor: mock,
		}
		require.NoError(t, (&sendNotifyHandler{}).Execute(ctx))

		call, ok := mock.findCall("SendNotify")
		require.True(t, ok, "应调用SendNotify端口")
		assert.Equal(t, "msg_battle_report", call.args["msgID"])
		assert.Equal(t, "player_1", call.args["target"])
		assert.Equal(t, map[string]any{"round": 3}, call.args["params"])
	})

	t.Run("change_terrain透传位置与地形", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{
			Params: map[string]any{
				"position":   map[string]any{"x": int32(5), "y": int32(9)},
				"terrain_id": "grassland",
			},
			Accessor: mock,
		}
		require.NoError(t, (&changeTerrainHandler{}).Execute(ctx))

		call, ok := mock.findCall("ChangeTerrain")
		require.True(t, ok, "应调用ChangeTerrain端口")
		assert.Equal(t, int32(5), call.args["posX"])
		assert.Equal(t, int32(9), call.args["posY"])
		assert.Equal(t, "grassland", call.args["terrainID"])
	})
}

// TestAction_Execute_InvalidParams 参数缺失时返回错误、不panic、不产生副作用调用。
func TestAction_Execute_InvalidParams(t *testing.T) {
	t.Run("apply_buff缺target", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{Params: map[string]any{"buff_id": "buff_1"}, Accessor: mock}
		err := (&applyBuffHandler{}).Execute(ctx)
		assert.Error(t, err)
		assert.Empty(t, mock.calls, "参数缺失时不应调用端口产生副作用")
	})

	t.Run("remove_buff缺buff_id与source_tag", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{Params: map[string]any{"target": "player_1"}, Accessor: mock}
		err := (&removeBuffHandler{}).Execute(ctx)
		assert.Error(t, err)
		assert.Empty(t, mock.calls)
	})

	t.Run("modify_resource缺amount", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{Params: map[string]any{"resource_id": "wood", "target": "player_1"}, Accessor: mock}
		err := (&modifyResourceHandler{}).Execute(ctx)
		assert.Error(t, err)
		assert.Empty(t, mock.calls)
	})

	t.Run("spawn_entity缺position", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{Params: map[string]any{"template_id": "ant_worker"}, Accessor: mock}
		err := (&spawnEntityHandler{}).Execute(ctx)
		assert.Error(t, err)
		assert.Empty(t, mock.calls)
	})

	t.Run("trigger_combat缺defender_id", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{Params: map[string]any{"attacker_id": "entity_1"}, Accessor: mock}
		err := (&triggerCombatHandler{}).Execute(ctx)
		assert.Error(t, err)
		assert.Empty(t, mock.calls)
	})

	t.Run("send_notify缺msg_id", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{Params: map[string]any{"target": "player_1"}, Accessor: mock}
		err := (&sendNotifyHandler{}).Execute(ctx)
		assert.Error(t, err)
		assert.Empty(t, mock.calls)
	})

	t.Run("change_terrain缺terrain_id", func(t *testing.T) {
		mock := &mockActionAccessor{}
		ctx := ActionContext{Params: map[string]any{"position": map[string]any{"x": 1, "y": 2}}, Accessor: mock}
		err := (&changeTerrainHandler{}).Execute(ctx)
		assert.Error(t, err)
		assert.Empty(t, mock.calls)
	})
}

// customTestHandler 模拟题材层新增动作处理器，验证blank import扩展模式下框架零改动。
type customTestHandler struct {
	executed bool // 是否被执行过，供断言
}

// Type 返回动作类型标识"release_pheromone"（题材层示例动作）。
func (h *customTestHandler) Type() string { return "release_pheromone" }

// Execute 记录执行标记，题材层副作用在E16落地，此处仅验证注册与分发链路。
func (h *customTestHandler) Execute(ctx ActionContext) error {
	h.executed = true
	return nil
}

// TestAction_RegisterCustomHandler_BlankImport 模拟题材层blank import扩展：
// 自定义handler仅通过现有Register/Get API注册执行，框架（rule包）零改动。
func TestAction_RegisterCustomHandler_BlankImport(t *testing.T) {
	registry := NewActionRegistry()
	handler := &customTestHandler{}

	require.NoError(t, registry.Register(handler), "自定义动作应能通过现有Register注册")

	got, err := registry.Get("release_pheromone")
	require.NoError(t, err)
	assert.Contains(t, registry.Types(), "release_pheromone")

	ctx := ActionContext{Accessor: &mockActionAccessor{}}
	require.NoError(t, got.Execute(ctx))
	assert.True(t, handler.executed, "自定义动作应能经现有分发链路执行")
}

// TestActionRegistry_ParallelWithRuleRegistry 验证ActionRegistry与现有RuleRegistry并列共存不冲突：
// 两者为独立注册表，互不感知，各自注册与查询互不影响。
func TestActionRegistry_ParallelWithRuleRegistry(t *testing.T) {
	actionRegistry := NewActionRegistry()
	ruleRegistry := NewRuleRegistry()

	require.NoError(t, actionRegistry.Register(&applyBuffHandler{}))
	require.NoError(t, ruleRegistry.Register("combat.damage_formulas", &mockRuleExecutor{result: RuleResult{Success: true}}))

	action, err := actionRegistry.Get("apply_buff")
	require.NoError(t, err)
	assert.Equal(t, "apply_buff", action.Type())

	executor, err := ruleRegistry.Get("combat.damage_formulas")
	require.NoError(t, err)
	assert.NotNil(t, executor)

	// 动作注册表查询不存在的规则级key报错，反之亦然：两类注册表互不污染。
	_, err = actionRegistry.Get("combat.damage_formulas")
	assert.Error(t, err)
	_, err = ruleRegistry.Get("apply_buff")
	assert.Error(t, err)
}
