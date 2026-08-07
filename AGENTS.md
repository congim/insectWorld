# 全局开发规范（AGENTS.md）

> 本文件是通用SLG服务端项目的**全局设定**，所有编码任务（含 `.codeartsdoer/specs/server_biz_spec/tasks.md` 中的任务组 1、3-8）必须遵守。本规范与 `docs/engine/09-服务端架构/DDD设计.md` 的微服务+DDD四层架构保持一致，不冲突。任何与本规范冲突的代码不得合入主干。

---

## 0. 规范总则

### 0.1 适用项目范围
本规范适用于通用SLG服务端全部Go代码仓库，包括：
- 5个业务服务：World / Combat / Economy / Social / Operation
- 3个基础服务：Gateway / Config / Persist
- 5个共享内核模块：entity / buff / eventbus / config / rule（Go module 内嵌）
- 运营管理面 AdminService 及其 Protobuf 契约

### 0.2 规范强制级别
- **MUST**：必须执行，违反即CI阻断合入
- **SHOULD**：强烈建议，违反需在PR中说明豁免理由
- **MAY**：推荐做法，不强制

### 0.3 检查机制总览
| 检查方式 | 工具/载体 | 触发时机 |
|---------|----------|---------|
| CI静态检查 | gofmt / goimports / golangci-lint / 自定义扫描脚本 | 每次PR提交 |
| 代码审查 | PR Review Checklist（见本文第10章） | PR评审阶段 |
| 静态扫描 | ZeroHardcodeScanner / 表名扫描器 / 字段注释扫描器 | CI流水线 |
| 单元测试 | go test -race -cover | CI流水线 |

---

## 1. 宏定义规范到文件内（MUST）

### 1.1 具体要求
1. **常量/宏定义必须就近归属**：常量定义必须放在其归属的文件内，禁止散落到与使用方无关的"大杂烩"常量文件。
2. **按聚合根归属**：领域常量定义在对应聚合根目录的 `consts.go` 或归属文件顶部；跨聚合共享的常量定义在 `domain/<service>/consts.go`；服务级常量定义在 `internal/<service>/consts.go`。
3. **扩展点ID常量集中**：扩展点ID（如 `combat.damage_formulas`、`economy.production_rules`）统一在 `pkg/config/extension_registry.go` 的常量块定义，对应 tasks.md 1.1 节。
4. **错误码常量集中**：错误码（如 `INVALID_PARAMS`、`RULE_VIOLATION`、`RESOURCE_INSUFFICIENT`、`COOLDOWN_ACTIVE`）在 `internal/<service>/errors/codes.go` 集中定义，使用 `iota` 或显式整型赋值。
5. **禁止魔法数字**：业务逻辑中不得出现裸数值常量，必须用具名常量替代。框架基础设施白名单除外（如默认超时、默认重试次数，需附注释说明）。
6. **常量命名**：使用 `CamelCase` 或 `PascalCase`，全大写加下划线（`MAX_ROUNDS`）仅用于与外部协议强对齐的场景。

### 1.2 适用范围
- 全部 `.go` 源文件（domain / application / interfaces / infrastructure / pkg 共享内核）
- Protobuf 生成的代码除外（由 protoc 生成，不手工编辑）

### 1.3 检查方式
- **CI静态检查**：自定义Go AST扫描脚本，检测裸数值常量出现在业务逻辑分支中
- **代码审查**：PR Review 检查常量是否就近归属、是否出现"大杂烩"常量文件
- **静态扫描**：ZeroHardcodeScanner 白名单外的数值常量告警

---

## 2. 数据库表命名规范——t_ 前缀（MUST）

### 2.1 具体要求
1. **表名必须以 `t_` 前缀**：所有数据库表名必须以小写 `t_` 开头，后接蛇形命名（snake_case）的业务实体名。
   - 正例：`t_player`、`t_alliance`、`t_combat`、`t_movement_order`、`t_resource_balance`、`t_season`、`t_config_version`
   - 反例：`player`、`Player`、`players`、`tbl_player`、`T_PLAYER`
