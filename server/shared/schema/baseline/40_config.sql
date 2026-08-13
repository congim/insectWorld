-- Config上下文数据库基线

CREATE TABLE t_config_version (
    id BIGINT NOT NULL COMMENT '配置版本ID',
    version VARCHAR(32) NOT NULL COMMENT '配置语义版本',
    config_type INT NOT NULL COMMENT '配置类型',
    status INT NOT NULL COMMENT '版本状态：1=草稿 2=已发布 3=已回滚',
    operator VARCHAR(64) NOT NULL COMMENT '操作人标识',
    publish_time BIGINT NOT NULL DEFAULT 0 COMMENT '发布时间戳，未发布为0',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，Unix毫秒',
    PRIMARY KEY (id),
    UNIQUE KEY uk_config_version_version_type (version, config_type),
    KEY idx_config_version_type_time (config_type, create_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='配置版本表';

CREATE TABLE t_config_audit_log (
    id BIGINT NOT NULL COMMENT '配置审计日志ID',
    version_id BIGINT NOT NULL COMMENT '关联配置版本ID',
    operator VARCHAR(64) NOT NULL COMMENT '操作人标识',
    action INT NOT NULL COMMENT '操作类型：1=创建 2=发布 3=回滚 4=删除',
    before_value TEXT COMMENT '操作前配置摘要',
    after_value TEXT COMMENT '操作后配置摘要',
    create_time BIGINT NOT NULL COMMENT '操作时间戳，Unix毫秒',
    PRIMARY KEY (id),
    KEY idx_config_audit_log_version (version_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='配置审计日志表';
