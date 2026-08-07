-- Economy服务DDL建表语句
-- 对应shared/schema/tables/economy.go表名常量，t_前缀+蛇形+单数（规范2）

-- 资源余额表，存储玩家各资源类型的余额
CREATE TABLE IF NOT EXISTS t_resource_balance (
    id BIGINT NOT NULL COMMENT '主键ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    resource_type INT NOT NULL COMMENT '资源类型：1=粮食 2=木材 3=石料 4=铁矿 5=金币',
    amount BIGINT NOT NULL DEFAULT 0 COMMENT '资源余额，int64（规范8）',
    update_time BIGINT NOT NULL COMMENT '更新时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_resource_balance_player_type (player_id, resource_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源余额表';

-- 生产线表，存储资源产出线的配置与状态
CREATE TABLE IF NOT EXISTS t_production_line (
    id BIGINT NOT NULL COMMENT '生产线ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    resource_type INT NOT NULL COMMENT '产出资源类型',
    rate BIGINT NOT NULL COMMENT '产出速率，每小时产出量',
    status INT NOT NULL COMMENT '生产线状态：1=运行中 2=已暂停 3=已停止',
    last_collect_time BIGINT NOT NULL COMMENT '上次采集时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_production_line_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='生产线表';

-- 交易订单表，存储玩家间交易订单
CREATE TABLE IF NOT EXISTS t_trade_order (
    id BIGINT NOT NULL COMMENT '交易订单ID',
    seller_id BIGINT NOT NULL COMMENT '卖方玩家ID',
    buyer_id BIGINT DEFAULT 0 COMMENT '买方玩家ID，0表示未成交',
    resource_type INT NOT NULL COMMENT '交易资源类型',
    amount BIGINT NOT NULL COMMENT '交易数量',
    price BIGINT NOT NULL COMMENT '交易单价，int64分（规范8）',
    status INT NOT NULL COMMENT '订单状态：1=挂单中 2=已成交 3=已取消',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_trade_order_seller (seller_id),
    KEY idx_trade_order_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='交易订单表';

-- 转换订单表，存储资源转换订单
CREATE TABLE IF NOT EXISTS t_conversion_order (
    id BIGINT NOT NULL COMMENT '转换订单ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    from_type INT NOT NULL COMMENT '源资源类型',
    to_type INT NOT NULL COMMENT '目标资源类型',
    from_amount BIGINT NOT NULL COMMENT '源资源数量',
    to_amount BIGINT NOT NULL COMMENT '目标资源数量',
    status INT NOT NULL COMMENT '订单状态：1=进行中 2=已完成 3=已失败',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_conversion_order_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='转换订单表';