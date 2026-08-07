// Package persistence Social服务持久化层。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

const (
	TableAlliance             = tables.TAlliance             // 联盟表
	TablePlayer               = tables.TPlayer               // 玩家表
	TableAllianceMemberRel    = tables.TAllianceMemberRel    // 联盟成员关联表
	TableAllianceDiplomacyRel = tables.TAllianceDiplomacyRel // 联盟外交关联表
	TableWelfareRecord        = tables.TWelfareRecord        // 福利记录表
)
