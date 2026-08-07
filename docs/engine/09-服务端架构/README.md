# 服务端架构

> 基于Go语言的微服务+DDD架构设计。14根核心模块按业务边界收拢为5个业务服务+3个基础服务，5个高频基础模块作为共享内核内嵌。

---

## 架构总览

```
                         ┌──────────────────────┐
                         │    Load Balancer      │
                         └──────────┬───────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
              ┌─────▼─────┐  ┌─────▼─────┐  ┌─────▼─────┐
              │ Gateway 1 │  │ Gateway 2 │  │ Gateway N │
              └─────┬─────┘  └─────┬─────┘  └─────┬─────┘
                    └───────────────┼───────────────┘
                                    │
                         ┌──────────▼───────────┐
                         │   Service Mesh       │
                         │   Message Bus (NATS) │
                         └──────────┬───────────┘
                                    │
          ┌────────────┬────────────┼────────────┬────────────┐
          │            │            │            │            │
    ┌─────▼────┐ ┌─────▼────┐ ┌─────▼────┐ ┌─────▼────┐ ┌─────▼────┐
    │  World   │ │  Combat  │ │ Economy  │ │  Social  │ │  Oper    │
    │ Service  │ │ Service  │ │ Service  │ │ Service  │ │ Service  │
    └─────┬────┘ └─────┬────┘ └─────┬────┘ └─────┬────┘ └─────┬────┘
          └────────────┴────────────┼────────────┴────────────┘
                                    │
                    ┌───────────────┼───────────────┐
              ┌─────▼─────┐  ┌─────▼─────┐  ┌─────▼─────┐
              │  Redis    │  │  MySQL    │  │ MongoDB   │
              │  Cluster  │  │  Shards   │  │  Cluster  │
              └───────────┘  └───────────┘  └───────────┘
```

---

## 微服务拆分策略

### 为什么14模块不能1:1拆为14个微服务

| 如果1:1拆分 | 问题 |
|------------|------|
| 实体系统独立服务 | 每次属性读写走RPC，单场战斗上百次调用，延迟不可接受 |
| Buff系统独立服务 | Buff计算在战斗中每轮触发，RPC开销远大于计算开销 |
| 事件总线独立服务 | 事件总线是通信基础设施，自己做服务=用RPC调RPC |
| 规则系统独立服务 | 规则执行需要访问实体属性，跨服务读取性能差 |

**结论**：高频基础模块作为共享内核（Go module内嵌），业务模块按业务边界收拢为微服务。

### 共享内核（Shared Kernel）

5个模块作为Go module被所有服务内嵌使用，不独立部署：

| 模块 | 内嵌原因 | 对应框架模块 | 数据访问边界 |
|------|---------|-------------|-------------|
| entity | 实体操作太频繁，RPC不可接受 | T1 实体系统 | 仅访问本服务内聚合的实体 |
| buff | Buff计算在战斗/经济中高频触发 | T2 Buff系统 | 仅访问所属实体的本地属性 |
| eventbus | 通信基础设施，分本地总线与远程适配两部分 | T4 事件总线 | 本地总线进程内同步；远程适配器连NATS |
| config | 配置加载和校验是启动期行为，内嵌即可 | T7 配置加载 | 仅读etcd/本地缓存，不写 |
| rule | 规则引擎需要访问实体属性，内嵌避免跨服务读取 | T3 规则系统 | 仅依赖entity/buff本地数据；跨服务数据由"规则上下文快照"通过事件预注入，规则执行时禁止RPC |

**共享内核版本治理**：所有服务引用同一版本的共享内核Go module（统一版本号 `pkg/vX.Y.Z`）。CI在共享内核变更时强制检查所有服务的依赖同步升级；不兼容变更走deprecation周期（旧API保留2个版本+告警），避免版本漂移导致事件Schema/接口不一致。

共享内核目录结构：

