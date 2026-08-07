-- 初始数据集中管理
-- 本目录集中存放全服务端的初始数据（种子数据）SQL文件
-- 初始数据在数据库初始化时执行，由Persist服务的migration执行器管理

-- 初始化路由表基础配置
INSERT INTO t_route_table (id, route_path, target_service, method, rate_limit) VALUES
(1, '/world/map', 'world', 'GET', 1000),
(2, '/world/move', 'world', 'POST', 500),
(3, '/combat/execute', 'combat', 'POST', 200),
(4, '/economy/collect', 'economy', 'POST', 500),
(5, '/social/alliance', 'social', 'GET', 1000),
(6, '/operation/season', 'operation', 'GET', 1000),
(7, '/config/query', 'config', 'GET', 2000);

-- 初始化资源类型配置（如需静态资源类型定义）
-- 资源类型：1=粮食 2=木材 3=石料 4=铁矿 5=金币