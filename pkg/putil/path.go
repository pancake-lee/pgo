package putil

import "strings"

// FixWinSlash 将路径中的反斜杠替换为正斜杠，go能在win下正确使用正斜杠路径。
// 该函数可以处理整段文本，例如完整的 JSON 字符串。
// 特别的，保留每个连续的两字符反斜杠对 "\\\\"（通常表示 UNC/SMB 的开头），
// 但也需要注意路径中本来就拼接了双反斜杠的情况，比如C:\a\\b.txt。
func FixWinSlash(path string) string {
	var b strings.Builder
	b.Grow(len(path))
	for i := 0; i < len(path); {
		if path[i] == '\\' {
			// 双反斜杠保留
			if i+1 < len(path) && path[i+1] == '\\' {
				b.WriteString("\\\\")
				i += 2
				continue
			}
			// 单反斜杠替换为 '/'
			b.WriteByte('/')
			i++
			continue
		}
		b.WriteByte(path[i])
		i++
	}
	return b.String()
}
