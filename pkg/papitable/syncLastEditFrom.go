package papitable

import "time"

// --------------------------------------------------
// 用于避免LTBL和MTBL相互修改的回调无限循环同步

var LastEditFrom_TEMP string = "TEMP"
var LastEditFrom_LTBL string = "LTBL"
var LastEditFrom_MTBL string = "MTBL"
var lastEditOptHandler SelectFieldOptionHandler

func init() {
	lastEditOptHandler.RegOptionS(LastEditFrom_TEMP, LastEditFrom_TEMP, ColorRed)
	lastEditOptHandler.RegOptionS(LastEditFrom_LTBL, LastEditFrom_LTBL, ColorPurple)
	lastEditOptHandler.RegOptionS(LastEditFrom_MTBL, LastEditFrom_MTBL, ColorBlue)
}

// --------------------------------------------------
// 用于避免LTBL或MTBL自己单方的回调无限循环同步

type LastEditFromMarker struct {
	keyToTimeMap map[string]time.Time
}

// local table
var LastEditFromMarker_LTBL = &LastEditFromMarker{
	keyToTimeMap: make(map[string]time.Time)}

// multi table
var LastEditFromMarker_MTBL = &LastEditFromMarker{
	keyToTimeMap: make(map[string]time.Time)}

func init() {
	// 定期清理过期的记录
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			LastEditFromMarker_LTBL.CleanTimeoutRecord()
			LastEditFromMarker_MTBL.CleanTimeoutRecord()
		}
	}()
}

func (m *LastEditFromMarker) CleanTimeoutRecord() {
	for k, v := range m.keyToTimeMap {
		if time.Since(v) > 5*time.Minute {
			delete(m.keyToTimeMap, k)
		}
	}
}

func (m *LastEditFromMarker) Set(key string) {
	m.keyToTimeMap[key] = time.Now()
}

func (m *LastEditFromMarker) SkipAndClean(key string, updateTime time.Time) bool {
	t, ok := m.keyToTimeMap[key]
	if ok && updateTime.Sub(t).Abs() < 1*time.Second {
		// 其实updateTime一定是晚于t的，abs纯粹保险
		delete(m.keyToTimeMap, key)
		return true
	}
	return false
}
