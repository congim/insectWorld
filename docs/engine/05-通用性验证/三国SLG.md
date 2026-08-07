# 通用性验证 · 三国SLG

## 4.3 题材B：三国SLG

### 核心玩法特征

| # | 特征 | 描述 |
|---|------|------|
| 1 | 州郡争夺 | 地图按州郡划分，占领州郡获得资源和战略优势 |
| 2 | 武将系统 | 武将有独特技能、羁绊、忠诚度，可被策反 |
| 3 | 兵种克制 | 步兵、骑兵、弓兵、攻城器之间有明确的克制关系 |
| 4 | 内政建设 | 城池建设农田、市场、兵营等，不同建筑提供不同功能 |
| 5 | 外交合纵 | 联盟间可结盟、互不侵犯、宣战，外交影响战争走向 |
| 6 | 攻城战 | 攻城有独特的战斗流程：攻城准备→攻城→破城 |
| 7 | 科技树 | 内政科技和军事科技两条线，影响发展和战斗力 |
| 8 | 赛季争霸 | 赛季末争夺洛阳，决定赛季排名和奖励 |
| 9 | 名城争夺 | 特定名城（洛阳、长安等）有特殊加成，是争夺焦点 |

### 映射到框架核心模块

| 特征 | 核心模块 | 机制支撑 |
|------|------|---------|
| 州郡争夺 | 地图系统 | map_regions定义州郡区域 |
| 州郡争夺 | 规则系统 | 占领州郡的规则和奖励 |
| 武将系统 | 实体系统 | 武将是hero类型实体，有独特属性和技能 |
| 武将系统 | Buff系统 | 羁绊通过Buff实现，忠诚度通过属性+规则实现 |
| 武将系统 | 规则系统 | 策反规则：条件满足时武将叛变 |
| 兵种克制 | 战斗系统 | counter_matrix定义克制关系 |
| 内政建设 | 实体系统 | 建筑实体，不同类型有不同功能 |
| 内政建设 | 经济系统 | 建筑产出资源，production_rules |
| 外交合纵 | 联盟系统 | alliance_diplomacy_states定义同盟、互不侵犯、敌对 |
| 外交合纵 | 规则系统 | 外交状态对战争的影响规则 |
| 攻城战 | 战斗系统 | combat_types定义攻城战，combat_phases定义攻城阶段 |
| 科技树 | 规则系统 | 科技解锁规则，前置条件检查 |
| 科技树 | Buff系统 | 科技效果通过Buff实现 |
| 赛季争霸 | 赛季系统 | season_phases定义争霸期，season_scoring_rules定义积分 |
| 赛季争霸 | 规则系统 | 争夺洛阳的规则和奖励 |
| 名城争夺 | 地图系统 | 名城是特殊区域，有区域属性 |
| 名城争夺 | Buff系统 | 占领名城的加成通过Buff实现 |

### 映射到框架内容配置

| 特征 | 内容配置（配置） | 扩展点 |
|------|-----------|--------|
| 州郡争夺 | terrains.json: 州郡区域定义 | map.map_regions |
| 州郡争夺 | rules.json: 占领规则 | rule.rule_definitions |
| 武将系统 | heroes.json: 武将定义 | entity.entity_types |
| 武将系统 | buffs.json: 羁绊Buff | buff.buff_types, buff.buff_aura_rules |
| 武将系统 | rules.json: 策反规则 | rule.rule_definitions |
| 兵种克制 | counter_matrix.json: 步/骑/弓/攻城克制 | combat.counter_matrix |
| 内政建设 | buildings.json: 城池建筑 | entity.entity_types |
| 内政建设 | economy.json: 建筑产出 | economy.production_rules |
| 外交合纵 | alliance.json: 外交状态 | alliance.alliance_diplomacy_states |
| 攻城战 | combat.json: 攻城战类型和阶段 | combat.combat_types, combat.combat_phases |
| 科技树 | techs.json: 科技定义 | rule.rule_definitions, buff.buff_types |
| 赛季争霸 | season.json: 争霸期和积分 | season.season_phases, season.season_scoring_rules |
| 名城争夺 | terrains.json: 名城区域 | map.map_regions |
| 名城争夺 | buffs.json: 名城加成 | buff.buff_types |

---

[← 返回通用性验证](README.md) | [← 返回总入口](../README.md)