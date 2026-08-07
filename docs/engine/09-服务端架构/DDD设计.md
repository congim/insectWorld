# DDD设计

> 每个微服务是一个限界上下文（Bounded Context），内部按DDD四层架构组织。聚合根定义了业务一致性的边界。

---

## DDD四层架构

每个微服务内部统一按以下分层组织：

```
service/
├── cmd/
│   └── main.go                        入口（启动、依赖注入、优雅停机）
├── internal/                          仅限服务内部访问
│   ├── interfaces/                    接口层——对接外部
│   │   ├── grpc/                      gRPC handler（同步调用入口）
│   │   ├── consumer/                  消息消费者（异步事件入口）
│   │   └── ws/                        WebSocket handler（如有直连客户端）
│   ├── application/                   应用层——用例编排
│   │   ├── command/                   写操作（CQRS写侧）
│   │   ├── query/                     读操作（CQRS读侧）
│   │   └── dto/                       数据传输对象
│   ├── domain/                        领域层——业务核心
│   │   ├── <aggregate>/               聚合根目录
│   │   │   ├── aggregate.go           聚合根
│   │   │   ├── entity.go              实体
│   │   │   ├── value_object.go        值对象
│   │   │   ├── event.go               领域事件
│   │   │   ├── service.go             领域服务
│   │   │   └── repository.go          仓储接口（domain层定义接口）
│   │   └── ...
│   └── infrastructure/                基础设施层——技术实现
│       ├── persistence/               仓储实现（Redis/MySQL/MongoDB）
│       ├── eventbus/                  事件总线实现（NATS适配器）
│       ├── cache/                     缓存实现
│       └── external/                  外部服务客户端
├── go.mod
└── go.sum
```

### 各层职责与依赖规则

| 层 | 职责 | 可依赖（接口） | 实现注入方式 |
|----|------|---------------|-------------|
| interfaces | 协议适配、消息反序列化、调用application层 | application | 直接调用 |
| application | 用例编排、事务边界、调用domain层、发布事件 | domain（接口）+ 仓储/事件总线接口 | infrastructure通过DI注入application |
| domain | 业务核心：聚合根、实体、值对象、领域事件、仓储接口 | 无（纯业务，不依赖任何外部） | 不被注入，被application直接调用 |
| infrastructure | 仓储实现、消息队列、缓存、外部服务 | domain（实现domain层定义的接口） | 在cmd/main.go组装时注入application |

**关键规则**：
- domain层不依赖任何外部包，只定义接口（Repository、EventBus等接口在domain层声明）。
- infrastructure层实现domain层定义的接口。
- application层**不直接import infrastructure**，只依赖domain层声明的接口；infrastructure实现通过依赖注入（DI）在 `cmd/main.go` 组装时传入application。这保证application可独立单测（用mock替换infrastructure）。
- 依赖方向：`interfaces → application → domain ← infrastructure`，infrastructure在启动时反向注入application。

---

## 各服务聚合根设计

### World Service（世界服务）

```
domain/
├── map/                    地图聚合
│   ├── aggregate.go        Map聚合根（地图ID+尺寸+格子类型）
│   ├── terrain.go          Terrain值对象（地形ID+通行性+视野修正）
│   ├── cell.go             Cell值对象（坐标+地形ID+区域ID）
│   ├── initializer.go      MapInitializer（启动期按配置初始化SpaceManager）
│   └── repository.go       MapRepository接口
├── region/                 区域聚合
│   ├── aggregate.go        Region聚合根（区域ID+格子集合+属性）
│   └── repository.go       RegionRepository接口
├── movement/               移动聚合
│   ├── aggregate.go        MovementOrder聚合根（实体ID+路径+状态）
│   ├── path.go             Path值对象（坐标序列+消耗）
│   ├── formation_vo.go     FormationVO值对象（编队队形+速度修正+成员数校验）
│   └── repository.go       MovementRepository接口
├── teleport/               传送聚合
│   ├── aggregate.go        TeleportAggregate聚合根（传送条件+冷却+资源消耗）
│   └── repository.go       TeleportRepository接口
└── vision/                 视野domain service
    └── vision_service.go   VisionService（视野计算，订阅位置变更触发重算）
```

