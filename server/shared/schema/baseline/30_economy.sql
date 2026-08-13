-- Economy上下文数据库基线
-- 资源类型由游戏包稳定字符串ID定义，通用代码不维护固定资源枚举

CREATE TABLE t_resource_account_balance (
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    resource_id VARCHAR(64) NOT NULL COMMENT '游戏包资源稳定ID',
    amount BIGINT NOT NULL DEFAULT 0 COMMENT '资源余额，必须大于等于0',
    updated_at BIGINT NOT NULL COMMENT '更新时间戳，Unix毫秒',
    PRIMARY KEY (player_id, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家资源账户余额表';

CREATE TABLE t_resource_operation (
    operation_id VARCHAR(160) NOT NULL COMMENT '资源操作全局幂等键',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    amounts_json JSON NOT NULL COMMENT '规范化资源变更量JSON',
    status INT NOT NULL COMMENT '操作状态：1=已应用 2=已撤销',
    created_at BIGINT NOT NULL COMMENT '首次操作时间戳，Unix毫秒',
    reversed_at BIGINT NOT NULL DEFAULT 0 COMMENT '撤销时间戳，未撤销为0',
    PRIMARY KEY (operation_id),
    KEY idx_resource_operation_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源变更幂等操作账本';
