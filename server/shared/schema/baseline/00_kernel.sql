-- 通用内核数据库基线
-- 只保存当前已有生产代码读写的共享技术表

CREATE TABLE t_outbox (
    id BIGINT NOT NULL AUTO_INCREMENT COMMENT '数据库自增主键',
    event_id VARCHAR(64) NOT NULL COMMENT '稳定事件ID，用于业务幂等',
    aggregate_id BIGINT NOT NULL COMMENT '事件所属聚合根ID',
    event_type VARCHAR(64) NOT NULL COMMENT '稳定事件类型',
    event_version INT NOT NULL DEFAULT 1 COMMENT '事件契约版本号',
    payload BLOB NOT NULL COMMENT '序列化事件负载',
    status INT NOT NULL DEFAULT 1 COMMENT '投递状态：1=待投递 2=已投递 3=失败 4=投递中',
    retry_count INT NOT NULL DEFAULT 0 COMMENT '累计失败重试次数',
    create_time BIGINT NOT NULL COMMENT '事件发生时间戳，Unix毫秒',
    publish_time BIGINT NOT NULL DEFAULT 0 COMMENT '成功投递时间戳，未投递为0',
    available_time BIGINT NOT NULL DEFAULT 0 COMMENT '下次可领取时间或当前租约到期时间，Unix毫秒',
    last_error VARCHAR(512) NOT NULL DEFAULT '' COMMENT '最近一次投递失败摘要',
    PRIMARY KEY (id),
    UNIQUE KEY uk_outbox_event_id (event_id),
    KEY idx_outbox_type_status_available (event_type, status, available_time, create_time),
    KEY idx_outbox_aggregate (aggregate_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='领域事件Outbox表';
