-- Persist服务DDL建表语句
-- 对应shared/schema/tables/persist.go表名常量，t_前缀+蛇形+单数（规范2）
-- Persist服务负责数据治理、快照归档、迁移脚本执行

-- Schema迁移记录表，存储DDL迁移脚本的执行版本与状态
CREATE TABLE IF NOT EXISTS t_schema_migration (
    id BIGINT NOT NULL COMMENT '主键ID',
    version VARCHAR(16) NOT NULL COMMENT '迁移版本号，如V001',
    description VARCHAR(128) NOT NULL COMMENT '迁移描述',
    status INT NOT NULL COMMENT '执行状态：1=待执行 2=已执行 3=执行失败',
    execute_time BIGINT DEFAULT 0 COMMENT '执行时间戳，毫秒级',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_schema_migration_version (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Schema迁移记录表';

-- 快照任务表，存储数据快照任务的配置与执行记录
CREATE TABLE IF NOT EXISTS t_snapshot_task (
    id BIGINT NOT NULL COMMENT '任务ID',
    task_type INT NOT NULL COMMENT '快照类型：1=全量 2=增量',
    target_table VARCHAR(64) NOT NULL COMMENT '目标表名',
    status INT NOT NULL COMMENT '任务状态：1=待执行 2=执行中 3=已完成 4=失败',
    start_time BIGINT DEFAULT 0 COMMENT '执行开始时间戳，毫秒级',
    end_time BIGINT DEFAULT 0 COMMENT '执行结束时间戳，毫秒级',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_snapshot_task_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='快照任务表';

-- 归档任务表，存储冷数据归档任务的配置与执行记录
CREATE TABLE IF NOT EXISTS t_archive_task (
    id BIGINT NOT NULL COMMENT '任务ID',
    source_table VARCHAR(64) NOT NULL COMMENT '源表名',
    archive_condition VARCHAR(256) NOT NULL COMMENT '归档条件表达式',
    status INT NOT NULL COMMENT '任务状态：1=待执行 2=执行中 3=已完成 4=失败',
    archived_count BIGINT DEFAULT 0 COMMENT '已归档记录数',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_archive_task_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='归档任务表';

-- 备份任务表，存储数据备份任务的配置与执行记录
CREATE TABLE IF NOT EXISTS t_backup_task (
    id BIGINT NOT NULL COMMENT '任务ID',
    backup_type INT NOT NULL COMMENT '备份类型：1=全量 2=增量 3=日志',
    status INT NOT NULL COMMENT '任务状态：1=待执行 2=执行中 3=已完成 4=失败',
    backup_path VARCHAR(256) NOT NULL COMMENT '备份文件路径',
    file_size BIGINT DEFAULT 0 COMMENT '备份文件大小，字节',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_backup_task_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='备份任务表';