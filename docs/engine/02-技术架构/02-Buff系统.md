# 技术架构 · 属性修改系统（Buff系统）

> 管理实体属性的临时修改——增益、减益、光环、触发式效果的统一机制。

---

## 核心模块定义

**是什么**：管理实体属性的临时修改——增益、减益、光环、触发式效果。Buff是SLG中"规则如何影响数值"的通用机制。

**为什么是核心模块**：每个SLG都有Buff。无论是英雄技能加攻、地形减防、科技加成、联盟光环，底层都是"在一段时间内修改实体属性"。Buff系统统一了所有临时属性修改的机制。

**配置注入方式**：

| 扩展点 | 用途 | 契约摘要 |
|-----------|------|---------|
| buff_types | Buff类型定义 | 类型ID→{效果列表, 持续方式, 叠加规则}映射 |
| buff_effects | Buff效果定义 | 效果ID→{目标属性, 修改方式, 修改值/公式}映射 |
| buff_stacking_rules | 叠加规则 | 规则ID→{同源处理, 异源处理, 上限}映射 |
| buff_trigger_rules | 触发规则 | 规则ID→{触发事件, 触发条件, 触发效果}映射 |
| buff_aura_rules | 光环规则 | 规则ID→{范围, 目标过滤, 效果}映射 |
| buff_dispel_rules | 驱散规则 | 规则ID→{驱散条件, 驱散范围, 优先级}映射 |

## 扩展点契约

###  buff.buff_types

| 字段 | 值 |
|------|---|
| extension_point_id | buff.buff_types |
| core_module | Buff系统 |
| purpose | 定义所有Buff类型 |
| input_contract | `{ "type": "object", "schema": { "additionalProperties": { "type": "object", "properties": { "buff_id": { "type": "string" }, "name": { "type": "string" }, "category": { "type": "string", "enum": ["buff", "debuff", "neutral"] }, "effects": { "type": "array", "items": { "type": "string", "description": "效果ID，引用buff.buff_effects" } }, "duration_type": { "type": "string", "enum": ["permanent", "timed", "round_based", "condition_based"] }, "duration_value": {}, "stacking_rule": { "type": "string", "description": "叠加规则ID，引用buff.buff_stacking_rules" }, "max_stacks": { "type": "integer", "minimum": 1 }, "removable": { "type": "boolean" }, "icon": { "type": "string" }, "description": { "type": "string" } }, "required": ["buff_id", "name", "category", "effects", "duration_type"] } }, "required": true }` |
| output_contract | `{ "type": "null", "description": "注册成功无返回值" }` |
| validation_rules | `[{"rule_id":"unique_buff_id","rule_type":"custom_check","parameters":{"field":"buff_id","uniqueness":true},"error_message":"Buff ID必须唯一"},{"rule_id":"valid_effects","rule_type":"reference_check","parameters":{"field":"effects","references":"buff.buff_effects"},"error_message":"效果必须引用已定义的Buff效果"}]` |
| default_value | `{}` |
| examples | `[{"buff_id":"forest_cover","name":"森林掩护","category":"buff","effects":["forest_def_bonus","forest_vision_reduction"],"duration_type":"condition_based","duration_value":"entity.on_terrain == 'forest'","stacking_rule":"replace_same_source","max_stacks":1,"removable":true}]` |

###  buff.buff_effects

| 字段 | 值 |
|------|---|
| extension_point_id | buff.buff_effects |
| core_module | Buff系统 |
| purpose | 定义Buff效果（属性修改的具体方式） |
| input_contract | `{ "type": "object", "schema": { "additionalProperties": { "type": "object", "properties": { "effect_id": { "type": "string" }, "name": { "type": "string" }, "target_attribute": { "type": "string" }, "modify_type": { "type": "string", "enum": ["add", "multiply", "override", "clamp_min", "clamp_max"] }, "modify_value": { "type": "string", "description": "修改值，可以是数值或公式" }, "modify_source": { "type": "string", "enum": ["fixed", "caster_attribute", "target_attribute", "formula"] } }, "required": ["effect_id", "name", "target_attribute", "modify_type", "modify_value"] } }, "required": true }` |
| output_contract | `{ "type": "number", "description": "效果应用后的属性修改量" }` |
| validation_rules | `[{"rule_id":"unique_effect_id","rule_type":"custom_check","parameters":{"field":"effect_id","uniqueness":true},"error_message":"效果ID必须唯一"},{"rule_id":"valid_attribute","rule_type":"reference_check","parameters":{"field":"target_attribute","references":"entity.entity_attributes"},"error_message":"目标属性必须引用已定义的属性"}]` |
| default_value | `{}` |
| examples | `[{"effect_id":"forest_def_bonus","name":"森林防御加成","target_attribute":"def","modify_type":"multiply","modify_value":"1.3","modify_source":"fixed"},{"effect_id":"poison_dot","name":"中毒持续伤害","target_attribute":"hp","modify_type":"add","modify_value":"-caster.atk * 0.1","modify_source":"formula"}]` |

