# ADR-005 Framework骨架与骨头下沉策略：server/framework 独立module + 三批下沉

> **状态**：已接受（Accepted）
> **日期**：2026-08-11
> **决策者**：程基岩（engineering-lead）
> **关联**：《通用SLG服务端开发计划》P2-E5；《通用SLG框架共性抽离方案_终版》第四章（分层模型）/第五章（工程落地）；`AGENTS.md`；`server/shared/go.mod`（L0 module）
> **依赖**：E1 公式引擎（`shared/pkg/formula`）、E2 行为注册表（`shared/pkg/rule`）已落地
> **被依赖**：P3-E8~E13 六根新骨头（第二批下沉）、P4-E14 存量12根老骨头下沉（第三批）、E17 CI门禁

---

## 一、背景与问题

### 1.1 现状

| 事实 | 位置 | 说明 |
|------|------|------|
| 分层模型已冻结 | 《通用SLG框架共性抽离方案_终版》第四章 | L0=shared基础设施 / L1=framework共性机制（21根骨头，零题材字符串）/ L2=18配置 / L3=extensions题材；依赖单向 L0←L1←L2←L3 |
| 行为注册表+公式引擎已落地 | `server/shared/pkg/rule/`、`server/shared/pkg/formula/` | 已是L1级质量：零题材字符串、有单测、机制稳定（ADR-001/002已接受） |
| **framework module 不存在** | `server/` | 21根骨头无代码归属地，L1依赖链未建立 |
| 存量12根骨头散落8服务 | `server/{world,combat,economy,social,operation,...}/domain/` | 换皮时骨头动不了，需逐个下沉（P4） |
| 现有8服务+shared构建基线 | — | E5开工前全部 `go build ./...` 通过 |

### 1.2 问题

1. **L1 无落地载体**：终版方案定义了 `server/framework/`，但未落地——需要创建独立 Go module、规划21根骨头如何组织、明确与 shared 的边界。
2. **rule/formula 归属待裁决**：两者已实现于 shared（L0 module），但终版方案字面将 action_registry 列于 L1。直接物理迁入 framework 会破坏现有消费方 import（formula 被 combat 的 `execute_round.go`/`replay.go` 引用），复制代码则产生双实现。需给出裁决。
3. **下沉无节奏**：8人日 Epic 无法一次迁移21根骨头；需分批策略，避免与 P3 六骨头并行时互相踩踏。

---

## 二、决策（摘要）

1. **创建 `server/framework/` 独立 Go module**（`module insectworld/server/framework`，Go 1.26），通过 `require insectworld/server/shared v0.0.0` + `replace => ../shared` 依赖 L0，依赖方向 L1→L0，反向禁止。
2. **首批内容不复制代码、不物理迁移**：`shared/pkg/rule`、`shared/pkg/formula` 物理留在 shared（L0 内核原语），framework 经 require 引用；`framework/mechanism/ruleaction`、`framework/mechanism/formula` 以**类型别名+委托函数**聚合导出，建立 L1 稳定入口并验证依赖链。
3. **21根骨头组织方式**：`mechanism/` 统一伞目录 + 一根骨头一个包子目录（扁平，不按领域二次分组）。
4. **三批下沉**：
   - 第一批（本期 E5）：ruleaction/formula 聚合导出 + module + 依赖链验证；
   - 第二批（P3 E8-E13）：六根新骨头在 framework/mechanism/ 下**直接新建**；
   - 第三批（P4 E14）：存量12根老骨头**逐骨头原子下沉**（同 PR 移动+改 import+回归）。
5. **依赖规则**：L1 只依赖 L0；禁止 import 服务/题材代码；零题材字符串 CI 扫描（本期出脚本原型，E17 接 CI 门禁）。

---

## 三、设计要点

### 3.1 framework/ 目录规划（21根骨头组织方式）

```
server/framework/                  # module insectworld/server/framework
  go.mod                           # require shared + replace，Go 1.26
  doc.go                           # module入口：分层模型+依赖规则说明
  mechanism/                       # L1机制骨头统一伞目录
    ruleaction/                    # 规则系统骨头（存量#7）：行为注册表+规则执行编排
    formula/                       # 公式机制骨头：战斗/经济公式求值挂载
    playerprofile/                 # 新#13 玩家档案（P3第二批新建）
    building/                      # 新#14 建筑升级（P3第二批新建）
    gathering/                     # 新#15 采集调度（P3第二批新建）
    inventory/                     # 新#16 背包/道具（P3第二批新建）
    troop_training/                # 新#17 兵种训练（P3第二批新建）
    technology/                    # 新#18 个人科技（P3第二批新建）
    shop/                          # 新#19 商店骨架（P4）
    ranking/                       # 新#20 排行榜骨架（P4）
    tutorial/                      # 新#21 新手引导（P4）
    entity/ map/ movement/ combat/ economy/ buff/ alliance/ season/
    event/ network/ persist/       # 存量12根老骨头（P4第三批逐根下沉）
```

