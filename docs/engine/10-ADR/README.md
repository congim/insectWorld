# ADR 索引（Architecture Decision Records）

> 本目录记录通用SLG服务端的重大架构决策。每个ADR回答一个"为什么这么做"的问题，冻结后作为实现依据。
> 约定：ADR只追加不删除；被替代的ADR标注"被ADR-XXX替代"；变更需新增ADR而非修改旧ADR。

| 编号 | 标题 | 状态 | 日期 | 关联Epic |
|------|------|------|------|---------|
| [ADR-001](ADR-001-公式引擎.md) | 公式引擎升级：AST表达式引擎+内置函数库+种子化确定性随机 | 已接受 | 2026-08-11 | P1-E1 |
| [ADR-002](ADR-002-规则动作注册表.md) | 规则动作注册表：RuleActionHandler行为注册+启动全量校验 | 已接受 | 2026-08-11 | P1-E2 |
| [ADR-003](ADR-003-规则动作枚举口径统一.md) | 规则动作枚举口径统一：以E2代码为权威，内置7个+注册表可扩展 | 已接受 | 2026-08-11 | P1-E7 |
| [ADR-004](ADR-004-战斗快照配置版本绑定.md) | 战斗快照配置版本绑定：CombatSnapshot.ConfigVersion+结算校验熔断+热更删除实例扫描 | 已接受 | 2026-08-11 | P1-E3 |
| [ADR-005](ADR-005-Framework骨架与骨头下沉策略.md) | Framework骨架与骨头下沉策略：server/framework独立module+三批下沉+rule/formula不迁移引用 | 已接受 | 2026-08-11 | P2-E5 |

## 决策记录要点

- **ADR-001（E1 公式引擎）**：四则运算升级为 AST+函数库+种子随机。同种子同结果，战斗回放一致。
- **ADR-002（E2 行为注册表）**：动作类型从写死枚举改为注册表驱动，新动作=题材层 init() 注册，框架零改动。
- **ADR-003（E7 动作枚举口径统一）**：以E2已实现代码为权威事实，统一动作枚举=内置7个（apply_buff/remove_buff/modify_resource/spawn_entity/trigger_combat/send_notify/change_terrain）+注册表可扩展；modify_attribute归入buff体系、grant_resource/send_event分别由modify_resource/send_notify覆盖、custom为题材层注册动作类型标识、execute_formula列为内置动作候选（依赖E1公式引擎，本期未内置，YAGNI）。
- **ADR-004（E3 热更×快照×权威方）**：战斗快照绑定 config_version + 版本化配置查询；结算前校验缺失走熔断（默认强制平局）；热更删除配置项发布前扫描进行中实例（阻塞/强制）；双向回滚语义（快照类保持开始版本、即时类跟随当前版本）；种子取快照冻结值保证重放一致。
- **ADR-005（E5 Framework骨架）**：创建 server/framework 独立module（L1，依赖L0 shared）；21根骨头按 mechanism/ 伞目录+扁平骨头子包组织；rule/formula 不复制不迁移，framework 经 require 引用+类型别名聚合导出；三批下沉（骨架→P3六新骨头→P4存量12骨头逐根原子迁移）；L1只依赖L0，零题材字符串CI扫描（E17接入）。