| 聚合根/Domain Service | 一致性边界 | 关键方法 |
|----------------------|-----------|---------|
| Map | 地图尺寸、格子类型不可变 | GetCell, ChangeTerrain, GetRegion, InBounds |
| Region | 区域内格子集合、区域属性 | Create, Destroy, AddCell, RemoveCell, SetAttribute |
| MovementOrder | 移动路径、移动状态、当前位置 | StartMove, UpdatePosition, Arrive, Block |
| TeleportAggregate | 传送条件、冷却、资源消耗 | Teleport, CheckCooldown |
| FormationVO | 编队队形、成员数校验、速度修正 | Validate, GetSpeedModifier |
| VisionService | 视野范围、地形修正 | Recompute |
| MapInitializer | 启动期按配置初始化空间管理器 | Init |

**Region.Create/Destroy 用例说明**：
- Create：校验格子集合在地图范围内（Map.InBounds）+ 区域ID不存在 → 聚合根创建 → 发布 `region.created` 事件
- Destroy：校验区域内无活跃实体（gRPC查询本服务各分片）→ 聚合根销毁 → 发布 `region.destroyed` 事件。Destroy前必须确认无活跃实体，否则返回 RULE_VIOLATION

**MapInitializer 启动期初始化**：服务启动时按 `game.json` 的格子类型（hex/quad/free）分支初始化 SpaceManager 子类（HexSpaceManager/QuadSpaceManager/FreeSpaceManager），从 ConfigQueryAPI 查询 `terrains.json` 注册地形定义和通行性规则。地图就绪后可接受实体入驻。

**VisionService 视野计算**：从 ConfigQueryAPI 查询 `map.map_vision_rules`（base_vision_range + 地形修正 + 视野类型），计算可见格子集合。订阅实体位置变更事件触发视野重算，发布 `vision.changed` 事件。

**TeleportAggregate 传送**：从 ConfigQueryAPI 查询 `movement.movement_types` 的传送规则（增援/迁城/传送门），gRPC查询Social校验目标是否同盟领土，gRPC调用Economy扣减传送消耗资源，记录传送冷却。

**MovementOrder 跨区域聚合归属**：World Service按区域分片，MovementOrder聚合根**始终归属于实体当前所在区域的分片**。跨区域移动时聚合根发生**迁移**：

```
实体从区域A移动到区域B：
  1. 区域A分片：MovementOrder执行到边界，发布 movement.cross_region 事件（含聚合根序列化快照）
  2. 区域A分片：本地卸载该MovementOrder（停止Tick）
  3. 区域B分片：消费事件，反序列化重建MovementOrder，继续执行剩余路径
  4. 区域B分片：成为该聚合根的新权威方，后续状态变更由B负责
```

迁移期间的一致性保证：迁移瞬间聚合根在A已卸载、在B尚未重建（<100ms），此期间对该实体的移动查询返回"迁移中"状态，客户端展示平滑过渡。迁移是幂等的——若B重建失败，A可重新接管或重发事件。

### Combat Service（战斗服务）

```
domain/
├── combat/                 战斗聚合
│   ├── aggregate.go        Combat聚合根（战斗ID+类型+参战方+状态）
│   ├── participant.go      Participant实体（参战单位+属性快照）
│   ├── round.go            Round值对象（轮次+伤害记录）
│   ├── result.go           Result值对象（胜负+存活+战利品）
│   ├── skill_service.go    SkillService domain service（技能触发判定+效果执行+冷却管理）
│   ├── formation_effect_vo.go  FormationEffectVO值对象（阵型效果，按时机应用）
│   ├── result_modifier_vo.go   ResultModifierVO值对象（战果修正器，链式应用）
│   └── repository.go       CombatRepository接口
└── report/                 战报聚合
    ├── aggregate.go        CombatReport聚合根（战报ID+完整记录）
    └── repository.go       ReportRepository接口
```

| 聚合根/Domain Service | 一致性边界 | 关键方法 |
|----------------------|-----------|---------|
| Combat | 参战方、战斗状态、当前轮次 | Start, ExecuteRound, End, Retreat |
| CombatReport | 战报不可变性 | AddRound, Finalize |
| SkillService | 技能触发条件、效果执行、冷却 | CheckTrigger, Execute |
| FormationEffectVO | 阵型效果、应用时机 | Apply |
| ResultModifierVO | 战果修正器、链式应用 | Apply |

