// 本文件实现确定性随机源接口与冻结算法的PCG32风格伪随机数生成器。
// PRNG算法代码内冻结、不依赖math/rand的版本行为，保证跨Go版本、跨进程同种子同序列，
// 战斗回放一致性依赖此特性（ADR-001 3.3种子化随机方案）。
package formula

import "fmt"

// RandSource 确定性随机源接口，由调用方注入，保证同一种子产生同一序列。
// 内置实现采用固定算法（代码内冻结），不依赖math/rand的版本行为，跨进程同种子同结果。
type RandSource interface {
	// Float64 返回[0,1)区间内的伪随机浮点数，供random()函数消费。
	Float64() float64
	// Int63n 返回[0,n)区间内的伪随机整数，n必须大于0，供随机判定类函数消费。
	Int63n(n int64) int64
}

// PCG32冻结算法常量（禁止修改：任何改动都会破坏跨版本回放一致性，ADR-001 3.3）。
const (
	pcg32Multiplier uint64  = 6364136223846793005 // 线性同余递推乘数，PCG论文固定常量
	pcg32Shift      uint    = 18                  // 输出函数异或位移第一段，PCG论文固定常量
	pcg32RightShift uint    = 27                  // 输出函数异或位移第二段，PCG论文固定常量
	pcg32RotShift   uint    = 59                  // 输出函数循环右移量提取位，PCG论文固定常量
	pcg32TwoPow32   float64 = 1 << 32             // 2的32次方，将32位随机数映射到[0,1)区间
	int63Mask       int64   = (1 << 63) - 1       // 63位掩码，将64位随机数截断到int63范围
)

// Pcg32 PCG32风格的确定性伪随机数生成器，算法代码内冻结。
// 核心算法来自PCG论文的32位输出最小实现：线性同余递推状态 + 输出函数（异或位移+循环右移）；
// 同一Seed必然产生同一输出序列，与math/rand实现完全解耦。
type Pcg32 struct {
	state uint64 // 当前状态，线性同余递推的累加值
	inc   uint64 // 增量，初始化为奇数保证周期完整（PCG约束）
}

// NewPcg32 以指定种子创建PCG32随机数生成器，同种子必然产生同一序列。
// 战斗场景种子由调用方按 combatID+config_version+round 派生（ADR-001 3.3），本构造器只负责算法初始化。
func NewPcg32(seed uint64) *Pcg32 {
	p := &Pcg32{}
	p.Seed(seed)
	return p
}

// Seed 重置生成器状态为指定种子，采用PCG官方srandom初始化流程：
// 先置零状态，以种子派生增量（低位置1保证奇数），推进两次完成状态置乱。
func (p *Pcg32) Seed(seed uint64) {
	p.state = 0
	p.inc = (seed << 1) | 1
	p.next()
	p.state += seed
	p.next()
}

// next 推进生成器状态并输出一个32位伪随机数（PCG32核心算法，冻结勿改）。
func (p *Pcg32) next() uint32 {
	p.state = p.state*pcg32Multiplier + p.inc
	// 输出函数：高位异或位移取32位，再按状态最高5位循环右移，提升低位随机性
	xorshifted := uint32(((p.state >> pcg32Shift) ^ p.state) >> pcg32RightShift)
	rot := uint32(p.state >> pcg32RotShift)
	return (xorshifted >> rot) | (xorshifted << ((-rot) & 31))
}

// Float64 返回[0,1)区间内的伪随机浮点数，将32位随机数除以2的32次方映射到区间。
func (p *Pcg32) Float64() float64 {
	return float64(p.next()) / pcg32TwoPow32
}

// Int63 返回[0,2^63)区间内的伪随机整数，由两次32位输出拼接后截断到63位。
func (p *Pcg32) Int63() int64 {
	high := int64(p.next()) << 32
	low := int64(p.next())
	return (high | low) & int63Mask
}

// Int63n 返回[0,n)区间内的伪随机整数，n必须大于0，否则panic（接口契约，属于调用方编程错误）。
// 采用取模实现，模偏差在游戏配置随机场景下可忽略（AGENTS.md规范4.7避免过早优化）。
func (p *Pcg32) Int63n(n int64) int64 {
	if n <= 0 {
		panic(fmt.Sprintf("Pcg32.Int63n: n必须大于0，实际%d（接口契约：调用方编程错误）", n))
	}
	return p.Int63() % n
}
