-- 迁移脚本：V001 初始化World服务表结构
-- 对应DDL: shared/schema/ddl/world.sql
-- 迁移脚本命名规范：V<3位版本号>__<蛇形描述>.sql
-- 由Persist服务的migration执行器按版本号顺序执行

-- 添加玩家最后登录时间字段（示例迁移）
ALTER TABLE t_player ADD COLUMN last_login_ip VARCHAR(45) DEFAULT '' COMMENT '最后登录IP' AFTER last_login_time;