**ExecuteRound 战斗轮次执行编排**（完整用例）：
```
ExecuteRound 编排顺序：
  1. 长时战斗每轮开始前刷新关键属性快照（攻防血量）
  2. 应用阵型效果（轮次开始时机）—— FormationEffectVO.Apply(timing=round_start)
  3. 执行轮次核心计算（伤害结算）
  4. 技能释放编排 —— SkillService.CheckTrigger判定触发条件 → SkillService.Execute执行效果（伤害/治疗/挂Buff/召唤/位移）→ 冷却管理
  5. 应用阵型效果（轮次结束时机）—— FormationEffectVO.Apply(timing=round_end)
  6. 轮数超限检查 —— 超过config.combat.max_rounds强制平局
  7. 保存聚合根 + 写Outbox + 发布 combat.round_completed 事件（含技能释放和阵型效果应用结果）
```

**SkillService 技能释放**：从 ConfigQueryAPI 查询 `combat.combat_skills` 技能定义，按触发条件（round_start/hp_below_30%等）判定是否触发，执行效果（伤害/治疗/挂Buff/召唤/位移）分发，管理技能冷却（冷却未过返回 COOLDOWN_ACTIVE）。

**FormationEffectVO 阵型效果**：从 ConfigQueryAPI 查询 `combat.combat_formation_effects`，按应用时机（开战/每轮/轮次开始/轮次结束）应用，处理效果叠加规则。

**ResultModifierVO 战果修正器**：从 ConfigQueryAPI 查询 `combat.combat_loot_rules` 的修正器，在 Combat.End 结算前链式应用修正器（首胜加成/连胜加成）。

**关键设计**：Combat聚合内缓存参战方的属性快照，战斗期间不实时跨服务读取属性。战斗结束时通过事件同步最终结果。

**属性快照刷新策略**（按战斗类型区分，避免长时战斗快照严重过期）：

| 战斗类型 | 刷新时机 | 理由 |
|---------|---------|------|
| 短时战斗（单场PVP/野怪，<30秒） | 开战时gRPC拉取一次，战斗中冻结 | 短时战斗期间外部属性变更影响小，冻结保证战斗内一致 |
| 长时战斗（联盟战/城战，>30秒） | 每轮开始前刷新关键属性快照（攻防血量） | 长时战斗中联盟科技升级/赛季Buff可能改变属性，每轮刷新保证结算公平 |
| 行军中遭遇战 | 开战时拉取，战斗中冻结 | 遭遇战短促，冻结即可 |

**快照与权威方的归属切换**：战斗期间参战方属性的**临时权威方是Combat Service**（战斗中Buff/伤害修改属性由Combat聚合内部管理）；战斗结束时通过 `combat.ended` 事件将最终属性同步回Social Service（永久权威方）恢复全局一致。详见[服务通信.md](服务通信.md)的"实体属性快照同步"。

### Economy Service（经济服务）

```
domain/
├── wallet/                 钱包聚合
│   ├── aggregate.go        PlayerWallet聚合根（玩家ID+各资源余额）
│   ├── balance.go          Balance值对象（资源类型+数量+上限）
│   └── repository.go       WalletRepository接口
├── production/             生产聚合
│   ├── aggregate.go        ProductionLine聚合根（产出者+产出规则+进度）
│   ├── tick_scheduler.go   ProductionTickScheduler（按生产间隔分组全局调度）
│   └── repository.go       ProductionRepository接口
├── trade/                  交易聚合
│   ├── aggregate.go        TradeOrder聚合根（双方+交换内容+状态，支持player_player/player_npc/alliance_alliance）
│   └── repository.go       TradeRepository接口
└── conversion/             资源转换聚合
    ├── aggregate.go        ConversionAggregate聚合根（输入资源+输出资源+转换公式）
    └── repository.go       ConversionRepository接口
```

| 聚合根/Domain Service | 一致性边界 | 关键方法 |
|----------------------|-----------|---------|
| PlayerWallet | 资源余额不可为负、不可超上限 | Produce, Consume, Transfer |
| ProductionLine | 生产进度、产出间隔 | Tick, Collect |
| ProductionTickScheduler | 按间隔分组调度、经济修正器应用 | Tick |
| TradeOrder | 交易双方同时满足条件 | Create, Confirm, Execute, Cancel |
| ConversionAggregate | 输入充足校验、原子转换 | Convert |

