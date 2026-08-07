-- 跨服务共享DDL建表语句
-- 对应shared/schema/tables/shared.go表名常量，t_前缀+蛇形+单数（规范2）
-- 这些表被多个服务共同引用，不属于单一服务

-- Outbox表，存储待投递的领域事件
-- 各服务通过Outbox模式保证聚合根状态变更与事件发布的原子性
CREATE TABLE IF NOT EXISTS t_outbox (
    id BIGINT NOT NULL COMMENT '主键ID',
    event_id VARCHAR(64) NOT NULL COMMENT '事件ID，全局唯一，用于幂等去重',
    aggregate_id BIGINT NOT NULL COMMENT '聚合根ID',
    event_type VARCHAR(64) NOT NULL COMMENT '事件类型，如combat.ended',
    payload BLOB COMMENT '事件负载，序列化后的具体数据',
    status INT NOT NULL DEFAULT 1 COMMENT '投递状态：1=待投递 2=已投递 3=失败',
    retry_count INT NOT NULL DEFAULT 0 COMMENT '重试次数',
    create_time BIGINT NOT NULL COMMENT '记录创建时间戳，毫秒级',
    publish_time BIGINT DEFAULT 0 COMMENT '事件投递时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_outbox_event_id (event_id),
    KEY idx_outbox_status (status),
    KEY idx_outbox_aggregate (aggregate_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Outbox领域事件表';

-- 玩家归档表，存储玩家冷数据归档
-- 由Persist服务管理，从t_player归档的冷数据
CREATE TABLE IF NOT EXISTS t_player_archive (
    id BIGINT NOT NULL COMMENT '玩家ID',
    name VARCHAR(64) NOT NULL COMMENT '玩家名称',
    archive_time BIGINT NOT NULL COMMENT '归档时间戳，毫秒级',
    archive_json TEXT COMMENT '归档数据JSON',
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家归档表';