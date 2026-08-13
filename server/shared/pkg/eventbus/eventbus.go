// Package eventbus 事件总线共享内核，提供领域事件总线契约与Outbox通用接口。
//
// 本包定义SLG服务端跨服务复用的领域事件模型与事件总线契约，包括领域事件结构、
// 事件总线接口、Outbox记录与仓储接口、事件序列化接口。各服务domain层引用本包类型
// 声明领域事件，各服务infrastructure层实现EventBus（NATS适配）与OutboxRepository
// （MySQL适配）。共享内核属domain层，不反向依赖服务（规范3）。
// 遵循规范8（优先整型）：AggregateID用int64、Version用int、时间戳用int64毫秒。
package eventbus

import "context"

// DomainEvent 领域事件，聚合根状态变更产生的不可变事件记录。
// 事件采用最终一致性模型，通过Outbox表+事件总线可靠投递，保证不丢不重。
type DomainEvent struct {
	EventID     string // 事件ID，全局唯一，用于幂等去重，通常用UUID
	EventType   string // 事件类型，如"combat.ended"/"movement.arrived"/"resource.produced"
	AggregateID int64  // 聚合根ID，事件归属的聚合根，int64（规范8）
	Version     int    // 聚合根版本号，用于乐观并发控制与事件排序
	Timestamp   int64  // 事件产生时间戳，int64毫秒级Unix时间戳（规范8）
	Payload     []byte // 事件负载，序列化后的事件具体数据，由EventSerializer编解码
}

// EventHandler 事件处理函数类型，订阅方注册的事件回调。
// 处理函数需保证幂等性，重复消费同一事件不应产生副作用。
type EventHandler func(ctx context.Context, event DomainEvent) error

// EventBus 事件总线接口，提供领域事件的发布与订阅能力。
// 接口在domain层声明（共享内核），各服务infrastructure层实现NATS/Kafka适配（规范3）。
type EventBus interface {
	// Publish 发布领域事件到事件总线。
	// 事件通过Outbox表可靠投递，保证至少一次送达。
	Publish(ctx context.Context, event DomainEvent) error

	// Subscribe 订阅指定事件类型，注册处理函数。
	// eventType: 事件类型，如"combat.ended"
	// handler: 事件处理函数，需保证幂等性
	Subscribe(ctx context.Context, eventType string, handler EventHandler) error
}

// Outbox投递状态枚举常量，表示Outbox记录的投递进度。
// 取值映射：1=待投递 2=已投递 3=失败
const (
	OutboxStatusPending   = 1 // 待投递状态，事件已写入Outbox表但尚未成功投递到事件总线
	OutboxStatusPublished = 2 // 已投递状态，事件已成功投递到事件总线
	OutboxStatusFailed    = 3 // 失败状态，事件投递失败，需重试或人工介入
)

// OutboxRecord Outbox表记录，存储待投递的领域事件。
// Outbox模式保证聚合根状态变更与事件发布的事务原子性：
// 聚合根变更与Outbox记录写入同一事务，后台任务轮询Outbox表投递事件。
type OutboxRecord struct {
	EventID     string // 事件ID，全局唯一，与DomainEvent.EventID一致，用于幂等去重
	AggregateID int64  // 聚合根ID，事件归属的聚合根，int64（规范8）
	EventType   string // 事件类型，与DomainEvent.EventType一致
	Payload     []byte // 事件负载，序列化后的事件具体数据
	Status      int    // 投递状态：1=待投递 2=已投递 3=失败
	RetryCount  int    // 重试次数，投递失败时递增，达到上限需告警
	CreateTime  int64  // 记录创建时间戳，int64毫秒级Unix时间戳（规范8）
	PublishTime int64  // 事件投递时间戳，int64毫秒级Unix时间戳，未投递时为0
}

// OutboxRepository Outbox仓储接口，提供Outbox记录的持久化与查询能力。
// 接口在domain层声明（共享内核），各服务infrastructure层实现MySQL适配（规范3）。
type OutboxRepository interface {
	// Save 保存Outbox记录，在聚合根变更事务中调用。
	Save(ctx context.Context, record OutboxRecord) error

	// MarkPublished 标记事件已投递，更新状态为已投递并记录投递时间。
	MarkPublished(ctx context.Context, eventID string, publishTime int64) error

	// GetPending 获取待投递的Outbox记录，按创建时间升序返回。
	// limit: 返回记录数上限，避免单次拉取过多
	GetPending(ctx context.Context, limit int) ([]OutboxRecord, error)
}

// EventSerializer 事件序列化接口，提供事件负载的编解码能力。
// 各服务infrastructure层实现JSON/Protobuf适配。
type EventSerializer interface {
	// Serialize 序列化事件负载为字节数组。
	Serialize(payload interface{}) ([]byte, error)

	// Deserialize 反序列化字节数组为事件负载。
	Deserialize(data []byte, target interface{}) error
}
