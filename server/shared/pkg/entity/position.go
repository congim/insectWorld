// Package entity 实体共享内核，提供跨服务共享的实体基础类型与值对象。
package entity

// Position 坐标值对象，表示地图上的格子坐标。
// 使用整型坐标（规范8），SLG地图为格子坐标，整数即可。
type Position struct {
	X int32 // X轴坐标，格子坐标使用int32（规范8）
	Y int32 // Y轴坐标，格子坐标使用int32（规范8）
}

// Equal 判断两个坐标是否相同。
func (p Position) Equal(other Position) bool {
	return p.X == other.X && p.Y == other.Y
}

// Distance 曼哈顿距离，用于移动路径计算与视野范围判定。
// 曼哈顿距离 = |x1-x2| + |y1-y2|，适用于格子地图的路径消耗计算。
func (p Position) Distance(other Position) int32 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// Coordinate 行列坐标值对象，用于地图区域的网格化表示。
// 与Position的区别：Position用X/Y笛卡尔坐标，Coordinate用Row/Col行列坐标，
// 适用于地图分块加载、区域划分等场景。
type Coordinate struct {
	Row int32 // 行号，从0开始递增，对应地图的南北方向
	Col int32 // 列号，从0开始递增，对应地图的东西方向
}

// Equal 判断两个行列坐标是否相同。
func (c Coordinate) Equal(other Coordinate) bool {
	return c.Row == other.Row && c.Col == other.Col
}