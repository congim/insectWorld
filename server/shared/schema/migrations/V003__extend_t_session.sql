-- V003 扩展t_session表，补全会话状态字段以支持用户认证服务整合
-- 对齐spec 6.3 在线会话领域对象，新增token_version与device_id字段
-- 向前兼容：新增字段均有默认值，存量数据不受影响

-- 新增令牌版本号字段，用于令牌黑名单与单点登录踢下线
ALTER TABLE t_session ADD COLUMN token_version INT NOT NULL DEFAULT 1 COMMENT '令牌版本号，登出/踢下线时递增使旧令牌失效';

-- 新增设备ID字段，用于多端识别与单点登录策略
ALTER TABLE t_session ADD COLUMN device_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '设备ID，标识客户端设备，空字符串表示未指定';

-- 统一status字段注释为对齐spec 6.3的状态机定义
ALTER TABLE t_session MODIFY COLUMN status INT NOT NULL COMMENT '会话状态：1=活跃 2=待销毁';