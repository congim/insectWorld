-- Config服务DDL建表语句
-- 对应shared/schema/tables/config.go表名常量，t_前缀+蛇形+单数（规范2）

-- 配置版本表，存储配置版本历史与发布记录
CREATE TABLE IF NOT EXISTS t_config_version (
    id BIGINT NOT NULL COMMENT '版本ID',
    version VARCHAR(32) NOT NULL COMMENT '版本号',
    config_type INT NOT NULL COMMENT '配置类型',
    status INT NOT NULL COMMENT '版本状态：1=草稿 2=已发布 3=已回滚',
    operator VARCHAR(64) NOT NULL COMMENT '操作人',
    publish_time BIGINT DEFAULT 0 COMMENT '发布时间戳，毫秒级',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_config_version (version, config_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='配置版本表';

-- 配置快照表，存储配置版本快照内容
CREATE TABLE IF NOT EXISTS t_config_snapshot (
    id BIGINT NOT NULL COMMENT '快照ID',
    version_id BIGINT NOT NULL COMMENT '版本ID',
    content_json TEXT NOT NULL COMMENT '配置内容JSON',
    create_time BIGINT NOT NULL COMMENT '快照时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_config_snapshot_version (version_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='配置快照表';

-- 配置审计日志表，存储配置变更的操作审计记录
CREATE TABLE IF NOT EXISTS t_config_audit_log (
    id BIGINT NOT NULL COMMENT '审计日志ID',
    version_id BIGINT NOT NULL COMMENT '版本ID',
    operator VARCHAR(64) NOT NULL COMMENT '操作人',
    action INT NOT NULL COMMENT '操作类型：1=创建 2=发布 3=回滚 4=删除',
    before_value TEXT COMMENT '操作前值',
    after_value TEXT COMMENT '操作后值',
    create_time BIGINT NOT NULL COMMENT '操作时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_config_audit_log_version (version_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='配置审计日志表';