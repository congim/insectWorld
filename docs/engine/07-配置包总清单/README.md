# 配置包总清单

> 以下是框架要求的全部配置文件清单。每个文件对应一个或多个扩展点。

## 配置文件列表

| 配置文件 | 用途 | 必需/可选 | 主要扩展点 | 依赖 |
|---------|------|----------|-----------|------|
| **game.json** | 游戏全局配置（名称、版本、格子类型、地图尺寸等） | 必需 | map.map_grid_type, map.map_dimensions | 无 |
| **factions.json** | 种族/阵营定义 | 可选 | entity.entity_types, entity.entity_tags | game.json |
| **units.json** | 兵种/单位定义 | 必需 | entity.entity_types, entity.entity_attributes, entity.entity_tags | factions.json, entity.entity_components |
| **buildings.json** | 建筑定义 | 必需 | entity.entity_types, entity.entity_attributes | factions.json |
| **heroes.json** | 英雄定义 | 可选 | entity.entity_types, entity.entity_attributes, combat.combat_skills | factions.json |
| **techs.json** | 科技定义 | 可选 | rule.rule_definitions, buff.buff_types | units.json, buildings.json |
| **resources.json** | 资源类型定义 | 必需 | economy.resource_types | 无 |
| **terrains.json** | 地形定义 | 必需 | map.terrains, map.map_passability_rules, map.map_vision_rules | game.json |
| **counter_matrix.json** | 克制关系矩阵 | 可选 | combat.counter_matrix | units.json |
| **formulas.json** | 公式定义（伤害、经验、生产等） | 必需 | combat.damage_formulas, economy.production_rules, economy.consumption_rules | resources.json, units.json |
| **buffs.json** | Buff定义 | 可选 | buff.buff_types, buff.buff_effects, buff.buff_stacking_rules | entity.entity_attributes |
| **rules.json** | 规则定义 | 可选 | rule.rule_definitions, rule.rule_groups, rule.rule_conditions, rule.rule_actions | 所有其他配置 |
| **events.json** | 事件定义 | 可选 | event.event_types, event.event_filters, event.event_transformers | rules.json |
| **season.json** | 赛季定义 | 可选 | season.season_phases, season.season_transition_rules, season.season_reset_rules, season.season_inherit_rules, season.season_rewards, season.season_scoring_rules | rules.json, resources.json |
| **alliance.json** | 联盟定义 | 可选 | alliance.alliance_ranks, alliance.alliance_permissions, alliance.alliance_diplomacy_states | rules.json, resources.json |
| **movement.json** | 移动定义 | 可选 | movement.movement_types, movement.movement_costs, movement.movement_blocking_rules | terrains.json, units.json |
| **combat.json** | 战斗定义 | 可选 | combat.combat_types, combat.combat_phases, combat.combat_skills, combat.combat_formation_effects, combat.combat_loot_rules | units.json, counter_matrix.json, formulas.json |
| **economy.json** | 经济定义 | 可选 | economy.production_rules, economy.collection_rules, economy.trade_rules, economy.consumption_rules, economy.conversion_rules, economy.storage_rules | resources.json, units.json, buildings.json |
| **sync.json** | 网络同步定义 | 可选 | sync.sync_scope_rules, sync.sync_priority_rules, sync.reconnection_rules, sync.sync_validation_rules | units.json |
| **persistence.json** | 持久化定义 | 可选 | persistence.persistence_entities, persistence.persistence_triggers, persistence.snapshot_rules, persistence.archive_rules | units.json |

## 加载顺序

按依赖关系拓扑排序，加载顺序如下：

1. **L0 基础层**：`game.json` → `resources.json`（无依赖）
2. **L1 实体层**：`factions.json` → `units.json` / `buildings.json` / `heroes.json`（依赖 game.json）
3. **L2 地形层**：`terrains.json`（依赖 game.json）
4. **L3 公式层**：`counter_matrix.json` → `formulas.json`（依赖 units.json、resources.json）
5. **L4 规则层**：`techs.json` → `buffs.json` → `rules.json` → `events.json`（依赖实体、公式等）
6. **L5 运营层**：`season.json` / `alliance.json` / `movement.json` / `combat.json` / `economy.json` / `sync.json` / `persistence.json`（依赖前面所有层）

## 依赖关系图

```
game.json ─────────────────┬──→ factions.json ──→ units.json ──┐
                            │                     ├──→ buildings.json
                            │                     └──→ heroes.json
                            │
                            └──→ terrains.json ──→ movement.json
                            
resources.json ──┬──→ formulas.json ──→ combat.json
                 │                   └──→ economy.json
                 ├──→ season.json
                 └──→ alliance.json

units.json ──→ counter_matrix.json ──→ combat.json
           └──→ techs.json
           
buffs.json ──→ rules.json ──→ events.json
                            ├──→ season.json
                            └──→ alliance.json
```

---

## 子文件索引

- [全局配置.md](全局配置.md) - game.json 的完整Schema定义
- [实体配置.md](实体配置.md) - factions/units/buildings/heroes/techs 的Schema定义
- [规则配置.md](规则配置.md) - rules/events/buffs/formulas/counter_matrix 的Schema定义
- [运营配置.md](运营配置.md) - season/alliance/resources/terrains/movement/combat/economy 的Schema定义

[← 返回总入口](../README.md)