**组织方式决策：mechanism/ 伞目录 + 扁平骨头子包。**

备选与取舍：

| 方案 | 说明 | 评估 | 结论 |
|------|------|------|------|
| A. 直接子包 | `framework/building/`、`framework/combat/`... 21包直落 module 根 | module 根被21个业务包占满，无留给 doc/聚合入口/工具；语义上"机制"是它们的共同身份 | 否决 |
| B. 按领域分组 | `framework/combat/`（战斗+移动+Buff+网络同步）、`framework/economy/`（经济+背包+建筑+科技）... | 存量骨头**跨旧服务边界**（combat 域含战斗/移动/Buff/网络同步多根，economy 域含经济/背包/建筑/科技），按领域分组会固化即将拆解的旧服务边界，违背下沉目标 | 否决 |
| C. mechanism/ 伞目录 + 扁平骨头子包（选定） | 21根骨头 peer 机制，语义统一为"机制"；伞目录提供可发现性；module 根保持干净 | 与任务建议的 `mechanism/` 骨架一致；便于按骨头独立演进/测试/下沉 | ✅ 选定 |

**包名规范**：骨头包名用单个小写单词（AGENTS.md 9.1.2），**不用下划线**。任务字面 `rule_action/` 落地为 `ruleaction/`（本 ADR 记录此差异；同理 `troop_training` 类多词骨头用驼峰拼接如 `trooptraining`，或取单词语义如 `training`，P3 立项时定）。

### 3.2 首批内容决策：rule/formula 归属裁决（E5-S1 关键决策）

**裁决：不复制代码、不物理迁移；framework 通过 require 引用 shared；聚合导出。**

`shared/pkg/rule` + `shared/pkg/formula` **已经是 L1 级质量**（零题材字符串、单测齐、机制稳定、ADR-001/002 已接受），但物理位置在 shared module。决策理由：

1. **零风险**：formula 有活消费方（combat 的 `execute_round.go`/`replay.go`），物理迁移必改其 import 并让 combat 依赖 framework（涉及 module 图变更）；rule 虽暂无外部消费方，但迁移仍产生 churn。骨架期无收益、纯风险。
2. **依赖方向天然正确**：`framework → shared` 正是 L1→L0 合法方向，且这正是 E5 要验证的依赖链。物理迁移反而让 combat（服务，L2侧）依赖 framework，跨层路径变长。
3. **避免双实现**：复制代码会产生两套规则/公式实现，后续维护灾难（违反 AGENTS.md 4 不过度设计/不重复实现）。
4. **聚合导出提供 L1 稳定契约出口**：服务侧将来统一经 `framework/mechanism/*` 引用规则/公式机制，L0 细节被封装；`ruleaction` 包的编排层（启动校验接入、RuleExecutor 组合、规则触发链路）在 P3/P4 生长，位置已预留。
5. **引擎原语 vs 骨头编排分离**：formula 是通用表达式求值器（L0 基础设施属性）；rule 的注册表是机制原语（L0 内核）。真正属于 L1 的"规则系统骨头"是**编排层**，不是原语本身。

**与终版方案字面差异的裁决**：终版方案第五章将 action_registry 列于 L1 代码归属（`server/framework/`），本文裁决：注册表原语代码实体留在 `shared/pkg/rule`（L0），L1 的 `ruleaction` 骨头包负责编排与对外契约。理由：①避免复制/迁移冲击现有消费方；②registry 原语零题材字符串、机制稳定，作为 L0 共享内核更符合 AGENTS.md 0.1（rule 被列为共享内核模块）与分层纯净性；③终版方案 5.3"server/ 目录调整建议"本身也要求 shared 不变。**本裁决以 ADR-005 为权威**，后续文档口径以此为准。

### 3.3 三批下沉策略

