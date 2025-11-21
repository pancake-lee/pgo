package putil

import (
	"path/filepath"
	"strings"
)

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

// FileType 表示文件类型枚举
type FileType int

const (
	FileTypeOther FileType = iota
	FileTypeImage
	FileTypeVideo
)

// pathHelper 提供对路径的简单判断方法
type pathHelper struct {
	path string
}

// NewPath 创建 PathHelper
func NewPath(p string) *pathHelper {
	return &pathHelper{path: p}
}

// 全局后缀 -> MIME 映射，以及按类型分类的后缀集合
var (
	imageExtToMime = map[string]string{
		// images
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"bmp":  "image/bmp",
		"webp": "image/webp",
		"tif":  "image/tiff",
		"tiff": "image/tiff",
		"svg":  "image/svg+xml",
	}
	videoExtToMime = map[string]string{
		// videos
		"mp4":  "video/mp4",
		"mov":  "video/quicktime",
		"avi":  "video/x-msvideo",
		"mkv":  "video/x-matroska",
		"flv":  "video/x-flv",
		"wmv":  "video/x-ms-wmv",
		"webm": "video/webm",
	}
	otherExtToMime = map[string]string{
		// others common
		"pdf":  "application/pdf",
		"txt":  "text/plain",
		"csv":  "text/csv",
		"json": "application/json",
	}
	allExtToMime = map[string]string{}
)

func init() {
	// 合并所有后缀映射
	for k, v := range imageExtToMime {
		allExtToMime[k] = v
	}
	for k, v := range videoExtToMime {
		allExtToMime[k] = v
	}
	for k, v := range otherExtToMime {
		allExtToMime[k] = v
	}
}

// Type 根据文件后缀判断文件类型（Image/Video/Other）
func (h *pathHelper) Type() FileType {
	ext := h.ExtLower()
	if _, ok := imageExtToMime[ext]; ok {
		return FileTypeImage
	}
	if _, ok := videoExtToMime[ext]; ok {
		return FileTypeVideo
	}
	return FileTypeOther
}

// IsImage 快捷判断是否为图片
func (h *pathHelper) IsImage() bool {
	return h.Type() == FileTypeImage
}

// IsVideo 快捷判断是否为视频
func (h *pathHelper) IsVideo() bool {
	return h.Type() == FileTypeVideo
}

// https://mime.wcode.net/zh-hans/
func (h *pathHelper) MIME() string {
	ext := h.ExtLower()
	if m, ok := allExtToMime[ext]; ok {
		return m
	}
	switch h.Type() {
	case FileTypeImage:
		return "image/*"
	case FileTypeVideo:
		return "video/*"
	default:
		return "application/octet-stream"
	}
}

func (h *pathHelper) Ext() string {
	return strings.TrimPrefix(filepath.Ext(h.path), ".")
}

func (h *pathHelper) ExtLower() string {
	return strings.ToLower(h.Ext())
}
