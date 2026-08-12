# 通用SLG框架设计书

> 核心理念：**骨肉分离**。骨头是框架提供的通用机制，肉是游戏配置挂载的具体内容。换皮=换肉，骨头不动。
>
> 本文档不是实现规格书，不写Go代码、不写DDL、不写proto定义。写的是**框架契约**——每根骨头是什么、肉怎么挂上去、挂载点的契约怎么定义。

---

## 第一章 框架总览

### 1.1 什么是"骨肉分离"

骨肉分离是本框架的核心架构思想，用一张图说清楚：

```
┌─────────────────────────────────────────────────────────────┐
│                        游戏运行时                            │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                     骨 头 层                          │  │
│  │                                                       │  │
│  │  实体系统 │ 地图系统 │ 移动系统 │ 战斗系统 │ 经济系统  │  │
│  │  Buff系统 │ 规则系统 │ 联盟系统 │ 赛季系统 │ 事件系统  │  │
│  │  网络同步骨架 │ 持久化骨架                            │  │
│  │                                                       │  │
│  │  → 框架代码，不包含任何游戏内容字符串                  │  │
│  └───────────────────────────────────────────────────────┘  │
│                          ▲                                  │
│                          │ 骨肉接口（挂载点 × 契约）        │
│                          ▼                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                      肉 层                            │  │
│  │                                                       │  │
│  │  种族 │ 兵种 │ 建筑 │ 英雄 │ 技能 │ 公式 │ 克制矩阵   │  │
│  │  资源类型 │ 地形 │ 事件 │ 赛季规则 │ 联盟规则 │ ……    │  │
│  │                                                       │  │
│  │  → 配置文件，按契约格式填写，框架校验后加载             │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

- **骨头**：框架提供的通用机制（实体生命周期、地图空间管理、移动计算、战斗流程、经济流转、Buff挂载、规则触发、联盟状态机、赛季循环、网络同步骨架、持久化骨架……）。骨头代码里不出现任何游戏内容字符串——没有"步兵"、没有"粮食"、没有"城墙"。
- **肉**：游戏配置的具体内容（种族、兵种、建筑、英雄、技能、公式、克制矩阵、资源类型、地形、事件……）。肉是数据，不是代码。肉按契约格式填写，框架校验后加载。
- **骨肉接口**：每根骨头暴露的挂载点，肉必须按契约实现才能挂上去。契约定义了输入格式、输出保证、约束条件。违反契约的肉，框架拒绝加载。

**换皮 = 换肉，骨头不动。** 同一套框架，加载三国配置包就是三国SLG，加载星际配置包就是星际SLG，加载丧尸配置包就是丧尸SLG。骨头代码一行不改。

### 1.2 框架设计原则

| 原则 | 含义 | 反例（违反此原则） |
|------|------|-------------------|
| **机制通用，内容可配** | 框架只管"怎么做"，不管"是什么" | 框架代码里写死"步兵克制骑兵" |
| **挂载点显式** | 每个可扩展的点都要明确定义契约（输入是什么、输出是什么、约束是什么） | 框架内部硬编码一组魔法字符串，配置靠猜 |
| **零硬编码** | 框架代码里不出现任何游戏内容字符串 | 框架代码里出现"粮食"、"城墙"、"联盟"等游戏术语 |
| **配置即玩法** | 通过不同的配置组合，可以产生完全不同的游戏体验 | 想改一个克制关系必须改框架代码 |
| **最小惊讶** | 框架的默认行为应该是"最合理的通用行为"，特殊行为由配置覆盖 | 不配置任何东西时，框架的行为让人困惑 |

### 1.3 框架能力边界

**框架管的（骨头职责）：**

| 领域 | 框架管什么 |
|------|-----------|
| 实体生命周期 | 创建、销毁、查询、索引、组件挂载 |
| 空间管理 | 地图分块、坐标转换、范围查询 |
| 移动计算 | 寻路、插值、拦截检测、路径缓存 |
| 战斗流程 | 战斗状态机、阶段推进、结果演算、日志记录 |
| 资源流转 | 产出/消耗tick计算、溢出/饥荒处理、掠夺计算 |
| Buff系统 | 挂载、移除、叠加、驱散、属性计算顺序 |
| 规则引擎 | 事件监听、条件求值、动作执行、冷却管理、循环检测 |
| 联盟状态机 | CRUD、成员管理、外交状态转换、科技贡献 |
| 赛季循环 | 阶段推进、数据重置、遗产保留、排名计算 |
| 网络同步骨架 | Command校验、DeltaUpdate广播、FullSnapshot、断线重连 |
| 持久化骨架 | 脏标记、批量写入、实时写入、宕机恢复 |

**框架不管的（肉职责）：**

| 领域 | 具体内容 |
|------|---------|
| 具体数值 | 兵种攻防、建筑血量、科技加成数值 |
| 具体公式 | 伤害公式、产出公式、消耗公式的具体形式 |
| 具体技能效果 | 技能做什么、怎么表现 |
| 具体美术 | 模型、特效、动画 |
| 具体UI布局 | 界面排版、交互流程 |
| 具体世界观 | 种族设定、剧情文本、背景故事 |

### 1.4 换皮流程

从"拿到框架"到"做出一个新SLG"，只需要做以下步骤：

```
┌──────────────┐
│  1. 准备配置包  │  种族/兵种/建筑/英雄/技能/公式/克制矩阵/
│                │  资源/地形/事件/赛季/联盟/移动/战斗/经济
└──────┬───────┘
       ▼
┌──────────────┐
│  2. 准备美术   │  模型/特效/UI/地图贴图/图标
│     资源       │
└──────┬───────┘
       ▼
┌──────────────┐
│  3. 准备本地化 │  多语言文本/命名/描述/提示
│     文本       │
└──────┬───────┘
       ▼
┌──────────────┐
│  4. 按框架契约 │  校验每个配置文件是否符合对应挂载点的契约
│     校验配置包 │  不符合→报错→修改→重新校验
└──────┬───────┘
       ▼
┌──────────────┐
│  5. 启动框架   │  框架加载配置包→初始化各系统→进入运行
│     加载配置包 │
└──────┬───────┘
       ▼
