// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Economy服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TResourceAccountBalance 稳定字符串资源ID余额表
	TResourceAccountBalance = "t_resource_account_balance"
	// TResourceOperation 资源变更幂等操作账本
	TResourceOperation = "t_resource_operation"
)
