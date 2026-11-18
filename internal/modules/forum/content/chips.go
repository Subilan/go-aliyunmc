package content

// MentionIDs 返回正文中提及 chip 指向的用户 ID（按文档顺序去重）。
// 只识别 data-tag-type="mention" 的 span，class 不参与判断。
func MentionIDs(input string) []uint {
	return Parse(input).MentionIDs()
}

// NormalizeMentions 依据有效的用户集合重建提及 chip，并返回有效提及的用户 ID。
// 无效提及会降级为纯文本；没有提及节点时原样返回输入。
func NormalizeMentions(input string, usernames map[uint]string) (string, []uint) {
	return Parse(input).NormalizeMentions(usernames)
}

// ReplyFloors 返回正文中楼层引用 chip 的楼层号，按文档顺序。
// 只识别 data-tag-type="reply" 的 span。
func ReplyFloors(input string) []uint {
	return Parse(input).Floors()
}