┌──────────────┐
│  6. 跑通       │  跑通核心流程→调数值→测边界→上线
│     调数值     │
└──────────────┘
```

**关键点：步骤1-3是"做肉"，步骤4是"验肉"，步骤5-6是"跑肉"。整个过程不碰骨头代码。**

---

## 第二章 骨头清单（框架提供的通用机制）

### 2.1 实体系统

| 项目 | 说明 |
|------|------|
| **是什么** | 游戏世界中一切"东西"的抽象——城市、军队、建筑、兵种、地块、资源点、英雄……都是实体 |
| **为什么是骨头** | 每个SLG都有大量实体，它们的创建/销毁/查询/索引/组件管理是通用需求，不可能每个游戏重写一遍 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 框架提供Entity基类、组件注册/查询机制、全局唯一ID生成、内存索引（按类型/按位置/按所属玩家） |
| **约束** | 每种EntityType必须有id字段；组件必须可序列化；实体ID全局唯一且不可复用 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 实体类型定义 | 定义有哪些类型的实体、每种类型挂载哪些组件、组件的初始值模板 |
| 组件定义 | 定义组件的字段名和类型、字段的默认值、字段的约束（范围/枚举） |
| 实体模板 | 定义同类型实体的不同模板（如"1级兵营"和"5级兵营"是同类型不同模板） |

---

### 2.2 地图系统

| 项目 | 说明 |
|------|------|
| **是什么** | 游戏空间的组织方式——网格坐标、地形、分块、视野、迷雾 |
| **为什么是骨头** | 每个SLG都有地图，地图上的空间关系是移动/战斗/采集/领土的基础设施 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 二维四方网格、A*寻路、Chunk分块加载、视野裁剪、无迷雾 |
| **约束** | 地图必须是网格（支持六角格/四方格配置切换）；地形类型必须定义通行性矩阵；地图尺寸必须>0 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 地图规格 | 宽/高/格类型(四方格/六角格)/分块大小 |
| 地形类型定义 | 地形名称、通行性（按移动类型）、修正值（防御/移动/产出）、视觉效果ID |
| 地形生成规则 | 权重分布/区域规则/特殊点（如固定出生点、固定资源点） |
| 迷雾规则 | 可见/已探索/未探索的判定条件、探索半径、联盟共享视野规则 |

---

### 2.3 移动系统

| 项目 | 说明 |
|------|------|
| **是什么** | 实体在地图上移动的机制——寻路、行军、拦截、驻守 |
| **为什么是骨头** | 军队行军、采集队移动、贸易队运输……每个SLG都有移动需求，寻路算法和拦截检测是通用逻辑 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | A*寻路、线性插值渲染位置、拦截检测（两军进入触发距离则触发拦截事件）、路径缓存 |
| **约束** | 移动速度必须>0；拦截规则必须定义触发距离；移动类型必须在地形通行性矩阵中有定义 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 移动类型定义 | 地面/飞行/地下/水上……每种类型对各地形的通行性不同（布尔矩阵） |
| 速度修正来源 | 种族修正/科技修正/地形修正/天气修正/Buff修正——修正值的叠加规则由框架定，具体数值由配置定 |
| 移动状态机扩展 | 框架默认提供：待命/行军/战斗/撤退/驻守/巡逻，游戏可扩展新状态和转换条件 |
| 拦截规则 | 什么条件下两军交战（敌对关系+触发距离+移动方向） |

---

### 2.4 战斗系统

| 项目 | 说明 |
|------|------|
| **是什么** | 冲突解决的机制——从两军接触到战斗结束的完整流程 |
| **为什么是骨头** | 每个SLG都有PvP/PvE战斗，战斗状态机、阶段推进、结果演算是通用逻辑 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 战斗状态机、tick制演算（每tick按公式计算伤害）、战斗日志记录（用于回放和战报） |
| **约束** | 伤害公式必须返回非负数；克制矩阵必须是对角为1.0的方阵；战斗必须有结束条件 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 伤害公式 | formulas.json定义，框架提供公式引擎（支持变量替换和四则运算），具体公式由配置定 |
| 克制矩阵 | counter_matrix.json定义，N×N任意维度方阵，元素值=克制倍率 |
| 战斗阶段定义 | 框架默认提供"接近/远程/近战/结束"，游戏可自定义阶段和阶段转换条件 |
| 战斗触发条件 | 什么情况下两军开打（拦截/主动攻击/攻城/伏击） |
| 战斗结束条件 | 什么情况下战斗结束（一方血量阈值/时间限制/全灭/撤退成功） |
| 战斗结果处理 | 胜方/败方/平局各发生什么（追击/撤退/掠夺/经验/俘虏） |
| 攻城规则 | 城墙怎么打（城墙血量/攻击方修正/防御方修正）、城墙破了怎样（城内战斗/掠夺） |

---

### 2.5 经济系统

| 项目 | 说明 |
|------|------|
| **是什么** | 资源在游戏中流转的机制——产出、消耗、采集、贸易、掠夺 |
| **为什么是骨头** | 每个SLG都有资源产出/消耗/贸易/掠夺，流转逻辑和溢出/饥荒处理是通用需求 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 资源产出/消耗的tick计算、溢出丢弃、负资源触发饥荒事件 |
| **约束** | 资源类型至少1种；产出和消耗公式必须返回非负数；掠夺比例必须在[0,1]区间 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 资源类型定义 | 名称、是否可贸易、是否可掠夺、掠夺比例、存储上限公式、图标ID |
| 产出来源定义 | 建筑产出/地块产出/Buff产出/事件产出——每种来源的公式和触发条件 |
| 消耗定义 | 人口消耗/军队维持/建筑维护/科技研究——每种消耗的公式和扣减时机 |
| 采集规则 | 采集队配置/载重/采集速度/伏击规则 |
| 贸易规则 | 贸易条件/税率/运输时间/贸易路线 |
| 掠夺规则 | 掠夺比例/保护机制（保护量/保护比例/保护条件） |
| 饥荒/溢出规则 | 资源不足时怎么办（惩罚/限制/警告）、资源超上限时怎么办（丢弃/停止产出/转化） |

---

### 2.6 属性修改系统（Buff系统）

| 项目 | 说明 |
|------|------|
| **是什么** | 实体属性被临时/永久修改的机制——所有"加成"和"减益"的统一入口 |
| **为什么是骨头** | 种族加成、科技加成、装备加成、技能效果、地形修正、天气减益……全走这个系统，不可能每个游戏各写一套 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 固定计算顺序（基础→固定加成→乘百分比→技能固定加成→钳制）、Buff到期自动移除、来源标记驱散 |
| **约束** | 属性计算顺序**不可配置**（防止不同游戏算出不同结果导致混乱）；Buff效果必须可序列化；叠加规则必须从框架预定义的三种中选择 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| Buff模板定义 | 名称、效果列表（属性名+操作类型+值）、持续时间、叠加规则、来源标记、驱散条件 |
| 属性计算顺序 | **不可配置**，但框架文档明确公布顺序：①基础值 → ②固定加成求和 → ③乘百分比求积 → ④技能固定加成求和 → ⑤钳制（min/max） |
| 叠加规则 | 同ID Buff的叠加方式：replace(替换)/stack(叠加)/refresh(刷新持续时间)——由Buff模板指定 |
| 驱散规则 | 哪些来源标记的Buff可被驱散、驱散优先级——由Buff模板指定 |

---

### 2.7 规则系统

| 项目 | 说明 |
|------|------|
| **是什么** | "当X发生时，做Y"的通用触发-动作机制 |
| **为什么是骨头** | 伏击效果、围城效果、赛季阶段效果、事件响应、成就触发……全是"触发-动作"，统一走规则引擎 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 事件触发→条件检查→执行动作→冷却计时 |
| **约束** | 规则不能循环触发（框架做深度检测，超过阈值则报错）；动作类型可扩展（内置7个直接可用，新类型由题材层init()注册，框架代码零改动，见ADR-002/ADR-003） |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 规则定义 | 触发事件名、条件表达式、动作类型、动作参数、冷却时间、是否一次性 |
| 动作类型 | 内置动作类型（7个）：apply_buff / remove_buff / modify_resource / spawn_entity / trigger_combat / send_notify / change_terrain。新动作类型=题材层实现RuleActionHandler并经注册表init()注册，框架代码零改动（见ADR-002/ADR-003） |
| 条件表达式 | 简单比较表达式（支持>/>=/</<=/==/!=/in/and/or），框架提供求值器 |

---

### 2.8 联盟系统

| 项目 | 说明 |
|------|------|
| **是什么** | 玩家组织和社会关系的机制——联盟、外交、联盟科技、联盟战争 |
| **为什么是骨头** | 每个SLG都有联盟/公会/帮派，成员管理、外交状态机、联盟科技是通用逻辑 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 联盟CRUD、成员管理（加入/退出/踢出/转让盟主）、外交状态机、联盟科技贡献 |
| **约束** | 联盟必须有盟主；外交状态转换必须定义合法路径（不能从"同盟"直接跳到"敌对"而不经过"中立"） |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 联盟配置 | 成员上限、等级体系、科技列表、领土规则（占领/共享/争议） |
| 外交状态定义 | 默认5种：敌对/中立/友好/同盟/互不侵犯，游戏可增减，但必须定义合法转换路径 |
| 联盟科技定义 | 科技列表、每项科技的效果、贡献方式、解锁条件 |
| 联盟战争规则 | 宣战条件/计分规则/奖励分配/战争持续时间 |

---

### 2.9 赛季系统

| 项目 | 说明 |
|------|------|
| **是什么** | 游戏周期性重置的机制——赛季阶段推进、数据重置、遗产保留 |
| **为什么是骨头** | SLG的长线运营依赖赛季循环，阶段状态机和数据重置逻辑是通用的 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 赛季状态机、阶段自动切换、数据重置、遗产保留、排名计算 |
| **约束** | 赛季必须有结束条件；重置和遗产规则必须保证数据一致性（不能重置了联盟但保留联盟科技） |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 赛季配置 | 总时长、阶段列表、阶段效果、重置规则、遗产规则、胜利条件、段位体系 |
| 阶段定义 | 每个阶段的时间范围、阶段Buff、阶段限制（如禁止宣战/禁止迁城）、阶段转换条件 |
| 重置规则 | 哪些数据重置、重置方式（清零/归默认/删除）、重置顺序 |
| 遗产规则 | 哪些数据保留、保留比例/保留条件、遗产展示 |
| 胜利条件 | 征服/领土占比/生存时长/科技树完成/联盟积分……可组合 |

---

### 2.10 事件系统

| 项目 | 说明 |
|------|------|
| **是什么** | 游戏世界中动态发生的全局/区域事件——天气、天灾、随机事件、限时活动 |
| **为什么是骨头** | 天气/天灾/随机事件增加游戏变化性，触发-生效-移除的生命周期是通用逻辑 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 事件触发→预警期→生效→到期移除→玩家应对 |
| **约束** | 事件效果必须走Buff系统（属性修改系统）；事件必须有持续时间；事件必须有影响范围 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 事件定义 | 触发条件、预警时长、持续时间、影响范围（全局/区域/地块）、效果（Buff列表）、应对选项 |
| 事件触发算法 | 定时触发/随机触发/阶段触发/条件触发——框架提供触发器，配置指定触发策略和参数 |
| 应对选项 | 玩家可以选择的应对方式，每种有消耗（资源/道具）和效果（Buff/资源/实体） |

---

### 2.11 网络同步骨架

| 项目 | 说明 |
|------|------|
| **是什么** | 服务端-客户端数据同步的机制——Command提交、DeltaUpdate广播、FullSnapshot同步 |
| **为什么是骨头** | 每个SLG都需要多人同步，同步骨架是基础设施 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | Command校验→执行→广播DeltaUpdate→定期FullSnapshot→断线重连 |
| **约束** | 所有状态变更必须经服务端确认；DeltaUpdate必须包含版本号；FullSnapshot必须可完整恢复客户端状态 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 消息定义 | 框架定义消息骨架（Command/DeltaUpdate/FullSnapshot/EventNotify），游戏定义具体的Command类型和字段 |
| 同步频率 | DeltaUpdate广播间隔、FullSnapshot广播间隔、关键操作的实时同步标记 |
| 客户端预测策略 | 哪些Command可以客户端预测、预测校正阈值、校正时的插值策略 |

---

### 2.12 持久化骨架

| 项目 | 说明 |
|------|------|
| **是什么** | 游戏数据的存储和恢复机制——脏标记、批量写入、实时写入、宕机恢复 |
| **为什么是骨头** | 每个SLG都需要存档，持久化逻辑是基础设施 |
| **肉怎么挂** | 见下方挂载点 |
| **框架默认行为** | 脏数据标记→定期批量写入→关键操作实时写入→宕机恢复（从最近快照+重放日志） |
| **约束** | 玩家数据必须可完整恢复；关键操作（战斗/交易/掠夺）必须实时持久化；恢复后数据必须一致 |

**挂载点：**

| 挂载点 | 契约概要 |
|--------|---------|
| 实体持久化标记 | 哪些实体类型需要持久化、持久化频率（实时/定期/赛季结束） |
| 数据分区策略 | 按玩家/按联盟/按赛季/按地图Chunk——决定存储的物理分区方式 |
| 备份和恢复策略 | 备份频率、备份保留数量、恢复点选择、恢复后的数据校验 |

---

## 第三章 骨肉接口契约

> 本章是整个文档的核心。把第二章每根骨头的挂载点汇总，用统一的契约格式定义。
> 契约是框架和游戏之间的"合同"——框架承诺提供什么，游戏承诺提供什么，违反契约框架拒绝加载。

### 3.1 契约格式定义

每个挂载点的契约包含以下要素：

| 要素 | 说明 |
|------|------|
| **挂载点ID** | 全局唯一标识，格式为 `{骨头名}.{挂载点名}` |
| **所属骨头** | 属于哪根骨头（第二章的哪个系统） |
| **输入契约** | 肉必须提供什么——数据结构、字段名、字段类型、字段约束 |
| **输出契约** | 框架保证提供什么——回调参数、上下文对象、保证不变量 |
| **校验规则** | 框架怎么校验肉是否合法——类型检查、范围检查、引用完整性检查、自定义校验函数 |
| **默认值** | 不提供肉时的默认行为——框架的"最小惊讶"原则 |
| **示例** | 一个具体的肉的例子，帮助理解契约 |

---

### 3.2 实体系统挂载点契约

#### 3.2.1 EntityType定义契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `entity.type_define` |
| **所属骨头** | 实体系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一，不可为空<br>`name`：string，显示名称<br>`components`：string数组，引用component_define中定义的组件ID<br>`tags`：string数组，可选，用于分类查询<br>`persist`：bool，是否持久化，默认false |
| **输出契约** | 框架保证：按id查询返回唯一EntityType；按tag查询返回所有匹配的EntityType；创建实体时自动挂载指定组件 |
| **校验规则** | ①id全局唯一 ②components中每个ID必须在component_define中存在 ③id不可为空 |
| **默认值** | 无（EntityType必须定义，否则框架无法创建实体） |
| **示例** | `{"id":"city","name":"城市","components":["position","owner","health","resource_storage","building_slots"],"tags":["persist","map_entity"],"persist":true}` |

#### 3.2.2 Component定义契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `entity.component_define` |
| **所属骨头** | 实体系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`fields`：数组，每个字段包含：<br>&nbsp;&nbsp;`name`：string，字段名<br>&nbsp;&nbsp;`type`：枚举{int, float, bool, string, int_array, float_array, string_array, ref}<br>&nbsp;&nbsp;`default`：该类型的默认值<br>&nbsp;&nbsp;`range`：可选，[min, max]范围约束<br>&nbsp;&nbsp;`ref_type`：可选，type=ref时，引用的EntityType的id |
| **输出契约** | 框架保证：组件可按id查询；字段值在range范围内；ref类型字段引用的实体存在时返回实体，不存在时返回null |
| **校验规则** | ①id全局唯一 ②每个字段的default值符合type和range ③ref类型字段必须指定ref_type |
| **默认值** | 无（Component必须定义） |
| **示例** | `{"id":"health","fields":[{"name":"current","type":"int","default":100,"range":[0,999999]},{"name":"max","type":"int","default":100,"range":[1,999999]}]}` |

#### 3.2.3 实体模板契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `entity.template` |
| **所属骨头** | 实体系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`entity_type`：string，引用EntityType的id<br>`component_defaults`：map，key=组件ID，value=字段默认值覆盖 |
| **输出契约** | 框架保证：按模板创建实体时，组件初始值为模板值覆盖默认值 |
| **校验规则** | ①id全局唯一 ②entity_type必须存在 ③component_defaults中每个组件ID必须在EntityType的components中 ④每个字段值符合Component定义的type和range |
| **默认值** | 不提供模板时，使用Component定义中的default值 |
| **示例** | `{"id":"barracks_lv1","entity_type":"building","component_defaults":{"health":{"current":500,"max":500},"building_info":{"type":"barracks","level":1}}}` |

---

### 3.3 地图系统挂载点契约

#### 3.3.1 地图规格契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `map.spec` |
| **所属骨头** | 地图系统 |
| **输入契约** | 对象，包含：<br>`width`：int，地图宽度（格数），>0<br>`height`：int，地图高度（格数），>0<br>`grid_type`：枚举{square, hex}，网格类型<br>`chunk_size`：int，分块大小（格数），>0，必须能整除width和height<br>`origin`：可选，[x, y]坐标原点偏移，默认[0,0] |
| **输出契约** | 框架保证：坐标范围在[0, width)×[0, height)内有效；Chunk按chunk_size分块；六角格使用offset坐标系 |
| **校验规则** | ①width>0, height>0 ②chunk_size>0 ③width%chunk_size==0, height%chunk_size==0 |
| **默认值** | 无（地图规格必须定义） |
| **示例** | `{"width":600,"height":600,"grid_type":"hex","chunk_size":50}` |

#### 3.3.2 地形类型契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `map.terrain_type` |
| **所属骨头** | 地图系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`name`：string，显示名称<br>`passability`：map，key=移动类型ID，value=bool（是否可通行）<br>`modifiers`：对象，可选，包含：<br>&nbsp;&nbsp;`defense`：float，防御修正，默认0<br>&nbsp;&nbsp;`move_cost`：float，移动消耗修正，默认1.0<br>&nbsp;&nbsp;`production`：float，产出修正，默认1.0<br>`visual_id`：string，视觉效果ID，用于客户端渲染 |
| **输出契约** | 框架保证：寻路时按passability判断通行性；战斗时按modifiers.defense修正防御；产出计算时按modifiers.production修正 |
| **校验规则** | ①id全局唯一 ②passability必须覆盖所有移动类型 ③move_cost>0 |
| **默认值** | 无（地形类型必须定义，至少1种） |
| **示例** | `{"id":"plains","name":"平原","passability":{"ground":true,"flight":true,"water":false},"modifiers":{"defense":0,"move_cost":1.0,"production":1.0},"visual_id":"terrain_plains"}` |

#### 3.3.3 地形生成规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `map.terrain_gen` |
| **所属骨头** | 地图系统 |
| **输入契约** | 对象，包含：<br>`seed`：int，可选，随机种子，不提供则用系统时间<br>`weights`：map，key=地形ID，value=float权重，所有权重之和=1.0<br>`regions`：数组，可选，区域规则，每个区域包含：<br>&nbsp;&nbsp;`center`：[x, y]，区域中心<br>&nbsp;&nbsp;`radius`：int，区域半径<br>&nbsp;&nbsp;`terrain`：string，区域内强制地形ID<br>`special_points`：数组，可选，特殊点，每个点包含：<br>&nbsp;&nbsp;`position`：[x, y]<br>&nbsp;&nbsp;`terrain`：string，该点强制地形ID<br>&nbsp;&nbsp;`tag`：string，可选，标记（如"birth_point"） |
| **输出契约** | 框架保证：生成的地图中每个格子都有地形；special_points的位置地形为指定地形；regions内的格子地形为指定地形 |
| **校验规则** | ①weights中所有地形ID必须存在 ②weights之和=1.0（容差0.001） ③special_points的position在地图范围内 ④regions的center在地图范围内 |
| **默认值** | 全地图使用weights中权重最高的地形 |
| **示例** | `{"seed":42,"weights":{"plains":0.4,"forest":0.25,"mountain":0.15,"water":0.2},"special_points":[{"position":[10,10],"terrain":"plains","tag":"birth_point"}]}` |

#### 3.3.4 迷雾规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `map.fog` |
| **所属骨头** | 地图系统 |
| **输入契约** | 对象，包含：<br>`enabled`：bool，是否启用迷雾，默认false<br>`vision_radius`：int，基础视野半径（格数），>0<br>`states`：数组，迷雾状态列表，每个状态包含：<br>&nbsp;&nbsp;`id`：string，状态ID（如visible/explored/unknown）<br>&nbsp;&nbsp;`can_see_entities`：bool，该状态下是否能看到实体<br>&nbsp;&nbsp;`can_see_terrain`：bool，该状态下是否能看到地形<br>`alliance_share`：bool，联盟是否共享视野，默认false<br>`decay_rules`：可选，视野衰减规则（离开后多久从visible变为explored） |
| **输出契约** | 框架保证：视野内的格子状态为visible；曾经视野内但当前不在的格子为explored；从未在视野内的为unknown |
| **校验规则** | ①states至少包含visible/explored/unknown三种 ②vision_radius>0 |
| **默认值** | `enabled=false`，即全地图可见 |
| **示例** | `{"enabled":true,"vision_radius":5,"states":[{"id":"visible","can_see_entities":true,"can_see_terrain":true},{"id":"explored","can_see_entities":false,"can_see_terrain":true},{"id":"unknown","can_see_entities":false,"can_see_terrain":false}],"alliance_share":true}` |

---

### 3.4 移动系统挂载点契约

#### 3.4.1 移动类型契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `movement.type_define` |
| **所属骨头** | 移动系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一（如ground/flight/water/underground）<br>`name`：string，显示名称<br>`base_speed`：float，基础速度（格/秒），>0<br>`terrain_passability`：map，key=地形ID，value=bool，是否可通行<br>`terrain_cost`：map，可选，key=地形ID，value=float，该地形的移动消耗倍率，默认1.0 |
| **输出契约** | 框架保证：寻路时按terrain_passability过滤不可通行格子；移动时间=距离×terrain_cost/base_speed |
| **校验规则** | ①id全局唯一 ②base_speed>0 ③terrain_passability覆盖所有地形类型 ④terrain_cost中所有值>0 |
| **默认值** | 无（必须定义至少1种移动类型） |
| **示例** | `{"id":"ground","name":"地面","base_speed":2.0,"terrain_passability":{"plains":true,"forest":true,"mountain":false,"water":false},"terrain_cost":{"plains":1.0,"forest":1.5}}` |

#### 3.4.2 速度修正契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `movement.speed_modifier` |
| **所属骨头** | 移动系统 |
| **输入契约** | 数组，每个元素包含：<br>`source`：枚举{faction, tech, terrain, weather, buff}，修正来源<br>`source_id`：string，来源的具体ID（如种族ID/科技ID/Buff ID）<br>`modifier_type`：枚举{add, multiply}，修正方式<br>`value`：float，修正值（add时为格/秒增量，multiply时为倍率）<br>`priority`：int，修正优先级（数值越小越先计算） |
| **输出契约** | 框架保证：修正按priority升序计算；add修正先求和，multiply修正后求积；最终速度=max(base_speed + Σadd, 0) × Πmultiply |
| **校验规则** | ①source_id必须存在 ②multiply类型的value>0 ③priority不重复 |
| **默认值** | 无修正，速度=base_speed |
| **示例** | `{"source":"tech","source_id":"tech_riding","modifier_type":"multiply","value":1.2,"priority":10}` |

#### 3.4.3 移动状态扩展契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `movement.state_extend` |
| **所属骨头** | 移动系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，状态ID，不可与框架默认状态冲突<br>`name`：string，显示名称<br>`transitions`：数组，状态转换规则，每个转换包含：<br>&nbsp;&nbsp;`from`：string，源状态ID<br>&nbsp;&nbsp;`to`：string，目标状态ID<br>&nbsp;&nbsp;`condition`：string，条件表达式<br>&nbsp;&nbsp;`action`：可选，转换时触发的动作 |
| **输出契约** | 框架保证：状态机按transitions定义的规则转换；不满足任何条件时不转换 |
| **校验规则** | ①id不可与默认状态（idle/marching/combat/retreating/garrisoned/patrol）冲突 ②transitions中from和to必须存在 ③condition表达式语法合法 |
| **默认值** | 框架默认6种状态：idle/marching/combat/retreating/garrisoned/patrol，及默认转换规则 |
| **示例** | `{"id":"ambush","name":"伏击","transitions":[{"from":"idle","to":"ambush","condition":"has_ambush_order AND terrain == forest"}]}` |

#### 3.4.4 拦截规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `movement.intercept_rule` |
| **所属骨头** | 移动系统 |
| **输入契约** | 对象，包含：<br>`trigger_distance`：float，触发拦截的距离（格数），>0<br>`conditions`：数组，拦截条件列表，每个条件包含：<br>&nbsp;&nbsp;`diplomacy`：枚举外交状态，什么外交关系下触发拦截<br>&nbsp;&nbsp;`direction`：枚举{approaching, crossing, any}，拦截方向<br>`cooldown`：float，拦截冷却时间（秒），>=0 |
| **输出契约** | 框架保证：两军距离<=trigger_distance且满足conditions时触发拦截事件；拦截后进入cooldown期间不再拦截 |
| **校验规则** | ①trigger_distance>0 ②cooldown>=0 ③conditions至少1条 |
| **默认值** | `trigger_distance=1.0, conditions=[{diplomacy:hostile, direction:any}], cooldown=0` |
| **示例** | `{"trigger_distance":1.5,"conditions":[{"diplomacy":"hostile","direction":"approaching"}],"cooldown":5.0}` |

---

### 3.5 战斗系统挂载点契约

#### 3.5.1 伤害公式契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `combat.damage_formula` |
| **所属骨头** | 战斗系统 |
| **输入契约** | 对象，包含：<br>`formulas`：map，key=公式ID，value=公式表达式字符串<br>公式可用变量：`atk`攻击力、`def`防御力、`counter`克制倍率、`terrain_mod`地形修正、`buff_mod`Buff修正、`random`随机因子<br>公式可用运算：+、-、*、/、min()、max()、floor()、ceil()、random(a,b) |
| **输出契约** | 框架保证：公式引擎按表达式求值；结果为非负数（负数钳制为0）；变量替换后求值 |
| **校验规则** | ①每个公式表达式语法合法 ②公式中引用的变量在框架提供的变量列表中 ③公式求值结果必须>=0（框架钳制） |
| **默认值** | `basic_damage = max(atk * counter - def, 1)` |
| **示例** | `{"formulas":{"basic_damage":"max(atk * counter - def * terrain_mod, 1)","siege_damage":"atk * counter * 0.5"}}` |

#### 3.5.2 克制矩阵契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `combat.counter_matrix` |
| **所属骨头** | 战斗系统 |
| **输入契约** | 对象，包含：<br>`dimensions`：string数组，维度标签（如兵种类型列表）<br>`matrix`：二维float数组，N×N方阵，dimensions[i]对dimensions[j]的克制倍率=matrix[i][j]<br>约束：对角线元素必须=1.0；所有元素>0 |
| **输出契约** | 框架保证：战斗时按攻击方类型和防御方类型查表获取克制倍率；matrix[i][j]×matrix[j][i]不要求=1.0（允许非对称克制） |
| **校验规则** | ①matrix为方阵 ②对角线全为1.0 ③所有元素>0 ④dimensions长度=matrix维度 |
| **默认值** | 1×1矩阵，值为1.0（无克制） |
| **示例** | `{"dimensions":["infantry","cavalry","archer"],"matrix":[[1.0,0.8,1.2],[1.2,1.0,0.8],[0.8,1.2,1.0]]}` |

#### 3.5.3 战斗阶段契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `combat.phase_define` |
| **所属骨头** | 战斗系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，阶段ID<br>`name`：string，显示名称<br>`order`：int，阶段顺序，从0开始递增<br>`duration`：float或string，阶段持续条件——float为tick数，string为条件表达式<br>`actions`：数组，该阶段每tick执行的动作列表（如"ranged_attack"/"melee_attack"/"siege_attack"）<br>`transition_condition`：string，进入下一阶段的条件表达式 |
| **输出契约** | 框架保证：战斗按order顺序推进阶段；每个阶段按duration/transition_condition决定何时进入下一阶段；每个tick执行该阶段的actions |
| **校验规则** | ①order从0开始连续递增 ②至少1个阶段 ③最后一个阶段的transition_condition为"combat_end" |
| **默认值** | 4个阶段：approach(接近)→ranged(远程)→melee(近战)→end(结束) |
| **示例** | `{"id":"ranged","name":"远程阶段","order":1,"duration":"ranged_units_alive == 0","actions":["ranged_attack"],"transition_condition":"ranged_ammo == 0 OR ranged_units_alive == 0"}` |

#### 3.5.4 战斗触发条件契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `combat.trigger_condition` |
| **所属骨头** | 战斗系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，触发类型ID（如intercept/active_attack/siege/ambush）<br>`name`：string，显示名称<br>`condition`：string，条件表达式<br>`priority`：int，触发优先级（多个条件同时满足时，优先级高的生效） |
| **输出契约** | 框架保证：满足condition时触发战斗；多个条件同时满足时按priority选择 |
| **校验规则** | ①id全局唯一 ②condition表达式语法合法 |
| **默认值** | intercept(拦截触发)和active_attack(主动攻击)两种 |
| **示例** | `{"id":"ambush","name":"伏击触发","condition":"attacker.state == ambush AND defender.state == marching","priority":20}` |

#### 3.5.5 战斗结束条件契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `combat.end_condition` |
| **所属骨头** | 战斗系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，结束条件ID<br>`condition`：string，条件表达式<br>`result`：枚举{attacker_win, defender_win, draw}，满足条件时的战斗结果 |
| **输出契约** | 框架保证：每tick检查所有结束条件，第一个满足的条件决定战斗结果；无任何条件满足则战斗继续 |
| **校验规则** | ①至少1个结束条件 ②condition表达式语法合法 |
| **默认值** | `{"id":"annihilation","condition":"attacker_hp == 0 OR defender_hp == 0","result":"attacker_win"}`（全灭） |
| **示例** | `[{"id":"annihilation","condition":"defender_hp == 0","result":"attacker_win"},{"id":"timeout","condition":"combat_ticks > 100","result":"draw"}]` |

#### 3.5.6 战斗结果处理契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `combat.result_handler` |
| **所属骨头** | 战斗系统 |
| **输入契约** | 对象，包含三个结果分支：<br>`attacker_win`：数组，胜方（攻击方）的处理动作列表<br>`defender_win`：数组，胜方（防御方）的处理动作列表<br>`draw`：数组，平局的处理动作列表<br>每个动作包含：<br>&nbsp;&nbsp;`type`：枚举{pursue, retreat, loot, grant_exp, capture, destroy}<br>&nbsp;&nbsp;`params`：map，动作参数（如掠夺比例、追击距离、经验公式ID） |
| **输出契约** | 框架保证：战斗结束后按结果执行对应的动作列表；动作按数组顺序执行 |
| **校验规则** | 三个分支都必须定义；动作type必须在框架支持的类型列表中 |
| **默认值** | 胜方：掠夺+获得经验；败方：撤退；平局：双方原地 |
| **示例** | `{"attacker_win":[{"type":"loot","params":{"ratio":0.3}},{"type":"grant_exp","params":{"formula":"victory_exp"}}],"defender_win":[{"type":"pursue","params":{"distance":3}},{"type":"grant_exp","params":{"formula":"victory_exp"}}],"draw":[]}` |

#### 3.5.7 攻城规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `combat.siege_rule` |
| **所属骨头** | 战斗系统 |
| **输入契约** | 对象，包含：<br>`wall_hp_component`：string，城墙血量组件ID<br>`wall_defense_bonus`：float，城墙存在时防御方修正倍率，>0<br>`siege_damage_formula`：string，攻城伤害公式ID<br>`wall_breach_actions`：数组，城墙被攻破后的动作列表<br>`inner_combat`：bool，城墙被攻破后是否进入城内战斗，默认true |
| **输出契约** | 框架保证：城墙存在时防御方获得wall_defense_bonus修正；城墙HP降为0时触发wall_breach_actions；inner_combat=true时城墙破后继续城内战斗 |
| **校验规则** | ①wall_hp_component必须存在 ②wall_defense_bonus>0 ③siege_damage_formula必须存在 |
| **默认值** | 无攻城规则（普通野战） |
| **示例** | `{"wall_hp_component":"wall_health","wall_defense_bonus":2.0,"siege_damage_formula":"siege_damage","wall_breach_actions":[{"type":"send_notify","params":{"msg":"wall_breached"}}],"inner_combat":true}` |

---

### 3.6 经济系统挂载点契约

#### 3.6.1 资源类型契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `economy.resource_type` |
| **所属骨头** | 经济系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`name`：string，显示名称<br>`tradable`：bool，是否可贸易<br>`lootable`：bool，是否可掠夺<br>`loot_ratio`：float，掠夺比例，[0,1]<br>`storage_cap_formula`：string，存储上限公式ID，可选<br>`icon_id`：string，图标ID |
| **输出契约** | 框架保证：资源值不超过storage_cap_formula计算结果；掠夺量=持有量×loot_ratio |
| **校验规则** | ①id全局唯一 ②至少1种资源 ③loot_ratio∈[0,1] |
| **默认值** | 无（必须定义至少1种资源） |
| **示例** | `{"id":"food","name":"粮食","tradable":true,"lootable":true,"loot_ratio":0.3,"storage_cap_formula":"base_cap","icon_id":"icon_food"}` |

#### 3.6.2 产出来源契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `economy.production_source` |
| **所属骨头** | 经济系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`source_type`：枚举{building, tile, buff, event}，产出来源类型<br>`source_id`：string，来源的具体ID<br>`resource_id`：string，产出的资源类型ID<br>`formula`：string，产出公式ID<br>`tick_interval`：int，产出间隔（tick数），>0 |
| **输出契约** | 框架保证：每tick_interval个tick按formula计算产出并增加资源 |
| **校验规则** | ①resource_id必须存在 ②formula必须存在 ③tick_interval>0 |
| **默认值** | 无产出 |
| **示例** | `{"id":"farm_food","source_type":"building","source_id":"building_farm","resource_id":"food","formula":"farm_output","tick_interval":60}` |

#### 3.6.3 消耗定义契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `economy.consumption` |
| **所属骨头** | 经济系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`consume_type`：枚举{population, army_maintain, building_maintain, tech_research}，消耗类型<br>`consume_id`：string，消耗来源的具体ID<br>`resource_id`：string，消耗的资源类型ID<br>`formula`：string，消耗公式ID<br>`tick_interval`：int，消耗间隔（tick数），>0 |
| **输出契约** | 框架保证：每tick_interval个tick按formula计算消耗并扣减资源；资源不足时触发饥荒规则 |
| **校验规则** | ①resource_id必须存在 ②formula必须存在 ③tick_interval>0 |
| **默认值** | 无消耗 |
| **示例** | `{"id":"army_food","consume_type":"army_maintain","consume_id":"unit_infantry","resource_id":"food","formula":"army_upkeep","tick_interval":60}` |

#### 3.6.4 采集规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `economy.gather_rule` |
| **所属骨头** | 经济系统 |
| **输入契约** | 对象，包含：<br>`gather_speed`：float，采集速度（资源/秒），>0<br>`carry_capacity`：float，载重上限，>0<br>`gather_team_type`：string，采集队的EntityType ID<br>`ambush_vulnerable`：bool，采集时是否可被伏击<br>`return_on_full`：bool，载满后是否自动返回，默认true |
| **输出契约** | 框架保证：采集量=min(gather_speed×时间, carry_capacity)；载满后按return_on_full决定是否自动返回 |
| **校验规则** | ①gather_speed>0 ②carry_capacity>0 ③gather_team_type必须存在 |
| **默认值** | 无采集规则（不启用采集） |
| **示例** | `{"gather_speed":10.0,"carry_capacity":500.0,"gather_team_type":"gather_team","ambush_vulnerable":true,"return_on_full":true}` |

#### 3.6.5 贸易规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `economy.trade_rule` |
| **所属骨头** | 经济系统 |
| **输入契约** | 对象，包含：<br>`enabled`：bool，是否启用贸易<br>`tax_rate`：float，税率，[0,1]<br>`trade_route_type`：string，贸易路线的EntityType ID<br>`transport_speed`：float，运输速度（格/秒），>0<br>`min_trade_amount`：map，key=资源ID，value=int，最低贸易量<br>`max_trade_amount`：map，key=资源ID，value=int，最高贸易量 |
| **输出契约** | 框架保证：贸易量在[min, max]范围内；实际到账=贸易量×(1-tax_rate) |
| **校验规则** | ①tax_rate∈[0,1] ②transport_speed>0 ③min_trade_amount中每个资源ID必须存在且tradable=true |
| **默认值** | `enabled=false` |
| **示例** | `{"enabled":true,"tax_rate":0.1,"trade_route_type":"trade_caravan","transport_speed":1.5,"min_trade_amount":{"food":100},"max_trade_amount":{"food":10000}}` |

#### 3.6.6 掠夺规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `economy.loot_rule` |
| **所属骨头** | 经济系统 |
| **输入契约** | 对象，包含：<br>`protection_amount`：map，key=资源ID，value=float，保护量（低于此量不可掠夺）<br>`protection_ratio`：float，保护比例，[0,1]（掠夺后剩余量不低于持有量×此比例）<br>`city_protection`：bool，城墙存在时是否保护资源不可掠夺<br>`loot_order`：string数组，掠夺优先级（先掠夺哪种资源） |
| **输出契约** | 框架保证：掠夺后每种资源剩余量>=protection_amount；掠夺后总剩余量>=掠夺前总持有量×protection_ratio |
| **校验规则** | ①protection_ratio∈[0,1] ②loot_order中每个资源ID必须存在且lootable=true |
| **默认值** | `protection_amount={}, protection_ratio=0, city_protection=false` |
| **示例** | `{"protection_amount":{"food":100},"protection_ratio":0.2,"city_protection":true,"loot_order":["gold","food","wood"]}` |

#### 3.6.7 饥荒/溢出规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `economy.famine_overflow` |
| **所属骨头** | 经济系统 |
| **输入契约** | 对象，包含：<br>`famine`：对象，饥荒规则：<br>&nbsp;&nbsp;`trigger`：string，触发条件表达式（如"food < 0"）<br>&nbsp;&nbsp;`effects`：数组，饥荒效果（Buff ID列表）<br>&nbsp;&nbsp;`recovery`：string，恢复条件表达式<br>`overflow`：对象，溢出规则：<br>&nbsp;&nbsp;`strategy`：枚举{discard, stop_production, convert}，溢出策略<br>&nbsp;&nbsp;`convert_target`：可选，string，convert策略时转化为哪种资源ID<br>&nbsp;&nbsp;`convert_ratio`：可选，float，转化比率 |
| **输出契约** | 框架保证：触发famine.trigger时挂载famine.effects的Buff；满足famine.recovery时移除Buff；溢出时按strategy处理 |
| **校验规则** | ①trigger和recovery表达式语法合法 ②effects中每个Buff ID必须存在 ③convert策略时convert_target必须存在 |
| **默认值** | famine:无触发；overflow:strategy=discard |
| **示例** | `{"famine":{"trigger":"food < 0","effects":["buff_famine"],"recovery":"food >= 0"},"overflow":{"strategy":"convert","convert_target":"gold","convert_ratio":0.5}}` |

---

### 3.7 属性修改系统挂载点契约

#### 3.7.1 Buff模板契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `buff.template` |
| **所属骨头** | 属性修改系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`name`：string，显示名称<br>`effects`：数组，效果列表，每个效果包含：<br>&nbsp;&nbsp;`attr`：string，修改的属性名<br>&nbsp;&nbsp;`op`：枚举{add, multiply, clamp_min, clamp_max}，操作类型<br>&nbsp;&nbsp;`value`：float，操作值<br>`duration`：float或string，持续时间（秒）或"permanent"<br>`stack_rule`：枚举{replace, stack, refresh}，叠加规则<br>`source_tag`：string，来源标记（用于驱散分组）<br>`dispellable`：bool，是否可被驱散，默认true<br>`dispel_priority`：int，驱散优先级（数值越大越先被驱散） |
| **输出契约** | 框架保证：按stack_rule处理同ID Buff；按属性计算顺序计算最终值；到期自动移除；驱散时按dispel_priority排序 |
| **校验规则** | ①id全局唯一 ②effects中每个attr必须存在于实体组件中 ③duration>0或="permanent" ④clamp_min的value<=clamp_max的value（如果同时存在） |
| **默认值** | 无Buff模板 |
| **示例** | `{"id":"buff_rage","name":"狂暴","effects":[{"attr":"atk","op":"multiply","value":1.5},{"attr":"def","op":"add","value":-10}],"duration":30,"stack_rule":"refresh","source_tag":"skill","dispellable":true,"dispel_priority":5}` |

#### 3.7.2 属性计算顺序契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `buff.calc_order` |
| **所属骨头** | 属性修改系统 |
| **输入契约** | **不可配置**。框架固定计算顺序如下：<br>① **基础值**：组件中存储的原始值<br>② **固定加成求和**：所有op=add且source_tag∈{base, faction, tech, equipment}的效果求和<br>③ **乘百分比求积**：所有op=multiply的效果求积<br>④ **技能固定加成求和**：所有op=add且source_tag∈{skill, buff, event}的效果求和<br>⑤ **钳制**：op=clamp_min的效果取max，op=clamp_max的效果取min |
| **输出契约** | 框架保证：严格按照上述顺序计算；最终值=clamp_min(max, clamp_max(min, ④结果)) |
| **校验规则** | 无（不可配置，无需校验） |
| **默认值** | 即上述固定顺序 |
| **示例** | 基础atk=100 → 种族加成+20 → 科技乘1.2 → 技能加成+30 → clamp_min(50) → 最终atk=max(50, (100+20)×1.2+30) = max(50, 174) = 174 |

#### 3.7.3 叠加规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `buff.stack_rule` |
| **所属骨头** | 属性修改系统 |
| **输入契约** | **不可扩展**。框架提供三种叠加规则：<br>• `replace`：同ID新Buff替换旧Buff<br>• `stack`：同ID Buff效果叠加，各自独立计时<br>• `refresh`：同ID新Buff刷新旧Buff的持续时间，不叠加效果 |
| **输出契约** | 框架保证：按Buff模板中定义的stack_rule执行叠加逻辑 |
| **校验规则** | stack_rule必须是{replace, stack, refresh}之一 |
| **默认值** | replace |
| **示例** | stack_rule=stack时，两个atk+10的Buff共存，最终加成=+20 |

#### 3.7.4 驱散规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `buff.dispel_rule` |
| **所属骨头** | 属性修改系统 |
| **输入契约** | 驱散由Buff模板中的字段控制：<br>• `dispellable`：是否可被驱散<br>• `dispel_priority`：驱散优先级<br>• `source_tag`：来源标记，驱散时可按source_tag批量驱散 |
| **输出契约** | 框架保证：驱散时只移除dispellable=true的Buff；按dispel_priority从大到小依次驱散；可按source_tag批量驱散同组Buff |
| **校验规则** | 无（由Buff模板中的字段决定） |
| **默认值** | dispellable=true, dispel_priority=0 |
| **示例** | 驱散所有source_tag="skill"的Buff，按dispel_priority从大到小移除，直到移除数量达到指定值 |

---

### 3.8 规则系统挂载点契约

#### 3.8.1 规则定义契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `rule.define` |
| **所属骨头** | 规则系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`trigger_event`：string，触发事件名（如"on_combat_start"/"on_season_phase_change"/"on_entity_created"）<br>`condition`：string，条件表达式，可选（不提供则无条件触发）<br>`action_type`：string，动作类型；内置7个：apply_buff / remove_buff / modify_resource / spawn_entity / trigger_combat / send_notify / change_terrain，题材层可注册新类型（见3.8.2，运行时以注册表校验为准）<br>`action_params`：map，动作参数（取决于action_type）<br>`cooldown`：float，冷却时间（秒），>=0<br>`one_time`：bool，是否一次性规则，默认false<br>`priority`：int，优先级（同事件多规则时按优先级排序） |
| **输出契约** | 框架保证：事件触发时检查condition→执行action→进入cooldown；one_time规则执行后自动移除；循环深度超过阈值报错 |
| **校验规则** | ①id全局唯一 ②action_type在动作注册表中已注册（内置7个+题材层注册，启动全量校验，未注册拒绝启动） ③cooldown>=0 ④condition表达式语法合法 ⑤框架检测循环触发（A触发B，B触发A） |
| **默认值** | 无规则 |
| **示例** | `{"id":"ambush_bonus","trigger_event":"on_combat_start","condition":"attacker.has_buff(ambush) AND terrain == forest","action_type":"apply_buff","action_params":{"buff_id":"buff_ambush_bonus","target":"attacker"},"cooldown":0,"one_time":false,"priority":10}` |

#### 3.8.2 动作类型契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `rule.action_type` |
| **所属骨头** | 规则系统 |
| **输入契约** | **内置7个，注册表可扩展**。框架内置以下动作类型（新类型=题材层实现RuleActionHandler并经注册表init()注册，见ADR-002/ADR-003）：<br>• `apply_buff`：挂载Buff，params={buff_id, target}<br>• `remove_buff`：移除Buff，params={buff_id, target}或{source_tag, target}<br>• `modify_resource`：修改资源，params={resource_id, amount, target}<br>• `spawn_entity`：生成实体，params={template_id, position, owner}<br>• `trigger_combat`：触发战斗，params={attacker_id, defender_id}<br>• `send_notify`：发送通知，params={msg_id, target, params}<br>• `change_terrain`：改变地形，params={position, terrain_id} |
| **输出契约** | 框架保证：每种动作类型按定义的params执行；params校验不通过则跳过该动作并记录错误日志 |
| **校验规则** | action_type必须在动作注册表中已注册（内置7个+题材层注册的新类型）；新类型=题材层init()注册，框架代码零改动；启动全量校验，未注册动作拒绝启动 |
| **默认值** | 即上述7种内置动作（注册表初始内容） |
| **示例** | `{"action_type":"modify_resource","action_params":{"resource_id":"gold","amount":100,"target":"attacker_owner"}}` |

#### 3.8.3 条件表达式契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `rule.condition_expr` |
| **所属骨头** | 规则系统 |
| **输入契约** | **不可扩展语法**。框架支持以下表达式语法：<br>• 比较运算：>、>=、<、<=、==、!=<br>• 逻辑运算：AND、OR、NOT<br>• 集合运算：IN、NOT_IN<br>• 变量：事件上下文提供的变量（如attacker、defender、terrain、season_phase等）<br>• 函数：has_buff(id)、has_entity(type)、resource_gt(id, amount)、diplomacy_is(state) |
| **输出契约** | 框架保证：表达式求值结果为bool；变量不存在时返回false；函数参数不合法时返回false |
| **校验规则** | 表达式语法必须合法（框架提供语法校验器） |
| **默认值** | 无条件（即始终为true） |
| **示例** | `"attacker.faction != defender.faction AND terrain IN [forest, mountain] AND has_buff(ambush)"` |

---

### 3.9 联盟系统挂载点契约

#### 3.9.1 联盟配置契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `alliance.config` |
| **所属骨头** | 联盟系统 |
| **输入契约** | 对象，包含：<br>`max_members`：int，成员上限，>0<br>`max_level`：int，联盟等级上限，>0<br>`level_config`：数组，每级配置，包含：{level, member_cap, tech_cap, territory_cap}<br>`creation_cost`：map，key=资源ID，value=float，创建联盟消耗<br>`join_cooldown`：float，退出后重新加入的冷却时间（秒）<br>`territory_rule`：枚举{occupy, share, contest}，领土规则 |
| **输出契约** | 框架保证：成员数不超过max_members；联盟等级不超过max_level；创建联盟时扣减creation_cost |
| **校验规则** | ①max_members>0 ②max_level>0 ③level_config长度=max_level ④creation_cost中每个资源ID必须存在 |
| **默认值** | `max_members=50, max_level=10, territory_rule=occupy` |
| **示例** | `{"max_members":100,"max_level":5,"level_config":[{"level":1,"member_cap":20,"tech_cap":3,"territory_cap":50},{"level":2,"member_cap":40,"tech_cap":5,"territory_cap":100}],"creation_cost":{"gold":10000},"join_cooldown":86400,"territory_rule":"occupy"}` |

#### 3.9.2 外交状态契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `alliance.diplomacy` |
| **所属骨头** | 联盟系统 |
| **输入契约** | 对象，包含：<br>`states`：数组，外交状态列表，每个状态包含：<br>&nbsp;&nbsp;`id`：string，状态ID<br>&nbsp;&nbsp;`name`：string，显示名称<br>&nbsp;&nbsp;`can_attack`：bool，该状态下是否可攻击<br>&nbsp;&nbsp;`can_trade`：bool，该状态下是否可贸易<br>&nbsp;&nbsp;`can_share_vision`：bool，该状态下是否共享视野<br>`transitions`：数组，合法转换路径，每个转换包含：<br>&nbsp;&nbsp;`from`：string，源状态ID<br>&nbsp;&nbsp;`to`：string，目标状态ID<br>&nbsp;&nbsp;`cooldown`：float，转换冷却时间（秒） |
| **输出契约** | 框架保证：外交状态只能按transitions定义的路径转换；不在transitions中的转换请求被拒绝 |
| **校验规则** | ①states至少包含敌对和中立 ②transitions中from和to必须在states中 ③状态转换图不能有孤立节点 |
| **默认值** | 5种状态：hostile/neutral/friendly/allied/nap，及标准转换路径 |
| **示例** | `{"states":[{"id":"hostile","name":"敌对","can_attack":true,"can_trade":false,"can_share_vision":false},{"id":"neutral","name":"中立","can_attack":false,"can_trade":true,"can_share_vision":false},{"id":"allied","name":"同盟","can_attack":false,"can_trade":true,"can_share_vision":true}],"transitions":[{"from":"hostile","to":"neutral","cooldown":0},{"from":"neutral","to":"allied","cooldown":86400}]}` |

#### 3.9.3 联盟科技契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `alliance.tech` |
| **所属骨头** | 联盟系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，科技ID<br>`name`：string，显示名称<br>`level_cap`：int，等级上限，>0<br>`contribution_type`：枚举{resource, activity}，贡献方式<br>`contribution_cost`：map，key=资源ID，value=float，每级贡献消耗<br>`effects`：数组，效果列表（Buff ID列表，每级效果可不同）<br>`prerequisites`：string数组，前置科技ID列表 |
| **输出契约** | 框架保证：科技等级不超过level_cap；前置科技未满足时不可研究；贡献时扣减contribution_cost |
| **校验规则** | ①id全局唯一 ②level_cap>0 ③effects中每个Buff ID必须存在 ④prerequisites中每个ID必须存在 ⑤无循环依赖 |
| **默认值** | 无联盟科技 |
| **示例** | `{"id":"alliance_atk","name":"联盟攻击","level_cap":5,"contribution_type":"resource","contribution_cost":{"gold":1000},"effects":["buff_alliance_atk_1","buff_alliance_atk_2"],"prerequisites":[]}` |

#### 3.9.4 联盟战争契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `alliance.war` |
| **所属骨头** | 联盟系统 |
| **输入契约** | 对象，包含：<br>`declare_condition`：string，宣战条件表达式<br>`declare_cost`：map，key=资源ID，value=float，宣战消耗<br>`scoring_rules`：数组，计分规则，每条包含：{event, score}（如击杀+10，占领+50）<br>`duration`：float，战争持续时间（秒），0=无限<br>`reward`：对象，奖励配置，包含：{winner, loser, draw}，每个为资源map<br>`peace_condition`：string，议和条件表达式 |
| **输出契约** | 框架保证：宣战时检查declare_condition并扣减declare_cost；战争期间按scoring_rules累计积分；到期或满足peace_condition时结束战争并发放奖励 |
| **校验规则** | ①declare_condition表达式语法合法 ②duration>=0 ③declare_cost中每个资源ID必须存在 |
| **默认值** | 无联盟战争规则 |
| **示例** | `{"declare_condition":"alliance.level >= 3","declare_cost":{"gold":50000},"scoring_rules":[{"event":"kill_enemy","score":10},{"event":"occupy_territory","score":50}],"duration":604800,"reward":{"winner":{"gold":100000},"loser":{},"draw":{}},"peace_condition":"war_score_diff > 1000"}` |

---

### 3.10 赛季系统挂载点契约

#### 3.10.1 赛季配置契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `season.config` |
| **所属骨头** | 赛季系统 |
| **输入契约** | 对象，包含：<br>`total_duration`：float，赛季总时长（秒），>0<br>`phases`：string数组，阶段ID列表（按顺序）<br>`rank_tiers`：数组，段位定义，每个包含：{id, name, min_score, max_score, reward}<br>`victory_conditions`：string数组，胜利条件ID列表（OR关系，满足任一即胜利） |
| **输出契约** | 框架保证：赛季按phases顺序推进；总时长到期时赛季结束；排名按段位划分 |
| **校验规则** | ①total_duration>0 ②phases至少1个 ③rank_tiers的min/max无重叠且连续 |
| **默认值** | 无赛季（单赛季无限模式） |
| **示例** | `{"total_duration":2592000,"phases":["prep","development","war","resolution"],"rank_tiers":[{"id":"bronze","name":"青铜","min_score":0,"max_score":999},{"id":"silver","name":"白银","min_score":1000,"max_score":2999}],"victory_conditions":["vc_territory_60","vc_capital_capture"]}` |

#### 3.10.2 阶段定义契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `season.phase_define` |
| **所属骨头** | 赛季系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，阶段ID<br>`name`：string，显示名称<br>`duration_ratio`：float，该阶段占赛季总时长的比例，(0,1]<br>`buffs`：string数组，该阶段生效的Buff ID列表<br>`restrictions`：数组，该阶段的限制列表，每个包含：{type, params}，type枚举{no_declare_war, no_migrate, no_trade, speed_limit}<br>`auto_transition`：bool，是否自动切换到下一阶段，默认true |
| **输出契约** | 框架保证：阶段按duration_ratio分配时长；阶段开始时挂载buffs；阶段期间执行restrictions；阶段结束时移除buffs |
| **校验规则** | ①id全局唯一 ②duration_ratio∈(0,1] ③所有阶段的duration_ratio之和=1.0 ④buffs中每个ID必须存在 |
| **默认值** | 1个阶段，duration_ratio=1.0，无buff和限制 |
| **示例** | `{"id":"war","name":"战争阶段","duration_ratio":0.4,"buffs":["buff_war_morale"],"restrictions":[{"type":"no_migrate","params":{}}],"auto_transition":true}` |

#### 3.10.3 重置规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `season.reset_rule` |
| **所属骨头** | 赛季系统 |
| **输入契约** | 数组，每个元素包含：<br>`target`：枚举{player_data, alliance_data, map_state, entity_all, entity_type}，重置目标<br>`target_id`：string，可选，target=entity_type时的具体类型ID<br>`method`：枚举{clear, reset_default, delete}，重置方式<br>`order`：int，重置顺序（保证依赖关系） |
| **输出契约** | 框架保证：按order从小到大依次执行重置；重置后数据符合对应组件的default值 |
| **校验规则** | ①target_id在target=entity_type时必须存在 ②order不重复 ③重置顺序保证依赖（先重置子实体再重置父实体） |
| **默认值** | 无重置（所有数据保留） |
| **示例** | `[{"target":"entity_type","target_id":"army","method":"delete","order":1},{"target":"player_data","method":"reset_default","order":2}]` |

#### 3.10.4 遗产规则契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `season.legacy_rule` |
| **所属骨头** | 赛季系统 |
| **输入契约** | 数组，每个元素包含：<br>`target`：枚举{resource, tech, buff, cosmetic, achievement}，遗产类型<br>`target_id`：string，具体ID<br>`keep_ratio`：float，保留比例，[0,1]<br>`keep_amount`：float，可选，保留固定量<br>`condition`：string，可选，保留条件表达式 |
| **输出契约** | 框架保证：重置后按遗产规则保留指定数据；保留量=min(原值×keep_ratio, keep_amount)（如果keep_amount存在） |
| **校验规则** | ①keep_ratio∈[0,1] ②keep_amount>=0 ③condition表达式语法合法 ④遗产规则与重置规则不冲突（不能重置了又保留同一条数据） |
| **默认值** | 无遗产（全部重置） |
| **示例** | `[{"target":"tech","target_id":"tech_permanent","keep_ratio":1.0},{"target":"resource","target_id":"gold","keep_ratio":0.1,"keep_amount":5000}]` |

#### 3.10.5 胜利条件契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `season.victory_condition` |
| **所属骨头** | 赛季系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，胜利条件ID<br>`name`：string，显示名称<br>`type`：枚举{territory_ratio, capital_capture, survival_time, tech_complete, alliance_score}，条件类型<br>`params`：map，条件参数（如territory_ratio需要ratio值，capital_capture需要capital数量）<br>`check_formula`：string，可选，自定义检查公式 |
| **输出契约** | 框架保证：每tick检查胜利条件；满足时触发赛季结束流程 |
| **校验规则** | ①id全局唯一 ②type在框架支持的列表中 ③params包含type所需的所有参数 |
| **默认值** | 无胜利条件（赛季仅按时间结束） |
| **示例** | `[{"id":"vc_territory_60","name":"占领60%领土","type":"territory_ratio","params":{"ratio":0.6}},{"id":"vc_capital_capture","name":"攻占王都","type":"capital_capture","params":{"count":1}}]` |

---

### 3.11 事件系统挂载点契约

#### 3.11.1 事件定义契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `event.define` |
| **所属骨头** | 事件系统 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，全局唯一<br>`name`：string，显示名称<br>`scope`：枚举{global, region, tile}，影响范围<br>`trigger`：对象，触发规则（见触发算法契约）<br>`warning_duration`：float，预警时长（秒），>=0<br>`duration`：float，持续时长（秒），>0<br>`effects`：string数组，效果Buff ID列表<br>`counter_options`：数组，可选，应对选项列表（见应对选项契约） |
| **输出契约** | 框架保证：触发后进入预警期→预警期结束生效→挂载effects的Buff→到期移除Buff |
| **校验规则** | ①id全局唯一 ②duration>0 ③effects中每个Buff ID必须存在 ④scope=region时必须有区域定义 |
| **默认值** | 无事件 |
| **示例** | `{"id":"storm","name":"暴风雨","scope":"global","trigger":{"type":"random","params":{"chance":0.1,"interval":3600}},"warning_duration":300,"duration":1800,"effects":["buff_storm_move_debuff","buff_storm_ranged_debuff"],"counter_options":[{"id":"shelter","name":"避难","cost":{"food":100},"effect":"buff_storm_shelter"}]}` |

#### 3.11.2 触发算法契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `event.trigger_algo` |
| **所属骨头** | 事件系统 |
| **输入契约** | 对象（嵌入在事件定义的trigger字段中），包含：<br>`type`：枚举{timer, random, phase, condition}，触发类型<br>`params`：map，触发参数：<br>&nbsp;&nbsp;timer：{interval: float} 定时触发<br>&nbsp;&nbsp;random：{chance: float, interval: float} 每interval秒以chance概率触发<br>&nbsp;&nbsp;phase：{phase_id: string} 指定赛季阶段触发<br>&nbsp;&nbsp;condition：{expr: string} 条件表达式满足时触发 |
| **输出契约** | 框架保证：按type和params定义的策略检测触发时机 |
| **校验规则** | ①type在框架支持的列表中 ②params包含type所需的所有参数 ③random类型的chance∈(0,1] |
| **默认值** | 无触发（事件不发生） |
| **示例** | `{"type":"random","params":{"chance":0.05,"interval":7200}}` |

#### 3.11.3 应对选项契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `event.counter_option` |
| **所属骨头** | 事件系统 |
| **输入契约** | 数组（嵌入在事件定义的counter_options字段中），每个元素包含：<br>`id`：string，选项ID<br>`name`：string，显示名称<br>`cost`：map，key=资源ID，value=float，选择该选项的消耗<br>`effect`：string，选择后获得的Buff ID |
| **输出契约** | 框架保证：玩家选择选项时扣减cost并挂载effect的Buff；资源不足时不可选择 |
| **校验规则** | ①id在事件内唯一 ②cost中每个资源ID必须存在 ③effect必须是有效的Buff ID |
| **默认值** | 无应对选项（玩家只能等待事件结束） |
| **示例** | `{"id":"shelter","name":"避难","cost":{"food":100,"wood":50},"effect":"buff_storm_shelter"}` |

---

### 3.12 网络同步挂载点契约

#### 3.12.1 Command定义契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `network.command_define` |
| **所属骨头** | 网络同步骨架 |
| **输入契约** | 数组，每个元素包含：<br>`id`：string，Command类型ID<br>`fields`：数组，每个字段包含：{name, type, required, range}<br>`server_validate`：bool，是否需要服务端校验，默认true<br>`client_predict`：bool，是否允许客户端预测，默认false<br>`authority`：枚举{server, client_owner}，执行权限 |
| **输出契约** | 框架保证：Command提交时按fields校验字段；server_validate=true时服务端校验后才执行；client_predict=true时客户端本地先执行再等服务端确认 |
| **校验规则** | ①id全局唯一 ②fields中required字段必须提供 ③client_predict=true时authority必须=client_owner |
| **默认值** | 框架提供基础Command：move/attack/gather/build/recruit |
| **示例** | `{"id":"cmd_recruit","fields":[{"name":"unit_type","type":"string","required":true},{"name":"count","type":"int","required":true,"range":[1,9999]}],"server_validate":true,"client_predict":false,"authority":"server"}` |

#### 3.12.2 同步频率契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `network.sync_frequency` |
| **所属骨头** | 网络同步骨架 |
| **输入契约** | 对象，包含：<br>`delta_interval`：float，DeltaUpdate广播间隔（秒），>0<br>`snapshot_interval`：float，FullSnapshot广播间隔（秒），>0，>=delta_interval<br>`critical_realtime`：bool，关键操作是否实时同步，默认true |
| **输出契约** | 框架保证：每delta_interval秒广播一次DeltaUpdate；每snapshot_interval秒广播一次FullSnapshot；关键操作立即同步 |
| **校验规则** | ①delta_interval>0 ②snapshot_interval>=delta_interval |
| **默认值** | `delta_interval=0.1, snapshot_interval=5.0, critical_realtime=true` |
| **示例** | `{"delta_interval":0.05,"snapshot_interval":2.0,"critical_realtime":true}` |

#### 3.12.3 预测策略契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `network.predict_strategy` |
| **所属骨头** | 网络同步骨架 |
| **输入契约** | 对象，包含：<br>`enabled`：bool，是否启用客户端预测，默认false<br>`correction_threshold`：float，校正阈值（客户端预测值与服务端确认值的最大允许偏差）<br>`correction_strategy`：枚举{snap, interpolate}，校正策略——snap=直接跳到服务端值，interpolate=插值过渡<br>`interpolate_duration`：float，interpolate策略的过渡时长（秒） |
| **输出契约** | 框架保证：偏差<=correction_threshold时不校正；偏差>correction_threshold时按correction_strategy校正 |
| **校验规则** | ①correction_threshold>=0 ②interpolate_duration>0（当correction_strategy=interpolate时） |
| **默认值** | `enabled=false` |
| **示例** | `{"enabled":true,"correction_threshold":0.5,"correction_strategy":"interpolate","interpolate_duration":0.2}` |

---

### 3.13 持久化挂载点契约

#### 3.13.1 持久化标记契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `persist.mark` |
| **所属骨头** | 持久化骨架 |
| **输入契约** | 数组，每个元素包含：<br>`entity_type`：string，实体类型ID<br>`frequency`：枚举{realtime, periodic, season_end}，持久化频率<br>`period`：float，可选，frequency=periodic时的周期（秒），>0<br>`critical_ops`：string数组，可选，该类型实体的关键操作列表（关键操作实时持久化） |
| **输出契约** | 框架保证：realtime=每次变更立即写入；periodic=每period秒批量写入；season_end=赛季结束时写入；关键操作始终实时写入 |
| **校验规则** | ①entity_type必须存在 ②frequency=periodic时period>0 |
| **默认值** | 所有实体frequency=periodic, period=60 |
| **示例** | `{"entity_type":"city","frequency":"periodic","period":30,"critical_ops":["upgrade","destroy","transfer"]}` |

#### 3.13.2 数据分区契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `persist.partition` |
| **所属骨头** | 持久化骨架 |
| **输入契约** | 对象，包含：<br>`strategy`：枚举{by_player, by_alliance, by_season, by_chunk}，分区策略<br>`shard_key`：string，分区键（实体组件中的字段名）<br>`cross_partition_query`：bool，是否允许跨分区查询，默认false |
| **输出契约** | 框架保证：数据按shard_key的值分区存储；同分区数据查询无需跨分区 |
| **校验规则** | ①shard_key必须是实体组件中存在的字段 ②cross_partition_query=true时框架提供跨分区查询接口 |
| **默认值** | `strategy=by_player, shard_key="owner_id", cross_partition_query=false` |
| **示例** | `{"strategy":"by_player","shard_key":"owner_id","cross_partition_query":true}` |

#### 3.13.3 备份恢复契约

| 要素 | 内容 |
|------|------|
| **挂载点ID** | `persist.backup` |
| **所属骨头** | 持久化骨架 |
| **输入契约** | 对象，包含：<br>`backup_interval`：float，备份间隔（秒），>0<br>`max_backups`：int，最大备份数量，>0<br>`recovery_point`：枚举{latest, specified}，恢复点选择<br>`post_recovery_validate`：bool，恢复后是否校验数据一致性，默认true |
| **输出契约** | 框架保证：每backup_interval秒创建备份；备份数量超过max_backups时删除最旧的；恢复后若post_recovery_validate=true则校验数据一致性 |
| **校验规则** | ①backup_interval>0 ②max_backups>0 |
| **默认值** | `backup_interval=3600, max_backups=24, recovery_point=latest, post_recovery_validate=true` |
| **示例** | `{"backup_interval":1800,"max_backups":48,"recovery_point":"latest","post_recovery_validate":true}` |

---

### 3.14 配置包总清单

汇总所有挂载点需要的配置文件列表：

| 配置文件 | 用途 | 必需/可选 | 依赖 |
|---------|------|----------|------|
| `game.json` | 地图规格/同步频率/全局参数/网格类型 | **必需** | 无 |
| `entity_types.json` | 实体类型定义 | **必需** | `components.json` |
| `components.json` | 组件定义 | **必需** | 无 |
| `factions.json` | 阵营/种族定义 | 可选 | `buffs.json` |
| `units.json` | 兵种定义 | 可选 | `entity_types.json`, `components.json`, `counter_matrix.json` |
| `buildings.json` | 建筑定义 | 可选 | `entity_types.json`, `components.json`, `resources.json` |
| `heroes.json` | 英雄定义 | 可选 | `entity_types.json`, `components.json`, `buffs.json` |
| `techs.json` | 科技定义 | 可选 | `buffs.json`, `resources.json` |
| `resources.json` | 资源类型定义 | **必需** | 无 |
| `terrains.json` | 地形定义 | **必需** | `movement.json`（通行性引用移动类型） |
| `counter_matrix.json` | 克制矩阵 | 可选 | 无 |
| `formulas.json` | 公式库 | **必需** | 无 |
| `buffs.json` | Buff模板 | 可选 | 无 |
| `rules.json` | 规则定义 | 可选 | `buffs.json`, `resources.json` |
| `events.json` | 事件定义 | 可选 | `buffs.json`, `resources.json` |
| `season.json` | 赛季配置 | 可选 | `buffs.json`, `rules.json` |
| `alliance.json` | 联盟配置 | 可选 | `resources.json`, `buffs.json` |
| `movement.json` | 移动类型/拦截规则/速度修正 | **必需** | `terrains.json` |
| `combat.json` | 战斗阶段/触发/结束/结果/攻城规则 | 可选 | `formulas.json`, `counter_matrix.json`, `buffs.json` |
| `economy.json` | 产出/消耗/采集/贸易/掠夺/饥荒溢出规则 | 可选 | `resources.json`, `formulas.json` |

**配置包目录结构：**

```
configs/
├── game.json              ← 地图规格/同步频率/全局参数
├── entity_types.json      ← 实体类型定义
├── components.json        ← 组件定义
├── factions.json          ← 阵营/种族定义
├── units.json             ← 兵种定义
├── buildings.json         ← 建筑定义
├── heroes.json            ← 英雄定义
├── techs.json             ← 科技定义
├── resources.json         ← 资源类型定义
├── terrains.json          ← 地形定义
├── counter_matrix.json    ← 克制矩阵
├── formulas.json          ← 公式库
├── buffs.json             ← Buff模板
├── rules.json             ← 规则定义
├── events.json            ← 事件定义
├── season.json            ← 赛季配置
├── alliance.json          ← 联盟配置
├── movement.json          ← 移动类型/拦截规则
├── combat.json            ← 战斗阶段/触发/结束/结果规则
└── economy.json           ← 产出/消耗/采集/贸易/掠夺规则
```

**配置包校验流程：**

```
1. 语法校验：每个JSON文件必须是合法JSON
2. 必需文件检查：所有标记为"必需"的文件必须存在
3. 字段类型校验：每个字段必须符合契约定义的类型
4. 字段约束校验：每个字段必须符合契约定义的约束（范围/枚举/正则）
5. 引用完整性校验：所有引用其他配置的ID必须存在（如components引用entity_types）
6. 交叉约束校验：跨配置文件的约束必须满足（如克制矩阵维度=兵种数量）
7. 循环依赖检测：规则系统不能循环触发，科技树不能循环依赖
8. 自定义校验：框架允许游戏注册自定义校验函数
```

**校验不通过时，框架拒绝加载并输出详细错误报告（文件名+字段路径+错误原因+修复建议）。**

---

> **文档状态**：前半部分（第一章到第三章）完成。后半部分将包含：第四章 骨头交互关系（系统间的调用和依赖图）、第五章 框架运行时架构（初始化顺序/tick循环/事件总线）、第六章 换皮实战案例（三国SLG/星际SLG的配置包对比）、第七章 框架扩展指南（如何新增骨头/如何新增动作类型）。