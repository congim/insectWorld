// Package vo Inventory服务值对象，定义道具与背包相关的不可变值对象。
// 值对象无独立生命周期，通过组合关系归属于聚合根。
package vo

// ItemID 道具实例ID，全局唯一，由雪花算法生成（规范8用int64）。
type ItemID int64

// ItemDefID 道具定义ID，对应items.json的道具定义，配置驱动（规范8用int64）。
type ItemDefID int64

// 道具来源类型常量（规范1），标识道具的获取渠道。
const (
	ItemSourceDrop     = 1 // 战斗掉落
	ItemSourceReward   = 2 // 任务/活动奖励
	ItemSourcePurchase = 3 // 商城购买
	ItemSourceInitial  = 4 // 初始赠送
)

// Item 道具值对象，描述一个道具实例的完整属性，不可变。
type Item struct {
	ItemID     ItemID    // 道具实例ID，全局唯一
	DefID      ItemDefID // 道具定义ID，对应items.json配置
	Count      int64     // 堆叠数量，可堆叠道具的当前堆叠数（规范8用int64）
	ExpireTime int64     // 过期时间戳（毫秒），0表示永不过期（规范8用int64）
	Source     int       // 获取来源：1=战斗掉落 2=任务奖励 3=商城购买 4=初始赠送
	ObtainTime int64     // 获取时间戳（毫秒）
}

// IsExpired 判断道具是否已过期，expireTime为0表示永不过期。
func (i Item) IsExpired(now int64) bool {
	if i.ExpireTime == 0 {
		return false
	}
	return now >= i.ExpireTime
}

// StackPolicy 堆叠策略值对象，从items.json配置驱动。
type StackPolicy struct {
	Stackable bool  // 是否可堆叠
	MaxStack  int64 // 最大堆叠数，Stackable为true时有效
}