###  buff.buff_stacking_rules

| 字段 | 值 |
|------|---|
| extension_point_id | buff.buff_stacking_rules |
| core_module | Buff系统 |
| purpose | 定义Buff叠加规则 |
| input_contract | `{ "type": "object", "schema": { "additionalProperties": { "type": "object", "properties": { "rule_id": { "type": "string" }, "name": { "type": "string" }, "same_source": { "type": "string", "enum": ["replace", "refresh", "stack", "ignore"] }, "diff_source": { "type": "string", "enum": ["stack", "take_stronger", "take_weaker", "ignore"] }, "max_total_stacks": { "type": "integer", "minimum": 1 }, "overflow_behavior": { "type": "string", "enum": ["discard_new", "remove_oldest", "remove_weakest"] } }, "required": ["rule_id", "name", "same_source", "diff_source"] } }, "required": true }` |
| output_contract | `{ "type": "null", "description": "注册成功无返回值" }` |
| validation_rules | `[{"rule_id":"unique_rule_id","rule_type":"custom_check","parameters":{"field":"rule_id","uniqueness":true},"error_message":"规则ID必须唯一"}]` |
| default_value | `{ "default_stacking": { "rule_id": "default_stacking", "name": "默认叠加", "same_source": "replace", "diff_source": "stack" } }` |
| examples | `[{"rule_id":"poison_stacking","name":"中毒叠加","same_source":"refresh","diff_source":"stack","max_total_stacks":5,"overflow_behavior":"remove_oldest"}]` |

###  buff.buff_trigger_rules

| 字段 | 值 |
|------|---|
| extension_point_id | buff.buff_trigger_rules |
| core_module | Buff系统 |
| purpose | 定义触发式Buff规则（如受击时触发反击Buff） |
| input_contract | `{ "type": "object", "schema": { "additionalProperties": { "type": "object", "properties": { "rule_id": { "type": "string" }, "trigger_event": { "type": "string", "description": "触发事件类型" }, "trigger_condition": { "type": "string" }, "apply_buff": { "type": "string", "description": "触发的Buff ID" }, "apply_target": { "type": "string", "enum": ["self", "source", "target", "area"] }, "cooldown": { "type": "number", "minimum": 0 }, "probability": { "type": "number", "minimum": 0, "maximum": 1 } }, "required": ["rule_id", "trigger_event", "apply_buff"] } }, "required": false }` |
| output_contract | `{ "type": "null", "description": "注册成功无返回值" }` |
| validation_rules | `[{"rule_id":"unique_rule_id","rule_type":"custom_check","parameters":{"field":"rule_id","uniqueness":true},"error_message":"规则ID必须唯一"},{"rule_id":"valid_buff","rule_type":"reference_check","parameters":{"field":"apply_buff","references":"buff.buff_types"},"error_message":"触发的Buff必须引用已定义的Buff类型"}]` |
| default_value | `{}` |
| examples | `[{"rule_id":"counter_attack","trigger_event":"combat.damage_received","trigger_condition":"damage > 0","apply_buff":"counter_stance","apply_target":"self","cooldown":3,"probability":0.3}]` |

###  buff.buff_aura_rules

| 字段 | 值 |
|------|---|
| extension_point_id | buff.buff_aura_rules |
| core_module | Buff系统 |
| purpose | 定义光环规则（范围性Buff） |
| input_contract | `{ "type": "object", "schema": { "additionalProperties": { "type": "object", "properties": { "rule_id": { "type": "string" }, "source_buff": { "type": "string" }, "range": { "type": "integer", "minimum": 1 }, "target_filter": { "type": "string", "description": "目标过滤条件" }, "apply_buff": { "type": "string" }, "check_interval_seconds": { "type": "number", "minimum": 1 } }, "required": ["rule_id", "source_buff", "range", "apply_buff"] } }, "required": false }` |
| output_contract | `{ "type": "null", "description": "注册成功无返回值" }` |
| validation_rules | `[{"rule_id":"unique_rule_id","rule_type":"custom_check","parameters":{"field":"rule_id","uniqueness":true},"error_message":"规则ID必须唯一"},{"rule_id":"valid_source_buff","rule_type":"reference_check","parameters":{"field":"source_buff","references":"buff.buff_types"},"error_message":"源Buff必须引用已定义的Buff类型"},{"rule_id":"valid_apply_buff","rule_type":"reference_check","parameters":{"field":"apply_buff","references":"buff.buff_types"},"error_message":"应用Buff必须引用已定义的Buff类型"}]` |
| default_value | `{}` |
| examples | `[{"rule_id":"commander_aura","source_buff":"commander_presence","range":3,"target_filter":"entity.tags.contains('ally')","apply_buff":"morale_boost","check_interval_seconds":5}]` |

