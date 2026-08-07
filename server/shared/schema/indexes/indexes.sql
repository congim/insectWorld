-- 索引定义集中管理
-- 本目录集中存放全服务端的索引定义SQL文件
-- 索引命名规范（规范2）：普通索引idx_<表名去t_>_<字段>，唯一索引uk_<表名去t_>_<字段>，外键fk_<表名去t_>_<字段>
-- 建表时的索引已在DDL中定义，本目录存放后续新增的索引

-- Outbox表性能优化索引
-- 按状态+创建时间查询待投递事件，提升Outbox轮询效率
CREATE INDEX idx_outbox_status_create ON t_outbox (status, create_time);

-- 战斗表按状态+开始时间查询，提升进行中战斗查询效率
CREATE INDEX idx_combat_status_start ON t_combat (status, start_time);

-- 移动订单按状态+创建时间查询，提升移动中订单查询效率
CREATE INDEX idx_movement_order_status_create ON t_movement_order (status, start_time);