**PlayerWallet 修正器应用优先级**（Produce/Consume 方法扩展）：
```
产出计算顺序：
  基础产出（config.production_rules）
  → 应用科技加成（config.economy_modifiers[type=tech]）
  → 应用联盟加成（config.economy_modifiers[type=alliance]，gRPC查询Social）
  → 应用赛季加成（config.economy_modifiers[type=season]）
  → 得到最终产出
```
修正器从 ConfigQueryAPI 查询 `economy.economy_modifiers`，按优先级链式应用（科技→联盟→赛季），优先级由配置定义而非硬编码。

**overflow_behavior 仓储上限处理**（Produce 时余额超上限的分支）：
从 ConfigQueryAPI 查询 `economy.storage_rules` 的 overflow_behavior，按配置分支处理：
| overflow_behavior | 处理 |
|-------------------|------|
| discard | 超出部分丢弃，余额保持上限 |
| stop_production | 停止生产线，余额保持上限 |
| convert_to_other | 超出部分按 conversion_rules 转换为其他资源 |

**ProductionTickScheduler 生产Tick全局调度**：按生产间隔分组调度（`schedules map[Interval][]ProductionLine`），Tick时按间隔分组触发对应生产线的Tick，避免所有生产线同时Tick造成峰值。gRPC查询Social获取联盟加成应用修正器。

**ConversionAggregate 资源转换**：从 ConfigQueryAPI 查询 `economy.conversion_rules`（输入资源+输出资源+转换公式），校验输入充足后原子执行转换，发布 `economy.resource_changed` 事件。

**TradeOrder 交易类型扩展**：支持三种交易类型（trade_type）：
- player_player：玩家间交易（现有Saga模式）
- player_npc：玩家与NPC交易（从config.trade_rules的NPC交易规则执行）
- alliance_alliance：联盟间交易（gRPC查询Social外交状态校验，敌对不可贸易）

### Social Service（社交服务）

```
domain/
├── player/                 玩家聚合
│   ├── aggregate.go        Player聚合根（玩家ID+基础属性+联盟归属）
│   └── repository.go       PlayerRepository接口
├── alliance/               联盟聚合
│   ├── aggregate.go        Alliance聚合根（联盟ID+成员+等级+权限）
│   ├── member.go           Member实体（玩家ID+职位+加入时间）
│   ├── welfare_service.go  WelfareService domain service（福利触发判定+效果发放）
│   ├── permission_checker.go  PermissionChecker domain service（基于配置的权限动态校验）
│   └── repository.go       AllianceRepository接口
└── diplomacy/              外交聚合
    ├── aggregate.go        DiplomacyRelation聚合根（双方联盟+状态）
    └── repository.go       DiplomacyRepository接口
```

| 聚合根/Domain Service | 一致性边界 | 关键方法 |
|----------------------|-----------|---------|
| Player | 玩家同时只能属于一个联盟 | JoinAlliance, LeaveAlliance |
| Alliance | 成员数上限、职位权限、等级 | Create, AddMember, RemoveMember, Upgrade |
| DiplomacyRelation | 外交状态转换需双方确认 | Propose, Accept, DeclareWar |
| WelfareService | 福利触发条件、效果发放 | Distribute |
| PermissionChecker | 基于配置的职位权限映射校验 | Check |

**Alliance.AddMember 加入冷却校验**：从 ConfigQueryAPI 查询 `alliance` 的 `join_cooldown` 规则，校验玩家是否在加入冷却期内（退出后一段时间内不可再加入联盟），冷却期内返回 RULE_VIOLATION。

**Alliance.RemoveMember 退出惩罚应用**：从 ConfigQueryAPI 查询 `alliance` 的 `leave_penalty` 规则，退出时应用惩罚（扣减资源/挂惩罚Buff）。**盟主退出特殊校验**：盟主不可直接退出，返回 RULE_VIOLATION"盟主需先转让"，必须先调用 TransferLeader 转让盟主后才能退出。

**WelfareService 联盟福利发放**：从 ConfigQueryAPI 查询 `alliance` 福利规则，判定触发条件（玩家满足配置的联盟福利条件，如职位/贡献度/在线时长），效果发放（产出加成/Buff/资源）——gRPC调用Economy发放资源、调用buff挂Buff。订阅玩家满足条件事件触发发放，发布 `welfare.distributed` 事件。