2. **命名用单数业务实体名**：表名用业务实体的单数形式（`t_player` 而非 `t_players`），与DDD聚合根命名对应。
3. **关联表命名**：多对多关联表用 `t_<实体A>_<实体B>_rel`，如 `t_player_alliance_rel`、`t_alliance_diplomacy_rel`。
4. **分片表命名**：分片表用 `t_<实体>_<分片号>`，如 `t_player_00`、`t_player_01`，分片号两位补零。
5. **索引命名**：普通索引 `idx_<表名去t_>_<字段>`，唯一索引 `uk_<表名去t_>_<字段>`，外键 `fk_<表名去t_>_<字段>`。
6. **ORM/SQL中不得硬编码表名绕过前缀**：仓储实现（infrastructure/persistence）中的表名必须通过集中定义的表名常量引用，常量块统一以 `t_` 开头。

### 2.2 适用范围
- MySQL 冷数据表（玩家档案、联盟数据、赛季快照、配置版本历史）
- MongoDB collection 不强制 `t_` 前缀（Mongo 用文档模型，命名遵循业务实体蛇形即可），但若Mongo collection与MySQL表语义对应则保持一致
- Redis key 不适用本规范（Redis key 命名见可观测性.md的key规范）
- 临时表、迁移脚本中的表同样适用

### 2.3 检查方式
- **CI静态检查**：表名扫描器扫描 `infrastructure/persistence` 下全部SQL/ORM表名定义，正则 `^t_[a-z][a-z0-9_]*$` 校验
- **代码审查**：PR Review 检查新增表是否带 `t_` 前缀
- **静态扫描**：DDL迁移脚本扫描，建表语句表名校验

---

## 3. DDD架构规范（MUST）

### 3.1 具体要求
本规范与 `docs/engine/09-服务端架构/DDD设计.md` 的DDD四层架构完全一致，不得偏离。

1. **四层架构**：每个微服务内部统一按 `interfaces / application / domain / infrastructure` 四层组织，目录结构严格遵循 DDD设计.md 第9-40行定义的布局。
2. **依赖方向**：`interfaces → application → domain ← infrastructure`，infrastructure 在 `cmd/main.go` 启动时反向注入 application。依赖方向不得反转。
3. **domain层零外部依赖**：domain层不依赖任何外部包（不import infrastructure、不import第三方ORM/消息队列SDK），只定义接口（Repository、EventBus 等接口在 domain 层声明）。
4. **application层不直接import infrastructure**：application 只依赖 domain 层声明的接口；infrastructure 实现通过依赖注入（DI）在 `cmd/main.go` 组装时传入 application。保证 application 可独立单测（用 mock 替换 infrastructure）。
5. **聚合根一致性边界**：聚合根内的状态变更必须保持强一致，跨聚合根通过领域事件最终一致。聚合根方法不得直接调用其他聚合根的方法，只能通过领域事件或 application 层编排。
6. **仓储接口归属**：Repository 接口在 domain 层定义，实现在 infrastructure/persistence 层。
7. **领域事件Outbox投递**：聚合根状态变更产生领域事件，通过 Outbox 表 + 事件总线可靠投递，保证事件不丢不重（对应 spec.md 4.2 可靠性要求）。
8. **共享内核版本治理**：所有服务引用同一版本的共享内核 Go module，CI 强制检查依赖同步升级（对应 spec.md 4.4 可维护性要求5）。
9. **CQRS读写分离**：application/command 为写侧，application/query 为读侧，读模型通过投影器异步更新。

### 3.2 适用范围
- 全部5个业务服务 + 3个基础服务的内部代码组织
- 共享内核模块（pkg 下的 entity/buff/eventbus/config/rule）遵循module边界，不按四层划分

### 3.3 检查方式
- **CI静态检查**：自定义Go import图扫描脚本，校验依赖方向（domain层import列表不得包含infrastructure/第三方SDK）
- **CI静态检查**：golangci-lint 启用 `depguard` 规则，禁止 application 直接 import infrastructure
- **代码审查**：PR Review 检查聚合根是否跨边界直接调用、Repository 接口是否在 domain 层定义
- **单元测试**：application 层必须可用 mock infrastructure 独立单测

---

## 4. 不过度设计原则（SHOULD）

