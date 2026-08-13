# 游戏包目录

每个子目录是一套可独立校验和编译的题材内容。通用内核只认识 manifest 与强类型 schema，不包含这里的名称和数值。

```text
<pack-id>/
├── manifest.yaml
└── configs/
    ├── game.json
    ├── factions.json
    ├── resources.json
    ├── units.json
    ├── buildings.json
    ├── terrains.json
    └── maps.json
```

首版 schema 由 `server/shared/pkg/gamepack` 中的 Go 类型和严格解码器执行：未知字段、重复 ID、断裂引用、非法数值、路径越界或引擎版本不兼容都会拒绝编译。

从仓库根目录验证全部游戏包：

```bash
cd tools
go run ./reskin_validator -root ../gamepacks -engine-version 0.1.0
```

当前 `frontier-demo` 只作为第二契约用例，不是并行开发的游戏产品。