###  buff.buff_dispel_rules

| 字段 | 值 |
|------|---|
| extension_point_id | buff.buff_dispel_rules |
| core_module | Buff系统 |
| purpose | 定义驱散规则 |
| input_contract | `{ "type": "object", "schema": { "additionalProperties": { "type": "object", "properties": { "rule_id": { "type": "string" }, "dispel_condition": { "type": "string" }, "dispel_scope": { "type": "string", "enum": ["all", "buff_only", "debuff_only", "by_tag"] }, "dispel_tags": { "type": "array", "items": { "type": "string" } }, "priority": { "type": "string", "enum": ["oldest_first", "weakest_first", "random"] }, "max_dispel_count": { "type": "integer", "minimum": 1 } }, "required": ["rule_id", "dispel_condition", "dispel_scope"] } }, "required": false }` |
| output_contract | `{ "type": "null", "description": "注册成功无返回值" }` |
| validation_rules | `[{"rule_id":"unique_rule_id","rule_type":"custom_check","parameters":{"field":"rule_id","uniqueness":true},"error_message":"规则ID必须唯一"}]` |
| default_value | `{}` |
| examples | `[{"rule_id":"cleanse","dispel_condition":"skill.cast == 'cleanse'","dispel_scope":"debuff_only","priority":"oldest_first","max_dispel_count":2}]` |

## 框架默认行为

- 默认叠加规则：同源替换，异源叠加
- 默认持续时间：按游戏回合/秒
- 默认无光环
- 默认无触发式效果
- Buff挂载触发 `buff.applied` 事件
- Buff移除触发 `buff.removed` 事件
- Buff层数变更触发 `buff.stacks_changed` 事件

## 约束

- Buff效果只修改属性，不可直接修改实体状态
- Buff持续时间到期自动移除
- 光环效果在源实体离开范围后自动移除
- Buff的修改是可逆的（移除Buff后属性恢复）
- 循环依赖检测：Buff A不可依赖Buff B的效果来计算自身效果，如果B同时依赖A

### 边界条件

| 参数 | 默认值 | 可配 | 说明 |
|------|--------|------|------|
| max_buffs_per_entity | 64 | 是 | 单实体Buff数上限，超限按优先级移除最低级 |
| max_stacks_per_buff | 99 | 是 | 单Buff层数上限 |
| buff_evaluation_order | by_priority | 是 | Buff效果计算顺序：by_priority/by_apply_order |
| aura_check_interval_seconds | 5 | 是 | 光环效果检测间隔 |
| buff_tick_interval_ms | 100 | 是 | Buff时间更新粒度（timed型Buff过期检测） |
| max_aura_range | 20 | 是 | 光环最大影响范围（格子数） |
| buff_removal_priority | oldest_first | 是 | 超上限时移除策略：oldest_first/weakest_first/lowest_priority |

## 与其他核心模块的关系

Buff系统**依赖实体系统**，**被多个玩法核心模块使用**：

- **依赖实体系统**：Buff的作用对象是实体，修改的是实体属性（`entity.entity_attributes`）
- **依赖事件总线**：通过事件监听触发式Buff（如 `combat.damage_received` 触发反击）
- **被战斗系统使用**：战斗中的技能效果、克制修正通过Buff实现
- **被经济系统使用**：经济修正（产出/消耗倍率）通过Buff实现
- **被规则系统使用**：规则的动作之一是施加/移除Buff（`apply_buff` / `remove_buff`）
- **被事件总线使用**：Buff的挂载/移除/层数变更发出事件供其他核心模块监听
- **被联盟系统使用**：联盟光环通过Buff实现
- **被赛季系统使用**：赛季加成通过Buff实现
- **被网络同步使用**：Buff状态需要同步到客户端

---
[返回技术架构README](README.md) | [返回框架总览](../README.md)