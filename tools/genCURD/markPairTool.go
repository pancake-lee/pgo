package main

import (
	"strings"

	"github.com/pancake-lee/pgo/pkg/plogger"
)

type iMarkPairTool interface {
	ReplaceAll(mark, fileStr, replaceStr string) string
	RemoveMarkSelf(mark, fileStr string) string
	GetContent(fileStr string, mark string) string
	ReplaceLoop(mark, fileStr string, count int, replaceFunc func(int, string) string) string
}
type _markPairTool struct{}

var markPairTool iMarkPairTool = new(_markPairTool)

// --------------------------------------------------
// 循环替换
func (r *_markPairTool) ReplaceLoop(mark, fileStr string, count int, replaceFunc func(int, string) string) string {
	if count == 0 {
		return r.ReplaceAll(mark, fileStr, "")
	}

	// 获取标记内的原始内容
	content := r.GetContent(fileStr, mark)
	if content == "" {
		return fileStr
	}

	newContent := ""
	for i := 0; i < count; i++ {
		newContent += replaceFunc(i, content)
	}

	return r.ReplaceAll(mark, fileStr, newContent)
}

// --------------------------------------------------
// 替换标记的内容
func (r *_markPairTool) ReplaceAll(mark, fileStr, replaceStr string) string {
	for strings.Contains(fileStr, mark) {
		fileStr = r.replaceOnce(mark, fileStr, replaceStr)
	}
	return fileStr
}

func (r *_markPairTool) replaceOnce(mark, fileStr, replaceStr string) string {
	iStart := r.indexBeforeMark(fileStr, mark+" START")
	iEnd := r.indexAfterMark(fileStr, mark+" END")
	if iStart == -1 || iEnd == -1 {
		plogger.Debugf("mark not found, mark: %v\n", mark)
		return fileStr
	}
	return fileStr[:iStart] + replaceStr + fileStr[iEnd:]
}

// --------------------------------------------------
// 标记无需操作，删掉标记注释本身
func (r *_markPairTool) RemoveMarkSelf(mark, fileStr string) string {
	for strings.Contains(fileStr, mark) {
		fileStr = r.removeMarkSelfOnce(mark, fileStr)
	}
	return fileStr
}
func (r *_markPairTool) removeMarkSelfOnce(mark, fileStr string) string {
	lastOfStart := r.indexBeforeMark(fileStr, mark+" START")
	nextOfStart := r.indexAfterMark(fileStr, mark+" START")
	lastOfEnd := r.indexBeforeMark(fileStr, mark+" END")
	nextOfEnd := r.indexAfterMark(fileStr, mark+" END")
	if lastOfStart == -1 || nextOfStart == -1 || lastOfEnd == -1 || nextOfEnd == -1 {
		plogger.Debugf("mark not found, mark: %v\n", mark)
		return fileStr
	}
	return fileStr[:lastOfStart] + fileStr[nextOfStart:lastOfEnd] + fileStr[nextOfEnd:]
}

// --------------------------------------------------
// 获取标记START和END之间的内容
func (r *_markPairTool) GetContent(fileStr string, mark string) string {
	nextOfStart := r.indexAfterMark(fileStr, mark+" START")
	lastOfEnd := r.indexBeforeMark(fileStr, mark+" END")
	if nextOfStart == -1 || lastOfEnd == -1 {
		plogger.Debugf("mark not found, mark: %v\n", mark)
		return ""
	}
	return fileStr[nextOfStart:lastOfEnd]
}

// --------------------------------------------------
// 从头开始找第一个mark
// 返回“指定mark标记的行”的上一行的行末
func (r *_markPairTool) indexBeforeMark(fileStr, mark string) int {
	i := strings.Index(fileStr, mark)
	if i == -1 {
		plogger.Debugf("mark not found, mark: %v\n", mark)
		return -1
	}
	return strings.LastIndex(fileStr[:i], "\n") + 1
}

// 从头开始找第一个mark
// 返回"指定mark标记的行"的下一行的行头
func (r *_markPairTool) indexAfterMark(fileStr, mark string) int {
	i := strings.Index(fileStr, mark)
	if i == -1 {
		plogger.Debugf("mark not found, mark: %v\n", mark)
		return -1
	}
	return i + strings.Index(fileStr[i:], "\n") + 1
}
