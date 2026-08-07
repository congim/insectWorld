-- Combat服务DDL建表语句
-- 对应shared/schema/tables/combat.go表名常量，t_前缀+蛇形+单数（规范2）
-- 字段类型遵循整型优先原则（规范8）

-- 战斗表，存储战斗状态与参战方信息
CREATE TABLE IF NOT EXISTS t_combat (
    id BIGINT NOT NULL COMMENT '战斗ID，雪花算法生成',
    attacker_id BIGINT NOT NULL COMMENT '攻击方实体ID',
    defender_id BIGINT NOT NULL COMMENT '防御方实体ID',
    status INT NOT NULL COMMENT '战斗状态：1=进行中 2=已结束 3=已取消',
    round INT NOT NULL DEFAULT 0 COMMENT '当前轮次',
    max_round INT NOT NULL COMMENT '最大轮次',
    winner INT DEFAULT 0 COMMENT '胜方：0=未决 1=攻击方 2=防御方 3=平局',
    start_time BIGINT NOT NULL COMMENT '战斗开始时间戳，毫秒级',
    end_time BIGINT DEFAULT 0 COMMENT '战斗结束时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_combat_attacker (attacker_id),
    KEY idx_combat_defender (defender_id),
    KEY idx_combat_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='战斗表';

-- 战斗轮次表，存储每轮战斗结果
CREATE TABLE IF NOT EXISTS t_combat_round (
    id BIGINT NOT NULL COMMENT '轮次记录ID',
    combat_id BIGINT NOT NULL COMMENT '战斗ID',
    round_num INT NOT NULL COMMENT '轮次序号',
    attacker_damage INT NOT NULL COMMENT '攻击方伤害值',
    defender_damage INT NOT NULL COMMENT '防御方伤害值',
    detail_json TEXT COMMENT '轮次详情JSON',
    create_time BIGINT NOT NULL COMMENT '轮次时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_combat_round_combat (combat_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='战斗轮次表';

-- 战报表，存储战报详情
CREATE TABLE IF NOT EXISTS t_combat_report (
    id BIGINT NOT NULL COMMENT '战报ID',
    combat_id BIGINT NOT NULL COMMENT '战斗ID',
    report_json TEXT COMMENT '战报详情JSON',
    create_time BIGINT NOT NULL COMMENT '战报生成时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_combat_report_combat (combat_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='战报表';

-- 技能冷却表，存储技能冷却记录
CREATE TABLE IF NOT EXISTS t_skill_cooldown (
    id BIGINT NOT NULL COMMENT '冷却记录ID',
    entity_id BIGINT NOT NULL COMMENT '实体ID',
    skill_id BIGINT NOT NULL COMMENT '技能ID',
    expire_time BIGINT NOT NULL COMMENT '冷却到期时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_skill_cooldown_entity_skill (entity_id, skill_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='技能冷却表';