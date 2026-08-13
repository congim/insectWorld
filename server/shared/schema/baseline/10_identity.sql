-- Identity上下文数据库基线

CREATE TABLE t_player_account (
    id BIGINT NOT NULL AUTO_INCREMENT COMMENT '数据库自增主键',
    player_id BIGINT NOT NULL COMMENT '玩家全局ID',
    username VARCHAR(64) NOT NULL COMMENT '登录用户名',
    password_hash VARCHAR(128) NOT NULL COMMENT '密码哈希，禁止保存明文密码',
    salt VARCHAR(64) NOT NULL DEFAULT '' COMMENT '独立密码盐，哈希算法内置盐时为空',
    status INT NOT NULL DEFAULT 1 COMMENT '账号状态：1=正常 2=封禁',
    ban_reason VARCHAR(256) NOT NULL DEFAULT '' COMMENT '封禁原因，未封禁时为空',
    ban_expire_time BIGINT NOT NULL DEFAULT 0 COMMENT '封禁到期时间戳，永久封禁为0',
    register_time BIGINT NOT NULL COMMENT '注册时间戳，Unix毫秒',
    register_ip VARCHAR(45) NOT NULL DEFAULT '' COMMENT '注册来源IP地址',
    PRIMARY KEY (id),
    UNIQUE KEY uk_player_account_player_id (player_id),
    UNIQUE KEY uk_player_account_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家认证账号表';

CREATE TABLE t_auth_audit_log (
    id BIGINT NOT NULL AUTO_INCREMENT COMMENT '数据库自增主键',
    op_type INT NOT NULL COMMENT '操作类型：1=注册成功 2=登录成功 3=登录失败 4=登出 5=封禁拦截 6=暴力破解锁定',
    subject VARCHAR(64) NOT NULL DEFAULT '' COMMENT '审计主体标识',
    result TINYINT NOT NULL COMMENT '操作结果：0=失败 1=成功',
    source_ip VARCHAR(45) NOT NULL DEFAULT '' COMMENT '操作来源IP地址',
    op_time BIGINT NOT NULL COMMENT '操作时间戳，Unix毫秒',
    extra VARCHAR(512) NOT NULL DEFAULT '' COMMENT '不含敏感信息的扩展JSON',
    PRIMARY KEY (id),
    KEY idx_auth_audit_log_time (op_time),
    KEY idx_auth_audit_log_subject (subject)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='认证审计日志表';
