// Package vo Persist服务domain层共享值对象，提供跨聚合根复用的值对象定义。
package vo

// SnapshotRange 快照范围值对象，定义快照数据的版本范围与涉及的表。
type SnapshotRange struct {
	StartVersion int64    // 起始版本号，增量快照的起始数据版本
	EndVersion   int64    // 结束版本号，增量快照的结束数据版本
	TableList    []string // 涉及的表名列表，指定快照范围的表
}