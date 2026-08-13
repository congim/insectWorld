// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Social服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TAlliance 联盟表，存储联盟基本信息与属性
	TAlliance = "t_alliance"
	// TPlayer 玩家表，存储玩家基本信息与状态
	TPlayer = "t_player"
	// TAllianceMemberRel 联盟成员关联表，存储玩家与联盟的成员关系
	TAllianceMemberRel = "t_alliance_member_rel"
	// TAllianceDiplomacyRel 联盟外交关联表，存储联盟间的外交关系
	TAllianceDiplomacyRel = "t_alliance_diplomacy_rel"
	// TWelfareRecord 福利记录表，存储联盟福利发放记录
	TWelfareRecord = "t_welfare_record"
)
