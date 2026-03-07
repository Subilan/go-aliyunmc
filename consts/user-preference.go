package consts

// UserPreferenceKey 表示用户偏好设置的键
type UserPreferenceKey string

const (
	// UserPreferenceParticipateInPlayTimeRanking 表示是否参与游戏时间排行
	UserPreferenceParticipateInPlayTimeRanking UserPreferenceKey = "participate_in_play_time_ranking"
)

// Valid 检查偏好设置键是否为有效的枚举值
func (k UserPreferenceKey) Valid() bool {
	switch k {
	case UserPreferenceParticipateInPlayTimeRanking:
		return true
	default:
		return false
	}
}