| 批次 | 内容 | 时机 | 产出/门禁 |
|------|------|------|-----------|
| **第一批：骨架落地** | framework module + `mechanism/ruleaction`、`mechanism/formula` 聚合导出（类型别名+委托函数） | 本期 E5 | framework 独立编译 + 单测绿；shared/8服务零改动回归绿；ADR-005 |
| **第二批：六根新骨头** | 玩家档案/建筑升级/采集调度/背包道具/兵种训练/个人科技 在 framework/mechanism/ 下**直接新建** | P3 E8-E13 | 新骨头天然落 L1；ruleaction 编排层同步生长（启动校验接入）；换皮校验零题材串（M3 验收） |
| **第三批：存量12根老骨头下沉** | 实体/地图/移动/战斗/经济/Buff/规则/联盟/赛季/事件/网络同步/持久化 从 8 服务 domain 逐根下沉 | P4 E14 | 每根骨头一个 PR：抽机制代码→framework 新包子目录→服务侧改引用→迁移单测→全量回归；服务瘦身为题材适配+编排层 |

**下沉节奏约束**：
- 每批之间以"framework 可编译 + 全部服务回归绿"为门禁，不做大爆炸式迁移；
- 第三批每根骨头下沉**优先于**该骨头的存量功能补齐（开发计划第四章重排原则：先 P3 骨头再补存量组件），避免重复实现；
- 第二、三批可并行，但**同一骨头不得同时被两边改动**（见 3.6 风险4）。

### 3.4 依赖规则与零题材字符串校验

**依赖规则（硬约束）**：
1. L1（framework）只依赖 L0（shared）与标准库；禁止 import 任何服务代码（`insectworld/server/{world,combat,economy,social,operation,gateway,config,persist,...}`）与题材代码（`extensions/`）。
2. 服务（L2侧）可以依赖 L1 与 L0；L0 禁止依赖 L1（shared 不 import framework）。
3. 新增 framework 骨头包只允许引用：标准库、`insectworld/server/shared/...`、`insectworld/server/framework/...`（本 module 内）。

**校验方式（E17 接 CI，本期落脚本原型 `server/scripts/check_framework_purity.sh`）**：
1. **import 白名单扫描**：遍历 framework/ 下所有 .go 的 import，非白名单（标准库 + shared + framework 内部）即失败；
2. **零题材字符串扫描**：扫描 framework/ 下 .go 源码的字符串字面量，命中题材黑名单词（昆虫/信息素/进化树/蛰伏/步兵/弓箭手/星际/虫洞/三国/计策/附庸/贡品/蚂蚁 等）即失败；白名单通用机制术语（buff/resource/entity/terrain/season/...）不告警；
3. 与既有 `ZeroHardcodeScanner`（数值零硬编码）互补，题材字符串扫描是其"零题材"维度的补充。

### 3.5 与 shared/pkg 的边界

| 层 | module | 内容 | 判断标准 |
|----|--------|------|---------|
| **L0 基础设施+共享契约** | `insectworld/server/shared` | `pkg/eventbus`（事件总线）、`pkg/config`（配置加载/热更/版本化/回滚）、`pkg/entity`（ECS实体）、`pkg/formula`（公式引擎原语）、`pkg/rule`（规则引擎+行为注册表原语）、`pkg/configdeps`、`schema/tables`（统一表定义）、`proto/`（服务间契约） | 纯基础设施/共享契约/机制原语；零玩法；稳定 |
| **L1 共性机制** | `insectworld/server/framework` | mechanism/ 下21根骨头（编排层+状态机+契约） | SLG共性机制；零题材字符串；依赖只向下（L0） |
| **L3 题材扩展** | `insectworld/server/extensions/<题材>`（未来） | 题材专属玩法 | 题材特有；依赖 L0+L1 |

**包归属决策树**（新代码落到哪层）：
```
机制是否纯基础设施/共享契约？        → L0 shared
机制是否SLG共性且零题材字符串？      → L1 framework/mechanism/<骨头>
是否题材专属玩法？                  → L3 extensions/<题材>
其余（配置内容）                   → L2 配置文件
```

**不迁移裁决重申**：已成熟的 rule/formula 原语**不再物理迁入** framework（3.2），framework 骨头包是其 L1 契约出口；若未来 shared 过度膨胀确需物理迁移，走 3.6 风险1 的原子迁移流程。

### 3.6 迁移风险与回滚方案