```
pkg/                          共享内核（Go module）
├── entity/                   实体系统
│   ├── entity.go             实体接口 + ID生成
│   ├── attribute.go          属性管理
│   ├── component.go          组件管理
│   └── repository.go         仓储接口
├── buff/                     Buff系统
│   ├── buff.go               Buff定义
│   ├── effect.go             效果计算
│   ├── stacking.go           叠加规则
│   └── aura.go               光环处理
├── eventbus/                 事件总线（本地与远程分离）
│   ├── local/                本地事件总线（服务内同步派发）
│   │   └── bus.go            进程内同步总线
│   ├── remote/               远程事件总线（跨服务异步）
│   │   ├── nats.go           NATS适配器
│   │   └── outbox.go         事务性发件箱（保证事件与状态原子提交）
│   └── event.go              事件类型定义
├── config/                   配置加载（运行面）
│   ├── loader.go             配置加载器
│   ├── validator.go          配置校验器
│   ├── watcher.go            etcd watch监听（接收Config Service的热更通知）
│   └── hotreload.go          热更新回调分发
├── rule/                     规则系统
│   ├── engine.go             规则引擎
│   ├── context.go            规则上下文（注入跨服务快照数据）
│   ├── condition.go          条件评估
│   └── action.go             动作执行
└── proto/                    Protobuf消息定义
    ├── request.proto          客户端请求
    ├── response.proto         服务端响应
    ├── push.proto             服务端推送
    └── event.proto            领域事件（含version字段与兼容规则）
```

### 5个业务服务 + 3个基础服务

**分类口径**：业务服务=承载SLG领域逻辑且有状态分片的服务（5个）；基础服务=接入/管理/横切关注点（3个）。

| 服务 | 类别 | 职责 | 收拢的框架模块 | 有状态 | 分片维度 |
|------|------|------|--------------|--------|---------|
| Gateway | 基础 | WebSocket接入、鉴权、路由、推送 | T5 网络同步（接入层） | 否（连接存Redis） | 无（任意实例） |
| World Service | 业务 | 地图、移动、实体位置 | B1 地图 + B2 移动 | 是 | 地图区域 |
| Combat Service | 业务 | 战斗、伤害、战报 | B3 战斗 | 是 | 战斗ID |
| Economy Service | 业务 | 资源、生产、交易 | B4 经济 | 是 | 玩家ID |
| Social Service | 业务 | 联盟、外交、玩家 | B5 联盟 | 是 | 联盟ID |
| Operation Service | 业务 | 赛季、事件、全局规则 | B6 赛季 + B7 事件 | 是 | 无（全局单例主备） |
| Config Service | 基础 | 配置管理面：写入etcd、广播热更通知 | T7 配置加载（管理面） | 否 | 无 |
| Persist Service | 基础 | 异步归档/快照/迁移（不参与在线读写） | T6 持久化（离线面） | 否 | 无 |

**Config Service 与共享内核 config 的协作**：Config Service 是配置的**管理面**（写入etcd、版本管理、热更广播）；各服务内嵌的共享内核 config 是**运行面**（watch etcd、本地缓存、reload回调）。热更链路：Config Service写入etcd → 各服务共享内核config watcher收到变更 → 校验通过后触发本地reload → 通过`config.reloaded`事件通知业务层切换。详见[服务通信.md](服务通信.md)的"配置热更时序"。

**Persist Service 与各服务 persistence 层的职责边界**：各服务的 `infrastructure/persistence` 负责**在线读写**（CQRS写侧落库、读侧查Redis）；Persist Service 负责**离线/异步数据治理**——定期快照（赛季结束快照全量数据）、冷数据归档（历史战报/过期赛季迁冷库）、数据迁移（版本升级时的Schema迁移）、备份恢复。Persist Service 通过消费领域事件和定时任务驱动，不参与在线请求路径。

### 业务收拢验证

| 业务场景 | 收拢到的服务 | 为什么不拆 |
|---------|-------------|-----------|
| 玩家移动到某地 | World Service | 地图+移动紧耦合，移动需频繁查地图 |
| 发起战斗 | Combat Service | 战斗流程+伤害+战报是一个完整业务流程 |
| 资源采集/生产 | Economy Service | 生产+消耗+仓储是资源生命周期 |
| 创建联盟/宣战 | Social Service | 联盟+外交+权限是一个聚合根 |
| 赛季阶段切换 | Operation Service | 赛季+事件+全局规则是运营节奏 |
| 配置热更（管理面） | Config Service | 配置写入etcd并广播，与运行面分离 |
| 数据归档/快照/迁移 | Persist Service | 离线数据治理，不参与在线读写路径 |

