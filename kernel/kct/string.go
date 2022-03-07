package kct

// 使用给定的符号进行字符串切割，和strings.Split()不同，如果连续2个符号，那么视为一个
// CutWith("adsf  ff f",' ')->{"adsf","ff","f"}
func CutWith(str string,cut byte) []string {
	var rs []string
	startIdx := -1
	for i := 0; i < len(str); i++ {
		if str[i] == cut && startIdx >= 0 {
			rs = append(rs, str[startIdx:i])
			startIdx = -1
		} else if str[i] != cut && startIdx < 0 {
			startIdx = i
		}
	}
	if startIdx >= 0 {
		rs = append(rs, str[startIdx:])
	}
	return rs
}
