# 通用性验证 · 昆虫世界SLG

## 4.2 题材A：昆虫世界SLG

### 核心玩法特征

| # | 特征 | 描述 |
|---|------|------|
| 1 | 蚁群分工 | 工蚁采集、兵蚁战斗、蚁后产卵——不同角色有完全不同的行为模式 |
| 2 | 信息素路径 | 蚂蚁通过信息素标记路径，其他蚂蚁会跟随信息素移动 |
| 3 | 虫族进化 | 通过消耗资源进化，解锁新兵种和新能力，进化不可逆 |
| 4 | 季节性蛰伏 | 冬季蚂蚁活动大幅减少，春季苏醒，影响经济和军事 |
| 5 | 巢穴建设 | 地下巢穴的多层结构，不同房间有不同功能 |
| 6 | 菌圃共生 | 蚂蚁培养真菌作为食物来源，形成独特的经济循环 |
| 7 | 异族入侵 | 白蚁、甲虫等异族入侵，形成PvE挑战 |

### 映射到框架核心模块

| 特征 | 核心模块 | 机制支撑 |
|------|------|---------|
| 蚁群分工 | 实体系统 | 不同实体类型（工蚁/兵蚁/蚁后）通过entity_types定义，不同组件组合实现不同行为 |
| 蚁群分工 | 经济系统 | 工蚁通过collection_rules采集，兵蚁不采集 |
| 信息素路径 | 移动系统 | movement_path_modifiers实现信息素路径偏好，movement_types定义信息素跟随移动 |
| 信息素路径 | 事件系统 | 蚂蚁移动时发布事件，规则系统在路径上添加信息素标记 |
| 虫族进化 | 规则系统 | 进化条件-动作规则，消耗资源→解锁新兵种/能力 |
| 虫族进化 | 经济系统 | 进化消耗资源，通过consumption_rules实现 |
| 虫族进化 | Buff系统 | 进化后的属性加成通过Buff实现 |
| 季节性蛰伏 | 赛季系统 | season_phases定义蛰伏期，phase_effects限制活动 |
| 季节性蛰伏 | 规则系统 | 蛰伏期的生产/移动限制通过规则实现 |
| 巢穴建设 | 实体系统 | 巢穴房间是建筑实体，通过entity_types定义 |
| 巢穴建设 | 地图系统 | 地下层通过map_regions定义，不同区域有不同功能 |
| 菌圃共生 | 经济系统 | conversion_rules实现"食物→真菌→更多食物"循环 |
| 菌圃共生 | 生产规则 | production_rules定义真菌产出 |
| 异族入侵 | 战斗系统 | combat_types定义PvE战斗类型 |
| 异族入侵 | 事件系统 | 定时事件触发异族入侵 |

### 映射到框架内容配置

| 特征 | 内容配置（配置） | 扩展点 |
|------|-----------|--------|
| 蚁群分工 | units.json: 工蚁、兵蚁、蚁后实体定义 | entity.entity_types |
| 蚁群分工 | economy.json: 工蚁采集规则 | economy.collection_rules |
| 信息素路径 | movement.json: 信息素移动类型和路径修正 | movement.movement_path_modifiers, movement.movement_types |
| 信息素路径 | events.json: 移动事件→信息素标记 | event.event_types, rule.rule_definitions |
| 虫族进化 | rules.json: 进化规则 | rule.rule_definitions |
| 虫族进化 | resources.json: 进化资源 | economy.resource_types |
| 虫族进化 | buffs.json: 进化加成 | buff.buff_types, buff.buff_effects |
| 季节性蛰伏 | season.json: 蛰伏期定义 | season.season_phases, season.season_transition_rules |
| 季节性蛰伏 | rules.json: 蛰伏限制规则 | rule.rule_definitions |
| 巢穴建设 | buildings.json: 巢穴房间定义 | entity.entity_types |
| 巡穴建设 | terrains.json: 地下地形 | map.terrains, map.map_regions |
| 菌圃共生 | economy.json: 真菌生产/转换规则 | economy.production_rules, economy.conversion_rules |
| 异族入侵 | combat.json: PvE战斗类型 | combat.combat_types |
| 异族入侵 | events.json: 入侵事件 | event.event_types |

---

[← 返回通用性验证](README.md) | [← 返回总入口](../README.md)