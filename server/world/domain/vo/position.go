// Package vo World服务domain层共享值对象，提供跨聚合根复用的值对象定义。
//
// 本包通过类型别名引用shared/pkg/entity共享内核类型，消除重复定义。
// Position类型实际定义在shared/pkg/entity包中，本包保留别名以维持现有代码兼容。
package vo

import "insectworld/server/shared/pkg/entity"

// Position 坐标值对象，表示地图上的格子坐标。
// 实际类型定义在shared/pkg/entity共享内核，通过类型别名消除重复定义。
// 使用整型坐标（规范8），SLG地图为格子坐标，整数即可。
type Position = entity.Position