### 4.1 具体要求
1. **满足当前需求即可**：抽象层级与泛化程度以满足 spec.md 当前需求为准，不为"未来可能的需求"预先抽象。对应 spec.md 1.4 职责边界——服务端是通用引擎，但通用性由配置驱动实现，不由代码过度抽象实现。
2. **禁止过度泛化**：不使用 `interface{}` / `any` 作为通用容器除非确有必要（如 ExtensionRegistry 存放异构配置）；强类型优先。
3. **禁止过度抽象**：不为单一实现创建接口。接口在存在多个实现或需要mock测试时才抽象。domain 层 Repository 接口例外（DDD要求）。
4. **禁止过度配置化**：配置注入的内容限于 spec.md 1.2 定义的18个配置文件覆盖的扩展点；不在配置中放代码逻辑。
5. **YAGNI原则**：不实现spec.md未要求的功能。tasks.md 中标注的功能缺口才实现，不自行扩大范围。
6. **简单优先**：在简单方案与"优雅但复杂"方案间选简单方案。能用值对象解决的不用聚合根，能用domain service解决的不引入新聚合根。
7. **避免过早优化**：不预先做性能优化除非spec.md 4.1 性能红线要求。先正确再优化。

### 4.2 适用范围
- 全部业务代码设计决策
- spec.md 已明确的功能按spec实现，未明确的不得自行扩展

### 4.3 检查方式
- **代码审查**：PR Review 重点检查抽象层级是否过深、是否为单一实现创建接口、是否引入未要求的泛化
- **设计评审**：新增聚合根/Domain Service 需在 design.md 中论证必要性

---

## 5. 注释语言规范——中文（MUST）

### 5.1 具体要求
1. **代码注释统一用中文**：所有注释（包注释、类型注释、方法注释、字段注释、行内注释、文件头注释）必须使用简体中文。
2. **包注释**：每个包必须有 `// Package <pkgname> 中文说明` 的包注释，说明该包的业务职责。
3. **导出标识符必须有中文注释**：所有导出的函数、类型、常量、变量必须有中文注释，以该标识符名称开头。
   - 正例：`// PlayerWallet 玩家钱包聚合根，维护各资源余额的一致性边界。`
   - 反例：`// PlayerWallet player wallet aggregate`
4. **注释内容说明"为什么"而非"是什么"**：避免复述代码字面含义，应说明业务意图、设计决策原因、约束来源。
5. **TODO/FIXME 用中文描述**：`// TODO 后续支持跨赛季继承公式扩展` 而非 `// TODO support inherit`。
6. **Protobuf 注释**：proto 文件中的注释也用中文，生成的Go代码注释由 protoc 生成可保留英文，但手写补充部分用中文。
7. **错误信息文案**：返回给客户端的错误消息文案用中文（如"已有移动进行中"），错误码本身用英文常量（如 `RULE_VIOLATION`）。

### 5.2 适用范围
- 全部手写 `.go` 源文件
- Protobuf `.proto` 文件
- DDL迁移脚本注释
- 生成的代码（protoc/gen 产物）除外

### 5.3 检查方式
- **CI静态检查**：自定义扫描脚本检测导出标识符是否缺少中文注释、注释是否含非中文（除标准库引用、技术术语缩写外）
- **代码审查**：PR Review 检查注释语言与内容质量
- **静态扫描**：golangci-lint 启用 `golint` / `revive` 检查导出标识符注释存在性

---

## 6. 结构体字段注释规范（MUST）

### 6.1 具体要求
1. **结构体每个字段必须有中文注释**：所有 `struct` 的每个字段必须紧跟行内中文注释，说明该字段的业务含义。
   - 正例：
     ```go
     type MovementOrder struct {
         orderID    int64  // 移动订单ID，全局唯一，由雪花算法生成
         entityID   int64  // 移动实体ID，对应World中的实体
         path       Path   // 移动路径值对象，含坐标序列与移动力消耗
         status     int    // 移动状态：1=待开始 2=移动中 3=已到达 4=已阻挡 5=迁移中
         startTime  int64  // 移动开始时间戳（毫秒）
     }
     ```
   - 反例：字段无注释，或注释为英文，或注释仅复述字段名
