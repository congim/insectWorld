// Package idgen ID生成能力接口，domain层声明，infrastructure层实现雪花算法适配。
//
// domain层零外部依赖（规范3），雪花算法保证全局唯一（spec 5.1.1 规则4）。
package idgen

import "context"

// IDGenerator ID生成能力接口，infrastructure层实现雪花算法适配。
//
// 接口在domain层声明（规范3 DDD），保证domain层不依赖第三方ID生成包。
// 生成的ID为int64（规范8），雪花算法保证全局唯一与时钟回拨处理。
type IDGenerator interface {
	// NextID 生成全局唯一int64 ID。
	// 时钟回拨时返回ErrIDGenClockBack（spec 4.2 可靠性）。
	NextID(ctx context.Context) (int64, error)
}
