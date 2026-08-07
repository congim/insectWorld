-- Operation服务DDL建表语句
-- 对应shared/schema/tables/operation.go表名常量，t_前缀+蛇形+单数（规范2）

-- 赛季表，存储赛季基本信息与时间范围
CREATE TABLE IF NOT EXISTS t_season (
    id BIGINT NOT NULL COMMENT '赛季ID',
    season_num INT NOT NULL COMMENT '赛季序号',
    status INT NOT NULL COMMENT '赛季状态：1=准备中 2=进行中 3=已结束',
    start_time BIGINT NOT NULL COMMENT '赛季开始时间戳，毫秒级',
    end_time BIGINT DEFAULT 0 COMMENT '赛季结束时间戳，毫秒级',
    PRIMARY KEY (id),
    UNIQUE KEY uk_season_num (season_num)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='赛季表';

-- 赛季阶段表，存储赛季各阶段的配置与状态
CREATE TABLE IF NOT EXISTS t_season_phase (
    id BIGINT NOT NULL COMMENT '阶段ID',
    season_id BIGINT NOT NULL COMMENT '赛季ID',
    phase_type INT NOT NULL COMMENT '阶段类型：1=准备 2=战斗 3=结算 4=休赛',
    status INT NOT NULL COMMENT '阶段状态：1=未开始 2=进行中 3=已结束',
    start_time BIGINT NOT NULL COMMENT '阶段开始时间戳，毫秒级',
    end_time BIGINT DEFAULT 0 COMMENT '阶段结束时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_season_phase_season (season_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='赛季阶段表';

-- 排行榜表，存储赛季排行榜数据
CREATE TABLE IF NOT EXISTS t_score_board (
    id BIGINT NOT NULL COMMENT '排行榜记录ID',
    season_id BIGINT NOT NULL COMMENT '赛季ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    score BIGINT NOT NULL COMMENT '积分',
    rank INT NOT NULL COMMENT '排名',
    update_time BIGINT NOT NULL COMMENT '更新时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_score_board_season (season_id),
    KEY idx_score_board_rank (season_id, rank)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='排行榜表';

-- 游戏事件表，存储赛季中的游戏事件记录
CREATE TABLE IF NOT EXISTS t_game_event (
    id BIGINT NOT NULL COMMENT '事件ID',
    season_id BIGINT NOT NULL COMMENT '赛季ID',
    event_type INT NOT NULL COMMENT '事件类型',
    player_id BIGINT DEFAULT 0 COMMENT '关联玩家ID',
    detail_json TEXT COMMENT '事件详情JSON',
    create_time BIGINT NOT NULL COMMENT '事件时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_game_event_season (season_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='游戏事件表';

-- 赛季快照表，存储赛季结束时的数据快照
CREATE TABLE IF NOT EXISTS t_season_snapshot (
    id BIGINT NOT NULL COMMENT '快照ID',
    season_id BIGINT NOT NULL COMMENT '赛季ID',
    snapshot_json TEXT COMMENT '快照数据JSON',
    create_time BIGINT NOT NULL COMMENT '快照时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_season_snapshot_season (season_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='赛季快照表';