package putil

import (
	"strings"
	"time"
)

func timeFormatToGoFormat(format string) string {
	//yyyy-MM-dd HH:mm:ss.fff
	//2006-01-02 15:04:05.000
	format = strings.ReplaceAll(format, "YYYY", "2006")
	format = strings.ReplaceAll(format, "yyyy", "2006")
	format = strings.ReplaceAll(format, "YY", "06")
	format = strings.ReplaceAll(format, "yy", "06")

	format = strings.ReplaceAll(format, "MM", "01")

	format = strings.ReplaceAll(format, "DD", "02")
	format = strings.ReplaceAll(format, "dd", "02")

	format = strings.ReplaceAll(format, "HH", "15")
	format = strings.ReplaceAll(format, "hh", "15")

	format = strings.ReplaceAll(format, "mm", "04")

	format = strings.ReplaceAll(format, "SS", "05")
	format = strings.ReplaceAll(format, "ss", "05")
	format = strings.ReplaceAll(format, "FFF", "000")
	format = strings.ReplaceAll(format, "fff", "000")
	return format
}

func TimeFromStrDefault(timeStr string) (time.Time, error) {
	return TimeFromStr(timeStr, "YYMMDDTHH:mm:ss")
}
func TimeFromStr(timeStr, format string) (time.Time, error) {
	format = timeFormatToGoFormat(format)
	return time.Parse(format, timeStr)
}

func TimeToStrDefault(t time.Time) string {
	return TimeToStr(t, "YYMMDDTHH:mm:ss")
}
func TimeToStr(t time.Time, format string) string {
	format = timeFormatToGoFormat(format)
	return t.Format(format)
}
