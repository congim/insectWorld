# Game 模块

`server/game` 是首个可玩闭环的模块化单体承载单元，不是独立微服务。当前实现 Growth 上下文的玩家档案、玩家建筑、训练任务和已训练单位名册。

## 边界

- Growth 拥有玩家档案、建筑、训练任务和单位名册的写权限；
- 游戏包只通过 `domain/catalog.Reader` 的只读契约进入领域；
- 资源余额仍由 Economy 拥有，Growth 只依赖 `domain/resource.Account` 防腐层；
- `infrastructure/memory` 只用于本地纵切与自动化测试，不代表生产持久化方案；
- Combat 后续消费单位名册投影，不直接修改训练聚合。

## 已实现链路

```text
编译游戏包 → 创建玩家/初始资源 → 建造建筑 → 完成建筑
          → 开始训练/扣除资源 → 完成训练/单位入账
```

创建玩家、建造和开始训练使用外部幂等键；资源变更和单位入账使用稳定操作ID二次防重。相同幂等键携带不同载荷会返回状态冲突。

## 验证

```bash
cd server/game
go test -race -cover ./...
```

下一步应实现 Economy 的字符串资源ID正式适配、MySQL仓储与Outbox，然后接入Gateway注册事件；不要把内存适配器用于生产启动入口。
