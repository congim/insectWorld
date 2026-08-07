-- Gateway服务DDL建表语句
-- 对应shared/schema/tables/gateway.go表名常量，t_前缀+蛇形+单数（规范2）

-- 会话表，存储玩家在线会话信息
CREATE TABLE IF NOT EXISTS t_session (
    id BIGINT NOT NULL COMMENT '会话ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    conn_id VARCHAR(64) NOT NULL COMMENT '连接ID',
    status INT NOT NULL COMMENT '会话状态：1=在线 2=离线',
    login_time BIGINT NOT NULL COMMENT '登录时间戳，毫秒级',
    heartbeat_time BIGINT NOT NULL COMMENT '最后心跳时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_session_player (player_id),
    KEY idx_session_conn (conn_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='会话表';

-- 连接记录表，存储玩家连接历史
CREATE TABLE IF NOT EXISTS t_connection_record (
    id BIGINT NOT NULL COMMENT '记录ID',
    player_id BIGINT NOT NULL COMMENT '玩家ID',
    ip VARCHAR(45) NOT NULL COMMENT '连接IP',
    connect_time BIGINT NOT NULL COMMENT '连接时间戳，毫秒级',
    disconnect_time BIGINT DEFAULT 0 COMMENT '断开时间戳，毫秒级',
    PRIMARY KEY (id),
    KEY idx_connection_record_player (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='连接记录表';

-- 路由表，存储请求路由配置
CREATE TABLE IF NOT EXISTS t_route_table (
    id BIGINT NOT NULL COMMENT '路由ID',
    route_path VARCHAR(128) NOT NULL COMMENT '路由路径',
    target_service VARCHAR(32) NOT NULL COMMENT '目标服务',
    method VARCHAR(16) NOT NULL COMMENT '请求方法',
    rate_limit INT DEFAULT 0 COMMENT '限流阈值，0表示不限流',
    PRIMARY KEY (id),
    UNIQUE KEY uk_route_table_path (route_path)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='路由表';

-- 玩家账号档案表，存储注册账号、密码哈希、封禁状态等
CREATE TABLE IF NOT EXISTS t_player_account (
    id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    player_id BIGINT NOT NULL COMMENT '玩家ID，雪花算法生成，全局唯一',
    username VARCHAR(64) NOT NULL COMMENT '用户名，注册时填写，唯一',
    password_hash VARCHAR(128) NOT NULL COMMENT '密码哈希值，bcrypt生成，不明文存储',
    salt VARCHAR(64) NOT NULL DEFAULT '' COMMENT '密码盐，bcrypt自带盐时存空字符串',
    status INT NOT NULL DEFAULT 1 COMMENT '账号状态：1=正常 2=封禁',
    ban_reason VARCHAR(256) NOT NULL DEFAULT '' COMMENT '封禁原因，未封禁时为空',
    ban_expire_time BIGINT NOT NULL DEFAULT 0 COMMENT '封禁过期时间戳，毫秒级，0=永久封禁',
    register_time BIGINT NOT NULL COMMENT '注册时间戳，毫秒级',
    register_ip VARCHAR(45) NOT NULL DEFAULT '' COMMENT '注册时来源IP',
    PRIMARY KEY (id),
    UNIQUE KEY uk_player_account_player_id (player_id),
    UNIQUE KEY uk_player_account_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家账号档案表';

-- 认证审计日志表，存储注册/登录/登出/封禁等操作审计记录
CREATE TABLE IF NOT EXISTS t_auth_audit_log (
    id BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    op_type INT NOT NULL COMMENT '操作类型：1=注册成功 2=登录成功 3=登录失败 4=登出 5=封禁拦截 6=暴力破解锁定',
    subject VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作主体，如用户名或玩家ID字符串',
    result TINYINT NOT NULL COMMENT '操作结果：1=成功 0=失败',
    source_ip VARCHAR(45) NOT NULL DEFAULT '' COMMENT '来源IP',
    op_time BIGINT NOT NULL COMMENT '操作时间戳，毫秒级',
    extra VARCHAR(512) NOT NULL DEFAULT '' COMMENT '附加信息，JSON格式扩展字段',
    PRIMARY KEY (id),
    KEY idx_auth_audit_log_op_time (op_time),
    KEY idx_auth_audit_log_subject (subject)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='认证审计日志表';