---

## 技术选型

| 层面 | 选型 | 原因 |
|------|------|------|
| 语言 | Go | 高并发、低延迟、编译型、跨平台 |
| 通信框架 | gRPC | 高性能RPC，Protobuf序列化 |
| 消息队列 | NATS | 轻延迟、轻量、支持发布订阅+请求响应 |
| 服务发现 | etcd | Go生态原生、强一致、K8s同款 |
| 配置中心 | etcd | 同上，watch机制做热更 |
| 热数据 | Redis Cluster | 低延迟缓存、分片、高可用 |
| 冷数据 | MySQL Shards | 关系型、分片、主从复制 |
| 文档数据 | MongoDB | 战报、事件历史等文档型数据 |
| 连接层 | WebSocket | 长连接、服务端推送 |
| 序列化 | Protobuf | 紧凑、跨语言、高性能 |
| 日志 | zap | Go高性能结构化日志 |
| 监控 | Prometheus + Grafana | 指标采集+可视化 |
| 链路追踪 | OpenTelemetry | 跨服务追踪 |

---

## CI强制检查项

| 检查项 | 阶段 | 说明 |
|--------|------|------|
| 零硬编码静态扫描 | PR提交 | ZeroHardcodeScanner扫描引擎代码无游戏特定名词/数值/枚举，详见[测试策略.md](测试策略.md) |
| 换皮通用性验证 | 发布前 | ReskinValidator验证至少3个题材仅靠配置差异即可运行，详见[测试策略.md](测试策略.md) |
| 配置包完整性校验 | 配置热更前 | 18个配置文件齐全+必需文件不缺失+依赖满足，详见[服务通信.md](服务通信.md) |
| 配置包兼容性校验 | 配置热更前 | 与存量数据Schema兼容，详见[服务通信.md](服务通信.md) |

---

## 增量设计新增组件（v3.3）

本次增量设计新增20个组件，补齐13个功能缺口，实现"配置驱动业务"的通用SLG服务端：

|$| 服务 | 新增组件 | 对应缺口 |
|------|---------|---------|
| World | MapInitializer、VisionService、TeleportAggregate、FormationVO | 缺口1 |
| Combat | SkillService、FormationEffectVO、ResultModifierVO | 缺口2 |
| Economy | ProductionTickScheduler、ConversionAggregate | 缺口3 |
| Social | WelfareService、PermissionChecker | 缺口4 |
| Operation | SeasonResetCoordinator、SeasonInheritService、RewardDistributor | 缺口5 |
| 共享内核 | ExtensionRegistry、ConfigQueryAPI、Validator扩展 | 缺口7、13 |
| 横切 | AdminService、ConfigConsistencyChecker、ZeroHardcodeScanner、ReskinValidator | 缺口9、11、6、12 |

所有新增组件遵循"配置驱动业务"原则——业务逻辑从ConfigQueryAPI查询配置执行，引擎代码零硬编码，换皮时仅替换配置包不改代码。

---

## 子文件索引

- [DDD设计.md](DDD设计.md) - 每个服务的DDD分层 + 聚合根 + 目录结构
- [高可用设计.md](高可用设计.md) - 无单点 + 分片 + 故障转移 + 容量规划
- [服务通信.md](服务通信.md) - 服务间通信 + 事件流转 + 数据一致性
- [安全设计.md](安全设计.md) - 鉴权 + 服务间mTLS + 防作弊 + DDoS防护
- [可观测性.md](可观测性.md) - 指标体系 + 日志规范 + 链路追踪 + 告警
- [测试策略.md](测试策略.md) - 契约测试 + 集成测试 + 混沌工程
- [部署策略.md](部署策略.md) - 滚动/金丝雀/蓝绿 + 有状态服务更新

[← 返回总入口](../README.md)