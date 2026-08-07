-- Social服务DDL建表语句
-- 对应shared/schema/tables/social.go表名常量，t_前缀+蛇形+单数（规范2）

-- 联盟表，存储联盟基本信息与属性
CREATE TABLE IF NOT EXISTS t_alliance (
    id BIGINT NOT NULL COMMENT '联盟ID，雪花算法生成',
    name VARCHAR(64) NOT NULL COMMENT '联盟名称',
    leader_id BIGINT NOT NULL COMMENT '盟主玩家ID',
    level INT NOT NULL DEFAULT 1 COMMENT '联盟等级',
    member_count INT NOT NULL DEFAULT 1 COMMENT '当前成员数',
    max_member INT NOT NULL COMMENT '成员上限',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_alliance_name (name),
    KEY idx_alliance_leader (leader_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='联盟表';

-- 玩家表，存储玩家基本信息与状态
CREATE TABLE IF NOT EXISTS t_player (
    id BIGINT NOT NULL COMMENT '玩家ID，雪花算法生成',
    name VARCHAR(64) NOT NULL COMMENT '玩家名称',
    alliance_id BIGINT DEFAULT 0 COMMENT '所属联盟ID，0表示无联盟',
    status INT NOT NULL DEFAULT 1 COMMENT '玩家状态：1=正常 2=封禁 3=注销',
    last_login_time BIGINT DEFAULT 0 COMMENT '最后登录时间戳，毫秒级',
    create_time BIGINT NOT NULL COMMENT '创建时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_player_name (name),
    KEY idx_player_alliance (alliance_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家表';

-- 联盟成员关联表，存储玩家与联盟的成员关系
CREATE TABLE IF NOT EXISTS t_alliance_member_rel (
    id BIGINT NOT NULL COMMENT '主键ID',
    alliance_id BIGINT NOT NULL COMMENT '联盟ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    position INT NOT NULL COMMENT '职位：1=盟主 2=副盟主 3=精英 4=普通成员',
    join_time BIGINT NOT NULL COMMENT '加入时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_alliance_member_player (player_id),
    KEY idx_alliance_member_alliance (alliance_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='联盟成员关联表';

-- 联盟外交关联表，存储联盟间的外交关系
CREATE TABLE IF NOT EXISTS t_alliance_diplomacy_rel (
    id BIGINT NOT NULL COMMENT '主键ID',
    alliance_id_a BIGINT NOT NULL COMMENT '联盟A ID',
    alliance_id_b BIGINT NOT NULL COMMENT '联盟B ID',
    relation_type INT NOT NULL COMMENT '外交关系：1=同盟 2=中立 3=敌对 4=停战',
    expire_time BIGINT DEFAULT 0 COMMENT '关系到期时间戳，0表示永久',
    create_time BIGINT NOT NULL COMMENT '建立时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_alliance_diplomacy_pair (alliance_id_a, alliance_id_b)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='联盟外交关联表';

-- 福利记录表，存储联盟福利发放记录
CREATE TABLE IF NOT EXISTS t_welfare_record (
    id BIGINT NOT NULL COMMENT '福利记录ID',
    alliance_id BIGINT NOT NULL COMMENT '联盟ID',
    player_id BIGINT NOT NULL COMMENT '领取玩家ID',
    welfare_type INT NOT NULL COMMENT '福利类型',
    amount BIGINT NOT NULL COMMENT '福利数量',
    create_time BIGINT NOT NULL COMMENT '发放时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_welfare_record_alliance (alliance_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='福利记录表';