// Package identity 定义成长上下文生成聚合实例ID的技术端口。
package identity

// Generator 生成进程内单调递增且大于0的聚合实例ID。
// 生产实现应替换为全局唯一ID生成器，调用方不依赖具体算法。
type Generator interface {
	Next() int64
}