**PermissionChecker 权限动态校验**：从 ConfigQueryAPI 查询 `alliance.alliance_permissions` 的职位权限映射（**非硬编码**），查询玩家在联盟中的职位，按配置的职位权限映射校验 `Check(allianceID, playerID, action) bool`。无权返回 PERMISSION_DENIED。权限规则完全由配置驱动，换皮时权限结构可变。

### Operation Service（运营服务）

```
domain/
├── season/                 赛季聚合
│   ├── aggregate.go        Season聚合根（赛季ID+当前阶段+开始时间）
│   ├── phase.go            Phase值对象（阶段ID+持续时间+效果）
│   ├── reset_coordinator.go   SeasonResetCoordinator（跨服务重置协调）
│   ├── inherit_service.go     SeasonInheritService（跨赛季继承）
│   ├── reward_distributor.go  RewardDistributor（奖励分批发放）
│   └── repository.go       SeasonRepository接口
├── scoring/                积分聚合
│   ├── aggregate.go        ScoreBoard聚合根（排名+积分+赛季）
│   └── repository.go       ScoreBoardRepository接口
└── gameevent/              游戏事件聚合
    ├── aggregate.go        GameEvent聚合根（事件ID+触发条件+效果）
    └── repository.go       GameEventRepository接口
```

| 聚合根/Domain Service | 一致性边界 | 关键方法 |
|----------------------|-----------|---------|
| Season | 阶段单向流转、不可回退 | StartPhase, TransitionPhase, EndSeason, Reset |
| ScoreBoard | 积分不可篡改、排名按积分排序 | AddScore, GetRank, Reset |
| GameEvent | 事件触发条件、效果执行 | Trigger, Execute, Expire |
| SeasonResetCoordinator | 跨服务重置范围、保留列表 | CoordinateReset |
| SeasonInheritService | 跨赛季继承规则、继承公式 | Inherit |
| RewardDistributor | 奖励规则、分批发放 | Distribute |

**Season.Reset 跨服务协调流程**（完整用例）：
```
赛季重置编排：
  1. SeasonResetCoordinator 从 ConfigQueryAPI 查询 season.reset_rules（重置范围+保留列表）
  2. 发布 season.ended 事件，各服务订阅按范围重置：
     ├─ World：重置地图实体位置（保留列表中的实体不动）
     ├─ Economy：重置资源（保留列表中的资源不动）
     ├─ Social：重置联盟状态（保留联盟等级/外交）
     └─ Combat：清空进行中战斗（按平局结算+退还）
  3. SeasonInheritService 从 ConfigQueryAPI 查询 season.inherit_rules，按继承公式将上赛季数据写入新赛季（如继承部分等级/资源/科技）
  4. RewardDistributor 从 ConfigQueryAPI 查询 season.reward_rules，按排名分批发放奖励（避免峰值）
  5. 发布 inherit.completed、reward.distributed 事件
```

**SeasonResetCoordinator**：重置范围和保留列表**由配置驱动**（`season.reset_rules` 的 reset_scope + retain_list），而非硬编码哪些重置哪些保留。换皮时重置策略可变。

**SeasonInheritService**：从 ConfigQueryAPI 查询 `season.inherit_rules`（继承项+继承公式），按公式将上赛季数据写入新赛季。继承规则由配置定义（如继承70%科技等级、继承50%资源）。

**RewardDistributor**：从 ConfigQueryAPI 查询 `season.reward_rules`（奖励档位+奖励内容），按排名分批发放奖励（每批100名玩家，避免同时发放造成Economy峰值）。

---

## 配置查询驱动业务机制

> 业务服务执行业务逻辑时，从共享内核config查询配置驱动业务规则。配置查询走本地缓存（不读etcd），保证查询延迟<1ms。这是"引擎通用机制（换皮不改代码）+ 配置注入内容（换皮改配置）"的实现基础。

### ConfigQueryAPI 统一接口

共享内核提供 `ConfigQueryAPI` Go接口，所有业务服务通过此接口查询配置：

