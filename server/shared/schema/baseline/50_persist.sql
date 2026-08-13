-- Persist上下文数据库基线

CREATE TABLE t_schema_migration (
    id BIGINT NOT NULL COMMENT '迁移执行记录ID',
    version BIGINT NOT NULL COMMENT '递增迁移版本号',
    description VARCHAR(128) NOT NULL COMMENT '迁移脚本名称或描述',
    status INT NOT NULL COMMENT '执行状态：1=待执行 2=已执行 3=执行失败',
    execute_time BIGINT NOT NULL DEFAULT 0 COMMENT '执行时间戳，Unix毫秒',
    create_time BIGINT NOT NULL COMMENT '记录创建时间戳，Unix毫秒',
    PRIMARY KEY (id),
    UNIQUE KEY uk_schema_migration_version (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据库迁移执行记录表';

CREATE TABLE t_snapshot_task (
    id BIGINT NOT NULL COMMENT '快照任务ID',
    task_type INT NOT NULL COMMENT '快照类型：1=全量 2=增量',
    target_table VARCHAR(64) NOT NULL COMMENT '目标表名，未确定时为空',
    status INT NOT NULL COMMENT '任务状态：1=待执行 2=执行中 3=已完成 4=失败',
    start_time BIGINT NOT NULL DEFAULT 0 COMMENT '执行开始时间戳，Unix毫秒',
    end_time BIGINT NOT NULL DEFAULT 0 COMMENT '执行结束时间戳，Unix毫秒',
    create_time BIGINT NOT NULL COMMENT '任务创建时间戳，Unix毫秒',
    PRIMARY KEY (id),
    KEY idx_snapshot_task_status (status),
    KEY idx_snapshot_task_type_time (task_type, create_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据快照任务表';

CREATE TABLE t_archive_task (
    id BIGINT NOT NULL COMMENT '归档任务ID',
    source_table VARCHAR(64) NOT NULL COMMENT '源表名或归档规则ID',
    archive_condition VARCHAR(256) NOT NULL COMMENT '归档条件摘要',
    status INT NOT NULL COMMENT '任务状态：1=待执行 2=执行中 3=已完成 4=失败',
    archived_count BIGINT NOT NULL DEFAULT 0 COMMENT '已归档记录数量',
    create_time BIGINT NOT NULL COMMENT '任务创建时间戳，Unix毫秒',
    PRIMARY KEY (id),
    KEY idx_archive_task_status_time (status, create_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='冷数据归档任务表';

CREATE TABLE t_backup_task (
    id BIGINT NOT NULL COMMENT '备份任务ID',
    backup_type INT NOT NULL COMMENT '备份类型：1=全量 2=增量 3=日志',
    status INT NOT NULL COMMENT '任务状态：1=待执行 2=执行中 3=已完成 4=失败',
    backup_path VARCHAR(256) NOT NULL COMMENT '备份文件路径，未生成时为空',
    file_size BIGINT NOT NULL DEFAULT 0 COMMENT '备份文件大小，单位字节',
    create_time BIGINT NOT NULL COMMENT '任务创建时间戳，Unix毫秒',
    PRIMARY KEY (id),
    KEY idx_backup_task_status_time (status, create_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据备份任务表';
