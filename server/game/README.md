# Game 模块

`server/game` 是首个可玩闭环的模块化单体承载单元，不是独立微服务。当前实现 Growth 上下文的玩家档案、玩家建筑、训练任务和已训练单位名册。

## 边界

- Growth 拥有玩家档案、建筑、训练任务和单位名册的写权限；
- 游戏包只通过 `domain/catalog.Reader` 的只读契约进入领域；
- 资源余额仍由 Economy 拥有，Growth 只依赖 `domain/resource.Account` 防腐层；
- `infrastructure/memory` 只用于本地纵切与自动化测试；`infrastructure/persistence` 提供Growth自有数据的MySQL仓储；
- `cmd` 是game部署单元的装配根，允许组合Growth与Economy公开应用服务，但两个上下文仍分别拥有自己的表；
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

玩家档案、建筑、训练任务与单位名册已有MySQL表和仓储，运行中聚合持久化绑定游戏包版本。Economy提供字符串资源ID应用API和幂等操作账本，Gateway账号与注册Outbox在同一事务提交。`cmd` 已装配MySQL仓储、指定游戏包、Economy应用服务、注册事件消费者和共享Outbox发布器；发布器只领取已注册的事件类型，并暴露成功、失败、耗时和投递中数量快照。

开发环境启动示例：

```bash
cd server/game
GAME_MYSQL_DSN='user:password@tcp(127.0.0.1:3306)/insect_world?parseTime=true' \
GAME_PACK_ROOT='../../gamepacks/insect-world' \
go run ./cmd
```

`GAME_PACK_ROOT` 必须显式指定，因此同一二进制可以装配其他通过契约校验的游戏包。下一步是增加对外命令协议和跨进程消息适配；当前本地事件总线只适用于Outbox轮询器与消费者位于同一game进程的部署方式。