```go
// pkg/config/query_api.go
package config

type ConfigQueryAPI interface {
    // 统一查询入口
    QueryByExtensionPoint(extPointID string) (interface{}, error)
    
    // 常用查询快捷方法（按配置包分组）
    // —— 空间配置 ——
    GetTerrain(terrainID string) (Terrain, error)
    GetMovementType(typeID string) (MovementType, error)
    // —— 战斗配置 ——
    GetCombatType(typeID string) (CombatType, error)
    GetCombatSkill(skillID string) (CombatSkill, error)
    GetFormationEffect(effectID string) (FormationEffect, error)
    GetDamageFormula(formulaID string) (DamageFormula, error)
    GetCounterMatrix() (CounterMatrix, error)
    GetMaxRounds(combatType string) (int, error)
    // —— 经济配置 ——
    GetProductionRule(ruleID string) (ProductionRule, error)
    GetTradeRule(ruleID string) (TradeRule, error)
    GetConversionRule(ruleID string) (ConversionRule, error)
    GetStorageRule(resourceType string) (StorageRule, error)
    GetEconomyModifiers(modifierType string) ([]Modifier, error)
    // —— 社交配置 ——
    GetAlliancePermissions() (PermissionMap, error)
    GetAllianceWelfare(welfareID string) (WelfareRule, error)
    // —— 运营配置 ——
    GetSeasonPhases() ([]Phase, error)
    GetSeasonTransitionRules() ([]TransitionRule, error)
    GetSeasonResetRules() (ResetRules, error)
    GetSeasonInheritRules() ([]InheritRule, error)
    GetSeasonRewards() ([]RewardTier, error)
    GetSeasonScoringRules() (ScoringRules, error)
    // —— 事件配置 ——
    GetEventTypes() ([]EventType, error)
}
```

### 配置查询路径

```
业务服务执行业务逻辑
  │
  ├─ 调用 ConfigQueryAPI.GetXXX()
  │
  ├─ ConfigQueryAPI 从 ExtensionRegistry 查询编译后配置
  │
  ├─ ExtensionRegistry 返回本地内存中编译后的配置（启动期加载/热更期替换）
  │
  └─ 返回配置给业务层（延迟<1ms，纯内存查询，不读etcd/不读文件）
```

### 各业务服务配置查询接口

| 服务 | 查询的配置 | 驱动的业务规则 |
|------|-----------|--------------|
| World | GetTerrain/GetMovementType | 地形通行性、移动消耗、视野修正、传送规则、编队规则 |
| Combat | GetCombatType/GetCombatSkill/GetFormationEffect/GetDamageFormula/GetCounterMatrix/GetMaxRounds | 战斗流程、技能释放、阵型效果、伤害计算、兵种克制、轮数上限 |
| Economy | GetProductionRule/GetTradeRule/GetConversionRule/GetStorageRule/GetEconomyModifiers | 生产规则、交易规则、资源转换、仓储上限、经济修正器 |
| Social | GetAlliancePermissions/GetAllianceWelfare | 权限校验、福利发放、加入冷却、退出惩罚 |
| Operation | GetSeasonPhases/GetSeasonTransitionRules/GetSeasonResetRules/GetSeasonInheritRules/GetSeasonRewards/GetSeasonScoringRules/GetEventTypes | 赛季阶段、阶段转换、重置范围、继承规则、奖励档位、积分规则、事件触发 |

### 配置参数化业务规则机制

业务执行时从 ConfigQueryAPI 查询配置，按配置定义的规则执行，**业务代码不含任何游戏特定数值/名词**：

```go
// 示例：战斗伤害计算（引擎通用代码，不含"步兵""弓箭手"等游戏特定名词）
func (c *Combat) ExecuteRound() {
    skill := configAPI.GetCombatSkill(c.triggeredSkillID)        // 从配置查询技能定义
    formula := configAPI.GetDamageFormula(skill.DamageFormulaID)  // 从配置查询伤害公式
    counter := configAPI.GetCounterMatrix()                       // 从配置查询克制矩阵
    damage := formula.Calculate(attacker, defender, counter)      // 按配置公式计算
    c.applyDamage(damage)
}
```

换皮时：代码不变（上述ExecuteRound逻辑通用），只替换配置包（combat_skills.json/formulas.json/counter_matrix.json），GetCombatSkill/GetDamageFormula/GetCounterMatrix返回新配置，伤害计算自动按新规则执行。

---

## CQRS读写分离

每个服务采用CQRS模式，写操作走聚合根，读操作走优化读模型：

```
application/
├── command/                写操作（走聚合根，保证业务一致性）
│   ├── move_entity.go      MoveEntityCommand → MovementOrder.StartMove
│   ├── start_combat.go     StartCombatCommand → Combat.Start
│   └── produce_resource.go ProduceResourceCommand → ProductionLine.Tick
├── query/                  读操作（走读模型，不走聚合根）
│   ├── get_map_view.go     GetMapViewQuery → 读Redis缓存
│   ├── get_combat_state.go GetCombatStateQuery → 读Redis缓存
│   └── get_ranking.go      GetRankingQuery → 读Redis有序集合
└── dto/
```

