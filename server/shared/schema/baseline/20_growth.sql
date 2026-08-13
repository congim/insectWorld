-- Growth上下文数据库基线
-- 内容类型使用游戏包稳定字符串ID，运行中聚合绑定config_version

CREATE TABLE t_player_profile (
    player_id BIGINT NOT NULL COMMENT '玩家全局ID',
    faction_id VARCHAR(64) NOT NULL COMMENT '游戏包阵营稳定ID',
    nickname VARCHAR(64) NOT NULL COMMENT '玩家展示昵称',
    level INT NOT NULL DEFAULT 1 COMMENT '玩家等级，必须大于0',
    experience BIGINT NOT NULL DEFAULT 0 COMMENT '玩家经验值，必须大于等于0',
    created_at BIGINT NOT NULL COMMENT '创建时间戳，Unix毫秒',
    config_version VARCHAR(32) NOT NULL COMMENT '聚合绑定的游戏包语义版本',
    command_id VARCHAR(128) NOT NULL COMMENT '创建玩家命令幂等键',
    PRIMARY KEY (player_id),
    UNIQUE KEY uk_player_profile_command (command_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家成长档案表';

CREATE TABLE t_player_building (
    id BIGINT NOT NULL COMMENT '建筑实例ID',
    player_id BIGINT NOT NULL COMMENT '所属玩家ID',
    type_id VARCHAR(64) NOT NULL COMMENT '游戏包建筑类型稳定ID',
    status INT NOT NULL COMMENT '建筑状态：1=建造中 2=可用',
    started_at BIGINT NOT NULL COMMENT '建造开始时间戳，Unix毫秒',
    complete_at BIGINT NOT NULL COMMENT '最早完成时间戳，Unix毫秒',
    config_version VARCHAR(32) NOT NULL COMMENT '聚合绑定的游戏包语义版本',
    command_id VARCHAR(128) NOT NULL COMMENT '建造命令幂等键',
    PRIMARY KEY (id),
    UNIQUE KEY uk_player_building_command (command_id),
    KEY idx_player_building_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家建筑表';

CREATE TABLE t_training_task (
    id BIGINT NOT NULL COMMENT '训练任务ID',
    player_id BIGINT NOT NULL COMMENT '所属玩家ID',
    building_id BIGINT NOT NULL COMMENT '执行训练的建筑实例ID',
    unit_type_id VARCHAR(64) NOT NULL COMMENT '游戏包单位类型稳定ID',
    count BIGINT NOT NULL COMMENT '训练数量，必须大于0',
    status INT NOT NULL COMMENT '训练状态：1=训练中 2=已完成',
    started_at BIGINT NOT NULL COMMENT '训练开始时间戳，Unix毫秒',
    complete_at BIGINT NOT NULL COMMENT '最早完成时间戳，Unix毫秒',
    config_version VARCHAR(32) NOT NULL COMMENT '聚合绑定的游戏包语义版本',
    command_id VARCHAR(128) NOT NULL COMMENT '训练命令幂等键',
    PRIMARY KEY (id),
    UNIQUE KEY uk_training_task_command (command_id),
    KEY idx_training_task_player (player_id),
    KEY idx_training_task_building (building_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='单位训练任务表';

CREATE TABLE t_unit_roster (
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    unit_type_id VARCHAR(64) NOT NULL COMMENT '游戏包单位类型稳定ID',
    count BIGINT NOT NULL DEFAULT 0 COMMENT '已训练且可用的单位数量',
    PRIMARY KEY (player_id, unit_type_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家单位名册表';

CREATE TABLE t_unit_grant_operation (
    operation_id VARCHAR(160) NOT NULL COMMENT '单位入账幂等操作ID',
    player_id BIGINT NOT NULL COMMENT '获得单位的玩家ID',
    unit_type_id VARCHAR(64) NOT NULL COMMENT '游戏包单位类型稳定ID',
    count BIGINT NOT NULL COMMENT '本次入账数量',
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '操作创建时间',
    PRIMARY KEY (operation_id),
    KEY idx_unit_grant_operation_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='单位入账幂等操作表';
