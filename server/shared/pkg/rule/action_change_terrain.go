// 本文件实现change_terrain规则动作：按配置参数修改指定位置的地形。
// 经ActionAccessor端口调用服务实现，地形规则合法性由宿主服务（World服务）负责。
package rule

import "fmt"

// changeTerrainHandler change_terrain动作处理器，薄处理器，副作用经ActionAccessor端口调用服务实现。
type changeTerrainHandler struct{}

// Type 返回动作类型标识"change_terrain"。
func (h *changeTerrainHandler) Type() string { return "change_terrain" }

// Execute 解析参数并调用上下文访问器修改地形，terrain_id与position必填。
func (h *changeTerrainHandler) Execute(ctx ActionContext) error {
	terrainID, _ := ctx.Params["terrain_id"].(string)
	posX, posY, ok := parsePositionParam(ctx.Params, "position")
	if terrainID == "" || !ok {
		return fmt.Errorf("动作change_terrain参数缺失或类型错误，需要terrain_id与position{x,y}")
	}
	if err := ctx.Accessor.ChangeTerrain(ctx, posX, posY, terrainID); err != nil {
		return fmt.Errorf("动作change_terrain执行失败: %w", err)
	}
	return nil
}

// init 在rule包加载时向全局注册表注册change_terrain动作。
func init() { RegisterAction(&changeTerrainHandler{}) }