| # | 风险 | 缓解 |
|---|------|------|
| 1 | **import 破坏**：物理迁移必改消费方 import | E5 不迁移（引用而非复制）；第三批逐骨头原子迁移（同 PR 内移动+改 import+全量回归），任一服务回归失败即回滚该 PR |
| 2 | **重复实现**：复制代码 → 两套实现漂移 | 禁止复制（本 ADR 硬约束）；聚合导出用类型别名（零复制，别名=同一类型无漂移） |
| 3 | **依赖方向污染**：framework 反向 import 服务/题材 | CI import 白名单扫描（3.4），违规即阻断合入 |
| 4 | **下沉时序冲突**：第三批与 P4 存量改造并行时同一骨头两边改动 | 每根骨头下沉前冻结其服务侧 domain 改动窗口；下沉优先于该骨头存量功能补齐；同骨头不同时两边改 |
| 5 | **module 图膨胀**：framework 引入 shared 全量传递依赖 | framework 只 import rule/formula 包时 go.mod 仅含 testify+shared+间接 zap 等，`go mod tidy` 自动最小化；不 import proto/schema 即不引入 grpc/protobuf（已实测验证） |

**回滚方案**：framework 是**新增 module**，与既有服务完全隔离；任何批次回滚 = 删除该批次新增的 framework 骨头包 + 恢复服务侧 import（git revert），**不触碰 shared 与其他服务**；本期 shared 与 8 服务零改动，天然可回滚。

---

## 四、备选方案对比

| 方案 | 说明 | 评估 | 结论 |
|------|------|------|------|
| A. 物理迁移 rule/formula 到 framework | 代码实体搬入 L1 | formula 破坏 combat import、module 图变更；骨架期无收益；与 shared 内核定位冲突（AGENTS.md 0.1） | 否决（推迟：仅在第三批确有需要时按原子迁移流程执行） |
| B. 复制代码到 framework | framework 内复制 rule/formula | 两套实现漂移，维护灾难；违反 AGENTS.md 4 | 否决（硬约束） |
| C. **framework 引用 shared + 聚合导出（选定）** | require + replace；类型别名+委托函数聚合 | 零风险、依赖链正确、L1 稳定契约出口、为 P3 编排层预留位置；成本极低 | ✅ 选定 |
| D. 仅 require 引用、不建聚合包 | 依赖链验证最简 | L1 无统一入口，服务仍直接散落依赖 L0 细节；编排层无落点 | 否决（聚合包成本极低，收益明确） |

---

## 五、影响范围

| 范围 | 影响 |
|------|------|
| `server/framework/`（新增） | go.mod + doc.go + mechanism/mechanism.go + mechanism/ruleaction/ + mechanism/formula/（各含测试） |
| `server/shared/`、8服务 | **本期零改动**（回归验证通过） |
| 文档 | ADR-005 新增；`docs/engine/10-ADR/README.md` 索引更新 |
| 脚本 | `server/scripts/check_framework_purity.sh` 新增（3.4 校验原型，E17 接 CI） |

---

## 六、验收标准（对齐开发计划 E5）

- [ ] framework module 创建完成，`go build ./...` 通过（独立 module 可编译）；
- [ ] 依赖链验证：framework go.mod `require insectworld/server/shared v0.0.0` + `replace => ../shared`；
- [ ] 首批代码单测绿：`mechanism/ruleaction`、`mechanism/formula` 测试通过（聚合导出功能委托正常）；
- [ ] 零题材字符串：framework 源码无题材词（purity 脚本原型通过）；
- [ ] 现有 8 服务 + shared 构建不受影响（回归验证通过）；
- [ ] 遵循 AGENTS.md：包名单词无下划线、中文注释、gofmt/vet 通过；
- [ ] ADR-005 落盘（本文件）+ ADR README 索引更新。

---

## 七、后续动作

1. **E6** 存量基础补齐与 E5 并行（不阻塞）；
2. **P3 E8-E13** 六根新骨头在 `framework/mechanism/` 下直接新建（第二批），立项时确定各骨头包名（驼峰拼接单词语义）；
3. **E17** 将 `check_framework_purity.sh` 接入 CI 门禁（import 白名单 + 零题材字符串扫描自动化）；
4. **P4 E14** 第三批逐骨头下沉：每根骨头一个 PR（抽机制→framework→服务改引用→迁移单测→全量回归）；
5. **E7** proto 契约与 framework 骨头契约对齐时，复核 ruleaction/formula 聚合导出的导出面是否需扩展。