2. **字段注释说明业务含义**：注释应说明字段在业务中的含义、取值范围、单位、来源（配置/计算/外部传入），而非仅复述字段名。
3. **枚举型字段注释列出取值映射**：用整型表示枚举的字段（见第8章），注释中必须列出全部取值到含义的映射。
4. **嵌套结构体递归适用**：嵌套的匿名结构体字段同样必须有注释。
5. **Protobuf 生成结构体**：protoc 生成的结构体字段注释由proto文件注释生成，proto文件中每个字段必须有中文注释（对应第5章）。
6. **数据库模型结构体**：ORM模型结构体（infrastructure/persistence）字段注释需对应数据库列的业务含义，与表名 `t_` 前缀规范协同。

### 6.2 适用范围
- 全部 `struct` 定义（domain 实体/值对象、application DTO、infrastructure 持久化模型、interfaces 请求响应）
- Protobuf 生成的 message 结构体（通过 proto 注释覆盖）

### 6.3 检查方式
- **CI静态检查**：Go AST 扫描脚本，遍历所有 struct 的字段，校验每个字段是否存在行内注释且注释为中文
- **代码审查**：PR Review 检查新增/修改结构体字段注释完整性
- **静态扫描**：golangci-lint 自定义 linter 或 `fieldalignment` 配合自定义规则

---

## 7. 业务日志规范——全面（MUST）

### 7.1 具体要求
1. **关键业务操作必须有日志**：以下场景必须记录日志，日志内容要全面（含上下文信息）：
   - 聚合根状态变更（创建/修改/删除）
   - 领域事件发布与消费
   - 跨服务 gRPC 调用（请求/响应/耗时/结果）
   - 配置热更触发与应用
   - 运营管理操作（配置热更/赛季管理/玩家封禁，对应 spec.md 4.3 安全性要求5的审计日志）
   - 错误与异常分支
   - 战斗结算、资源变更、赛季阶段切换等核心业务节点
2. **结构化日志**：使用 `zap` 结构化日志（对应 spec.md 4.4 可维护性要求3），禁止 `fmt.Println` / `log.Printf` 进入业务代码。
3. **日志必须包含关联字段**：每条业务日志必须携带 `request_id` / `trace_id` / `player_id`（如涉及玩家）用于分布式链路关联。
4. **日志级别规范**：
   - `Debug`：详细调试信息，生产关闭
   - `Info`：关键业务节点正常流转
   - `Warn`：可恢复的异常、降级、重试
   - `Error`：业务错误、外部调用失败（不panic）
   - `DPanic`/`Panic`：不可恢复的编程错误
5. **日志内容全面**：业务日志应包含足以定位问题的上下文：操作类型、操作主体、操作对象、关键参数、结果、耗时。禁止仅记录"操作完成"而无上下文。
6. **敏感信息脱敏**：玩家账号、支付信息、Token 等敏感字段不得明文入日志（对应 spec.md 4.3 安全性要求6）。
7. **日志不泄密配置数值**：通用引擎日志不得出现具体游戏数值（对应 ZeroHardcodeScanner 规则）。
8. **审计日志独立**：运营管理操作的审计日志独立存储，含操作人、操作时间、操作内容、操作结果、操作前后值。

### 7.2 适用范围
- 全部业务服务、基础服务、共享内核
- infrastructure 层的技术日志（连接、重连、池化）同样适用结构化日志

### 7.3 检查方式
- **CI静态检查**：扫描脚本检测 `fmt.Println` / `log.Printf` 在业务代码中的使用、检测业务函数是否缺少日志调用
- **代码审查**：PR Review 检查关键业务节点日志是否全面、是否携带关联字段、是否脱敏
- **静态扫描**：golangci-lint 启用 `forbidigo` 禁止 `fmt.Println` 在非main包使用
- **运行时验证**：测试用例验证关键操作产生预期日志（可选）

---

## 8. 类型选择规范——优先整型（MUST）

