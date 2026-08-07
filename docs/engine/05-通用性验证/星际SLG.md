# 通用性验证 · 星际SLG

## 4.4 题材C：星际SLG

### 核心玩法特征

| # | 特征 | 描述 |
|---|------|------|
| 1 | 星系地图 | 三维星系地图，星系间通过跃迁通道连接 |
| 2 | 舰队编队 | 舰队由多艘舰船组成，不同舰船类型有不同角色 |
| 3 | 三层战斗 | 太空战→轨道战→地面战，三层战斗逐层推进 |
| 4 | 资源星球 | 不同星球产出不同资源，需要殖民和开发 |
| 5 | 科技跃迁 | 科技树分支众多，高级科技需要稀有资源 |
| 6 | 星际外交 | 联盟间可签订互不侵犯、贸易协定、军事同盟 |
| 7 | 跃迁移动 | 舰队通过跃迁通道或超空间引擎在星系间移动 |
| 8 | 赛季银河战争 | 赛季末争夺银河核心，决定银河霸权 |

### 映射到框架核心模块

| 特征 | 核心模块 | 机制支撑 |
|------|------|---------|
| 星系地图 | 地图系统 | map_grid_type=free（自由坐标），map_regions定义星系 |
| 星系地图 | 地图系统 | 跃迁通道通过map_passability_rules定义（只有特定路径可通行） |
| 舰队编队 | 实体系统 | 舰船是unit实体，舰队是编队实体（包含子实体组件） |
| 舰队编队 | 移动系统 | movement_formation_rules定义舰队编队移动 |
| **三层战斗** | 战斗系统 | **关键验证点**：combat_types定义三种战斗类型，combat_phases定义三层战斗的阶段流转 |
| **三层战斗** | 规则系统 | 规则定义：太空战失败→不可进入轨道战；轨道战失败→不可进入地面战 |
| **三层战斗** | 事件系统 | 每层战斗结束发布事件，触发下一层战斗或最终结算 |
| 资源星球 | 经济系统 | production_rules定义星球产出，collection_rules定义采集 |
| 资源星球 | 实体系统 | 星球是resource_node实体 |
| 科技跃迁 | 规则系统 | 科技解锁规则，前置条件包含稀有资源检查 |
| 科技跃迁 | 经济系统 | 稀有资源消耗 |
| 星际外交 | 联盟系统 | alliance_diplomacy_states定义贸易协定、军事同盟等 |
| 跃迁移动 | 移动系统 | movement_teleport_rules定义跃迁规则，movement_types定义超空间移动 |
| 跃迁移动 | 地图系统 | 跃迁通道通过通行性规则定义 |
| 赛季银河战争 | 赛季系统 | season_phases定义银河战争期 |
| 赛季银河战争 | 规则系统 | 争夺银河核心的规则 |

### 三层战斗的详细映射（重点验证）

三层战斗是星际SLG最独特的玩法，需要特别验证框架能否覆盖：

**第一层：太空战**

- 战斗类型：`space_battle`（combat_types）
- 参与者：舰队 vs 舰队
- 阶段：探测→远程交火→近距交火→结算（combat_phases）
- 胜利条件：一方舰队全灭或撤退
- 失败后果：不可进入轨道层

**第二层：轨道战**

- 战斗类型：`orbital_battle`（combat_types）
- 参与者：获胜舰队 vs 轨道防御
- 阶段：突破防御→轨道轰炸→空降准备→结算（combat_phases）
- 胜利条件：轨道防御被摧毁
- 失败后果：不可进入地面层

**第三层：地面战**

- 战斗类型：`ground_battle`（combat_types）
- 参与者：空降部队 vs 地面驻军
- 阶段：空降→建立据点→推进→占领→结算（combat_phases）
- 胜利条件：占领关键设施

**层间流转**：通过规则系统+事件系统实现

- 太空战结束→发布`combat.ended(space_battle)`事件→规则检查结果→如果胜利，触发轨道战
- 轨道战结束→发布`combat.ended(orbital_battle)`事件→规则检查结果→如果胜利，触发地面战
- 地面战结束→发布`combat.ended(ground_battle)`事件→最终结算

**验证结论**：三层战斗完全可以通过战斗系统（三种combat_types + 对应combat_phases）+ 规则系统（层间流转规则）+ 事件系统（层间事件触发）实现，**无需修改框架**。

### 映射到框架内容配置

| 特征 | 内容配置（配置） | 扩展点 |
|------|-----------|--------|
| 星系地图 | game.json: free格子类型 | map.map_grid_type |
| 星系地图 | terrains.json: 星系区域和跃迁通道 | map.map_regions, map.map_passability_rules |
| 舰队编队 | units.json: 舰船定义 | entity.entity_types |
| 舰队编队 | movement.json: 舰队编队规则 | movement.movement_formation_rules |
| 三层战斗 | combat.json: 三种战斗类型和阶段 | combat.combat_types, combat.combat_phases |
| 三层战斗 | rules.json: 层间流转规则 | rule.rule_definitions |
| 三层战斗 | events.json: 层间事件 | event.event_types |
| 资源星球 | resources.json: 星际资源 | economy.resource_types |
| 资源星球 | economy.json: 星球产出 | economy.production_rules |
| 科技跃迁 | techs.json: 星际科技 | rule.rule_definitions |
| 星际外交 | alliance.json: 星际外交状态 | alliance.alliance_diplomacy_states |
| 跃迁移动 | movement.json: 跃迁规则 | movement.movement_teleport_rules |
| 赛季银河战争 | season.json: 银河战争期 | season.season_phases, season.season_scoring_rules |

---

[← 返回通用性验证](README.md) | [← 返回总入口](../README.md)