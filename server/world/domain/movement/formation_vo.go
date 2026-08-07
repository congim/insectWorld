// Package movement 移动订单聚合根，维护移动路径与状态。
// 本文件定义FormationVO编队值对象，封装编队队形与速度修正。
package movement

// FormationVO 编队值对象，封装编队队形、成员数校验与速度修正。
// 编队无独立生命周期，使用值对象而非聚合根（规范4）。
type FormationVO struct {
	FormationID     string  // 编队队形ID，对应movement.json的编队队形配置
	MemberCount     int     // 编队成员数，校验min_members<=memberCount<=max_members
	SpeedModifier   float64 // 速度修正系数，由配置注入（规范8配置数值例外，float64）
	MemberEntityIDs []int64 // 编队成员实体ID列表
	MinMembers      int     // 最小成员数，由配置注入
	MaxMembers      int     // 最大成员数，由配置注入
}

// NewFormation 创建编队值对象实例。
func NewFormation(formationID string, memberEntityIDs []int64, minMembers, maxMembers int, speedModifier float64) *FormationVO {
	return &FormationVO{
		FormationID:     formationID,
		MemberCount:     len(memberEntityIDs),
		MemberEntityIDs: memberEntityIDs,
		MinMembers:      minMembers,
		MaxMembers:      maxMembers,
		SpeedModifier:   speedModifier,
	}
}

// Validate 校验编队成员数是否在配置的[min, max]范围内。
func (f *FormationVO) Validate() bool {
	return f.MemberCount >= f.MinMembers && f.MemberCount <= f.MaxMembers
}

// GetSpeedModifier 返回速度修正系数。
func (f *FormationVO) GetSpeedModifier() float64 {
	return f.SpeedModifier
}

// ContainsEntity 判断实体是否在编队中。
func (f *FormationVO) ContainsEntity(entityID int64) bool {
	for _, id := range f.MemberEntityIDs {
		if id == entityID {
			return true
		}
	}
	return false
}
