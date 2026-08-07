-- 迁移脚本：V002 创建联盟外交关联表索引
-- 对应DDL: shared/schema/ddl/social.sql
-- 迁移脚本命名规范：V<3位版本号>__<蛇形描述>.sql

-- 添加联盟外交关系按到期时间的索引，用于定期清理过期关系
CREATE INDEX idx_alliance_diplomacy_expire ON t_alliance_diplomacy_rel (expire_time);