### 8.1 具体要求
1. **能用整型就用整型**：在整型可表达业务语义的场景，优先使用整型（`int` / `int32` / `int64`），而非浮点型或字符串。
2. **资源数量用 `int64`**：资源余额、数量、积分等用 `int64` 而非 `float64`。SLG资源数量为整数，用浮点会引入精度问题。
3. **状态/枚举用 `int`**：状态、类型、枚举用 `int` 而非 `string`。如移动状态、战斗状态、外交状态、赛季阶段、操作类型。枚举值用常量定义（对应第1章）。
4. **ID用 `int64`**：实体ID、订单ID、联盟ID、赛季ID等用 `int64`（雪花算法生成），而非字符串UUID（除非外部系统强制UUID）。
5. **时间戳用 `int64` 毫秒**：时间戳统一用 `int64` 毫秒级Unix时间戳，而非 `time.Time` 在持久化层（内存中可用 `time.Time`，持久化/传输用 `int64` 毫秒）。
6. **坐标用 `int32`/`int64`**：地图坐标用整型，不用浮点（SLG地图为格子坐标，整数即可）。
7. **金额用 `int64` 分**：若涉及金额，用 `int64` 分为单位，不用 `float64` 元。
8. **配置数值例外**：配置注入的数值（如伤害系数、移动力消耗修正）可为 `float64`，因为配置公式可能含小数。但配置加载后参与运算的中间结果按需选择，最终落库的业务量（如伤害值、资源量）用整型。
9. **Protobuf 类型对齐**：proto 字段类型选择与Go类型对齐，资源数量用 `int64`，状态用 `int32`，不用 `string` 表达枚举。
10. **禁止用 `float64` 表达离散业务量**：资源数量、战斗轮次、成员数、建筑等级等离散量禁止用浮点型。

### 8.2 适用范围
- 全部业务数据结构（domain 实体/值对象、DTO、持久化模型、Protobuf message）
- 配置加载后的配置数值字段例外（可为 float64）

### 8.3 检查方式
- **CI静态检查**：Go AST 扫描脚本，检测业务结构体中资源/数量/状态/ID字段是否使用整型
- **代码审查**：PR Review 检查新增字段类型选择是否符合优先整型原则
- **静态扫描**：Protobuf 扫描，检测枚举字段是否用 `int32` 而非 `string`

---

## 9. Go语言开发规范（MUST）

### 9.1 具体要求
本规范遵循Go语言官方规范：`gofmt` / `goimports` / `golint` / `Effective Go` / `Go Code Review Comments`。

1. **代码格式化**：全部 `.go` 文件必须通过 `gofmt` 格式化，import 必须通过 `goimports` 整理（分组：标准库 / 第三方 / 内部包）。
2. **命名规范**：
   - 包名：小写单词，不用下划线/驼峰，如 `movement`、`combat`、`wallet`
   - 导出标识符：`PascalCase`，如 `MovementOrder`、`StartMove`
   - 未导出标识符：`camelCase`，如 `orderID`、`startTime`
   - 接口名：单方法接口用方法名+er（`Reader`/`Repository`），多方法接口用业务名词
   - 常量名：`PascalCase` 或 `CamelCase`，不强制全大写下划线（见第1章）
3. **错误处理**：
   - 错误必须检查，不得忽略（禁止 `_ = err`）
   - 错误向上传递时包裹上下文：`fmt.Errorf("移动订单创建失败: %w", err)`
   - 业务错误用定义的错误码常量（见第1章），不得裸返回 `errors.New`
   - panic 仅用于不可恢复的编程错误，业务错误用 error 返回
4. **包结构**：
   - 每个包有明确的业务职责，单一职责
   - 禁止 `utils` / `common` / `helpers` 大杂烩包（通用函数按归属放入对应包）
   - 包名与目录名一致
5. **receiver 命名**：方法 receiver 用类型首字母小写，如 `func (m *MovementOrder) StartMove(...)`，不用 `this` / `self`。
6. **context 传递**：可能阻塞的函数第一个参数必须是 `context.Context`，如 `func (r *Repository) Find(ctx context.Context, id int64) (*Aggregate, error)`。
7. **goroutine 安全**：启动 goroutine 必须有明确的退出机制（context cancel / WaitGroup），禁止裸 `go func()` 无退出控制。
8. **接口小化**：接口定义在消费方而非实现方（Go惯例），接口尽量小。domain 层 Repository 接口例外（DDD要求在domain层定义）。
9. **零值可用**：导出结构体尽量保证零值可用，需要初始化的提供 `NewXXX()` 构造函数。
10. **测试规范**：
    - 单元测试文件 `xxx_test.go` 与被测文件同包
    - 测试函数名 `TestXxx`，用 `t.Run` 子测试组织
    - 用 `testify` 或标准库断言，禁止手写 `if got != want { t.Fatal }` 大量重复
    - application 层测试用 mock infrastructure（对应第3章DDD可测性）
    - 并发测试用 `go test -race`