写操作流程：Command → 加载聚合根 → 调用聚合根方法 → 保存聚合根（同事务写Outbox表） → 后台relay投递事件 → 事件触发读模型更新

读操作流程：Query → 直接读读模型（Redis/物化视图） → 返回DTO

**读模型更新链路**（解决"读模型何时更新"问题）：

| 更新策略 | 适用场景 | 一致性保证 |
|---------|---------|-----------|
| 写后同步更新 | 写操作聚合根保存后，同事务内更新读模型 | 强一致——读自己的写立即可见 |
| 事件驱动更新 | 领域事件被读模型投影器订阅，异步更新 | 最终一致——读模型滞后<100ms |
| 定时重建 | 排行榜等聚合读模型，定时全量重算 | 周期一致——按刷新周期 |

**默认策略**：玩家自身状态（自己的资源/兵种/位置）走"写后同步更新"，保证玩家操作后立即可见结果（符合"服务端权威"体验）；他人可见状态（排行榜/联盟成员列表/地图视野）走"事件驱动更新"，容忍短暂滞后。

**投影器（Projection）目录**：

```
infrastructure/
└── projection/              读模型投影器
    ├── map_view.go          订阅movement事件 → 更新Redis地图视野
    ├── combat_state.go      订阅combat事件 → 更新Redis战斗状态
    ├── ranking.go           订阅score事件 → 更新Redis有序集合
    └── wallet_view.go       订阅economy事件 → 更新Redis钱包视图
```

投影器与application层解耦，作为独立消费者goroutine运行，订阅NATS事件更新读模型。

---

## 领域事件设计

领域事件在聚合根内产生，由application层写入Outbox表，后台relay投递到事件总线：

```go
// domain/movement/event.go
package movement

type ArrivedEvent struct {
    EntityID   string
    Position   Position
    PathTaken  []Position
    Timestamp  int64
}

// application/command/move_entity.go
func (h *MoveEntityHandler) Handle(ctx context.Context, cmd MoveEntityCommand) error {
    // 1. 开启事务（聚合根状态与Outbox同事务）
    tx := h.txMgr.Begin(ctx)
    defer tx.Rollback()
    
    order, err := h.repo.Load(tx, cmd.OrderID)
    if err != nil {
        return err
    }
    
    event, err := order.Arrive(cmd.Position)
    if err != nil {
        return err
    }
    
    // 2. 同事务保存聚合根 + 写Outbox表（保证原子性）
    if err := h.repo.Save(tx, order); err != nil {
        return err
    }
    if err := h.outbox.Append(tx, event); err != nil {
        return err
    }
    
    // 3. 提交事务（状态与事件一起落库，不会丢事件）
    return tx.Commit()
}

// infrastructure/eventbus/remote/relay.go
// 后台relay goroutine：轮询Outbox表，投递到NATS，成功后标记已投递
func (r *OutboxRelay) Run(ctx context.Context) {
    for {
        events := r.outbox.FetchPending(ctx, batchSize)
        for _, e := range events {
            if err := r.nats.Publish(ctx, e); err == nil {
                r.outbox.MarkPublished(ctx, e.ID)
            }
            // 失败则下轮重试（至少一次投递）
        }
    }
}
```

**为什么用Outbox而非直接Publish**：直接 `repo.Save` 后 `eventBus.Publish`，若Save成功但Publish失败（NATS抖动/进程崩溃），事件丢失导致状态与事件不一致（如战斗结束事件丢→战利品不发放）。Outbox将事件与聚合根状态放在**同一数据库事务**中提交，保证"要么都成功要么都失败"；后台relay从Outbox表轮询投递到NATS，实现**至少一次**投递；消费者幂等去重达到**效果上一次**。

事件投递后：
1. 本服务内其他聚合可通过本地事件总线订阅（relay同时投递本地）
2. 跨服务消费者通过NATS订阅
3. 投递失败时relay重试，超过阈值告警（事件积压监控见[可观测性.md](可观测性.md)）

---

[← 返回服务端架构](README.md) | [← 返回总入口](../README.md)