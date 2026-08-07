-- World服务DDL建表语句
-- 对应shared/schema/tables/world.go表名常量，t_前缀+蛇形+单数（规范2）
-- 字段类型遵循整型优先原则（规范8）：ID用BIGINT、状态/类型用INT、时间戳用BIGINT毫秒、坐标用INT

-- 地图格子表，存储每个格子的地形与实体信息
CREATE TABLE IF NOT EXISTS t_map_cell (
    id BIGINT NOT NULL COMMENT '主键ID',
    map_id BIGINT NOT NULL COMMENT '地图ID',
    x INT NOT NULL COMMENT 'X轴坐标，格子坐标',
    y INT NOT NULL COMMENT 'Y轴坐标，格子坐标',
    terrain_id INT NOT NULL COMMENT '地形ID，1=平原 2=山地 3=森林 4=水域 5=沙漠',
    entity_id BIGINT DEFAULT 0 COMMENT '格子上的实体ID，0表示无实体',
    version INT NOT NULL DEFAULT 0 COMMENT '乐观并发版本号',
    update_time BIGINT NOT NULL COMMENT '更新时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_map_cell_coord (map_id, x, y),
    KEY idx_map_cell_entity (entity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='地图格子表';

-- 移动订单表，存储移动订单状态与路径
CREATE TABLE IF NOT EXISTS t_movement_order (
    id BIGINT NOT NULL COMMENT '移动订单ID，雪花算法生成',
    entity_id BIGINT NOT NULL COMMENT '移动实体ID',
    path_json TEXT COMMENT '移动路径JSON，坐标序列',
    status INT NOT NULL COMMENT '移动状态：1=待开始 2=移动中 3=已到达 4=已阻挡 5=迁移中',
    start_time BIGINT DEFAULT 0 COMMENT '移动开始时间戳，毫秒级',
    end_time BIGINT DEFAULT 0 COMMENT '移动结束时间戳，毫秒级',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_movement_order_entity (entity_id),
    KEY idx_movement_order_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='移动订单表';

-- 区域表，存储区域定义与格子范围
CREATE TABLE IF NOT EXISTS t_region (
    id BIGINT NOT NULL COMMENT '区域ID，雪花算法生成',
    center_x INT NOT NULL COMMENT '区域中心X坐标',
    center_y INT NOT NULL COMMENT '区域中心Y坐标',
    radius INT NOT NULL COMMENT '区域半径',
    cells_json TEXT COMMENT '区域包含的格子坐标列表JSON',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='区域表';

-- 传送记录表，存储传送历史与冷却记录
CREATE TABLE IF NOT EXISTS t_teleport_record (
    id BIGINT NOT NULL COMMENT '传送记录ID，雪花算法生成',
    entity_id BIGINT NOT NULL COMMENT '传送实体ID',
    teleport_type INT NOT NULL COMMENT '传送类型：1=普通 2=联盟 3=道具',
    target_x INT NOT NULL COMMENT '目标X坐标',
    target_y INT NOT NULL COMMENT '目标Y坐标',
    create_time BIGINT NOT NULL COMMENT '传送时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_teleport_record_entity (entity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='传送记录表';