11. **依赖管理**：Go module 统一管理，`go.mod` / `go.sum` 提交版本控制，共享内核版本同步（对应第3章）。
12. **未使用代码**：不得提交未使用的代码（变量、函数、import），CI 检查 `unused` / `deadcode`。

### 9.2 适用范围
- 全部 `.go` 源文件与测试文件
- Go module 管理的全部依赖

### 9.3 检查方式
- **CI静态检查**：`gofmt -l` 必须无输出、`goimports -l` 必须无输出
- **CI静态检查**：`golangci-lint run` 启用 `errcheck` / `golint` / `revive` / `unused` / `gosimple` / `staticcheck` / `ineffassign` / `unparam` / `go vet`
- **CI静态检查**：`go test -race -cover` 覆盖率门槛（application/domain 层 > 80%）
- **代码审查**：PR Review 按 Go Code Review Comments 检查

---

## 10. PR Review Checklist（代码审查清单）

代码审查时逐项核对，违反 MUST 项即要求修改：

- [ ] **宏定义**：常量就近归属，无散落，无魔法数字（第1章）
- [ ] **表名**：新增/修改表带 `t_` 前缀，蛇形单数（第2章）
- [ ] **DDD分层**：依赖方向正确，domain 零外部依赖，application 不直接 import infrastructure（第3章）
- [ ] **不过度设计**：抽象层级合理，无未要求的泛化/抽象（第4章）
- [ ] **注释中文**：导出标识符有中文注释，注释说明"为什么"（第5章）
- [ ] **字段注释**：struct 每个字段有中文注释，枚举字段列取值映射（第6章）
- [ ] **业务日志**：关键节点有结构化日志，含关联字段，敏感信息脱敏（第7章）
- [ ] **类型整型**：资源/数量/状态/ID 用整型，不用 float/string 表达离散量（第8章）
- [ ] **Go规范**：gofmt/goimports 通过，错误处理规范，命名规范（第9章）
- [ ] **与spec一致**：实现不超出 spec.md 范围，不偏离 design.md 方案

---

## 11. 与现有架构的一致性声明

本全局设定与以下现有架构文档保持一致，不冲突：

| 现有文档 | 一致性说明 |
|---------|-----------|
| `docs/engine/09-服务端架构/DDD设计.md` | 第3章DDD架构规范完全遵循其四层架构与依赖方向定义 |
| `docs/engine/09-服务端架构/服务通信.md` | 第7章日志关联字段（request_id/trace_id）与其链路追踪要求一致 |
| `docs/engine/09-服务端架构/可观测性.md` | 第7章结构化日志（zap）与其可观测性要求一致 |
| `docs/engine/09-服务端架构/安全设计.md` | 第7章审计日志/脱敏与其安全设计要求一致 |
| `docs/engine/09-服务端架构/测试策略.md` | 第9章测试规范与其测试策略一致；第1章ZeroHardcodeScanner与其零硬编码扫描一致 |
| `.codeartsdoer/specs/server_biz_spec/spec.md` | 第4章不过度设计与spec 1.4职责边界一致；第8章整型与spec资源/状态语义一致 |
| `.codeartsdoer/specs/server_biz_spec/design.md` | 第3章DDD与design的聚合根/仓储接口归属一致；第1章扩展点ID常量与design 2.3.2.1节一致 |
| `.codeartsdoer/specs/server_biz_spec/tasks.md` | 本规范作为 tasks.md 任务组 1、3-8 编码的强制约束 |

---

## 12. 规范维护

- 本规范由项目架构组维护，变更需评审
- 新增规范条目追加到对应章节，更新 PR Review Checklist
- 与现有架构文档冲突时，以本规范为准并同步修订架构文档
- 规范变更需通知所有在途编码任务，已合入代码按变更项回归修复