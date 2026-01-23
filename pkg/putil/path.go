package putil

import (
	"path"
	"path/filepath"
	"runtime"
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
	path          string
	targetOS      string
	isRel         bool
	caseSensitive bool
}

// NewPath 创建 PathHelper
func NewPath(p string) *pathHelper {
	return &pathHelper{
		path:          FixWinSlash(p), // Store internally as slash-separated
		targetOS:      "linux",
		isRel:         false,
		caseSensitive: false,
	}
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

// SetCurOS 设置为当前系统
func (h *pathHelper) SetCurOS() *pathHelper {
	h.targetOS = runtime.GOOS
	return h
}

// SetOS 设置为指定系统
func (h *pathHelper) SetOS(o string) *pathHelper {
	h.targetOS = o
	return h
}

// SetIsRel 设置是否为相对路径
func (h *pathHelper) SetIsRel(isRel bool) *pathHelper {
	h.isRel = isRel
	return h
}

// SetCaseSensitive 大小写敏感，提供set方法，默认为不敏感，如果敏感，则将统一转为小写
func (h *pathHelper) SetCaseSensitive(s bool) *pathHelper {
	h.caseSensitive = s
	return h
}

// AddPrefix 在路径前面添加前缀
func (h *pathHelper) AddPrefix(prefix string) *pathHelper {
	h.path = path.Join(prefix, h.path)
	return h
}

// Join( parts ...string ) 将各部分路径拼接成完整路径，也可以理解为AddSuffix
func (h *pathHelper) Join(parts ...string) *pathHelper {
	list := append([]string{h.path}, parts...)
	h.path = path.Join(list...)
	return h
}

// ToFile 去掉末尾的斜杠，表示文件路径
func (h *pathHelper) ToFile() *pathHelper {
	h.path = strings.TrimSuffix(h.path, "/")
	return h
}

// ToDir 在末尾加上斜杠，表示目录路径
func (h *pathHelper) ToDir() *pathHelper {
	if !strings.HasSuffix(h.path, "/") {
		h.path += "/"
	}
	return h
}

// IsDir 判断路径是否为目录
func (h *pathHelper) IsDir() bool {
	return strings.HasSuffix(h.path, "/")
}

// IsFile 判断路径是否为文件
func (h *pathHelper) IsFile() bool {
	return !h.IsDir()
}

// Clone 复制当前对象
func (h *pathHelper) Clone() *pathHelper {
	n := *h
	return &n
}

// GetPath 获取最终路径字符串
//
//			头部按isRel和os调整，特别的，如果是http或者//（smb）等特殊协议开头，则需要保留
//	     尾部按ToFile/ToDir调整
//	     中间按os调整斜杠方向，并且处理好相对路径，解析出/a/b/../c/等情况，输出/a/c/
func (h *pathHelper) GetPath() string {
	res := h.path
	if h.caseSensitive {
		res = strings.ToLower(res)
	}

	// 协议检查
	if strings.Contains(res, "://") || strings.HasPrefix(res, "//") {
		return res
	}

	// 清理路径 (处理 ..)
	hasTrailing := strings.HasSuffix(res, "/")
	res = path.Clean(res)
	if hasTrailing && res != "/" && res != "." {
		res += "/"
	}

	// 按OS调整
	if h.targetOS == "windows" {
		res = strings.ReplaceAll(res, "/", "\\")
		if !h.isRel {
			// 确保盘符
			if len(res) < 2 || res[1] != ':' {
				if strings.HasPrefix(res, "\\") {
					res = "C:" + res
				} else {
					res = "C:\\" + res
				}
			}
		} else {
			// 相对路径
			res = strings.TrimPrefix(res, "\\")
		}
	} else {
		// Linux
		if !h.isRel {
			if !strings.HasPrefix(res, "/") {
				res = "/" + res
			}
		} else {
			res = strings.TrimPrefix(res, "/")
		}
	}
	return res
}

// CutByDepth 去掉指定深度的路径部分，返回剩余路径
func (h *pathHelper) CutByDepth(depth int) *pathHelper {
	parts := h.SplitAll()
	if depth < len(parts) {
		h.path = path.Join(parts[depth:]...)
		// 如果join后丢失了原来的目录性质，暂不强求，视Join行为定
	} else {
		h.path = ""
	}
	return h
}

// GetFirst 获取路径的第一级目录名称
func (h *pathHelper) GetFirst() string {
	parts := h.SplitAll()
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// CutFirst 去掉路径的第一级目录，返回第一级目录名称
func (h *pathHelper) CutFirst() string {
	first := h.GetFirst()
	h.CutByDepth(1)
	return first
}

// GetLast 获取路径的最后一级，可能是文件也可能是目录
func (h *pathHelper) GetLast() string {
	return path.Base(h.path)
}

// CutLast 去掉路径的最后一级，返回最后一级名称
func (h *pathHelper) CutLast() string {
	last := h.GetLast()
	h.path = path.Dir(strings.TrimSuffix(h.path, "/"))
	return last
}

// GetLastFolder 获取路径的最后一个目录，如果是文件则返回其上级目录名称
func (h *pathHelper) GetLastFolder() string {
	p := h.path
	if h.IsFile() {
		p = path.Dir(p)
	}
	return path.Base(strings.TrimSuffix(p, "/"))
}

// Parent 获取路径的上一级目录
func (h *pathHelper) Parent() string {
	return path.Dir(strings.TrimSuffix(h.path, "/"))
}

// AllParents 获取路径的所有上级目录，每个元素都是完整的，不只是文件夹名
func (h *pathHelper) AllParents() []string {
	var parents []string
	p := h.path
	for {
		parent := path.Dir(strings.TrimSuffix(p, "/"))
		if parent == "." || parent == "/" || parent == p {
			break
		}
		parents = append(parents, parent)
		p = parent
	}
	return parents
}

// Base 获取路径的基础名称（不含目录和扩展名）
func (h *pathHelper) Base() string {
	base := path.Base(h.path)
	ext := path.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// Depth 获取路径的深度级别
func (h *pathHelper) Depth() int {
	return len(h.SplitAll())
}

// Split 直接调用 filepath.Split 方法，返回目录和文件部分
func (h *pathHelper) Split() (dir, file string) {
	return filepath.Split(h.path)
}

// SplitAll 类似Split，但目录部分返回字符串数组，包含所有层级文件夹名
func (h *pathHelper) SplitAll() []string {
	clean := strings.Trim(h.path, "/")
	if clean == "" {
		return []string{}
	}
	return strings.Split(clean, "/")
}

// IsParentPathOf 当前路径是否输入路径的父路径
func (h *pathHelper) IsParentPathOf(input string) bool {
	res := FixWinSlash(input)
	base := h.path
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return strings.HasPrefix(res, base)
}

// IsChildPathOf 当前路径是否输入路径的子路径
func (h *pathHelper) IsChildPathOf(input string) bool {
	res := FixWinSlash(input)
	if !strings.HasSuffix(res, "/") {
		res += "/"
	}
	return strings.HasPrefix(h.path, res)
}

// IsMatchPattern 判断当前路径是否匹配输入的模式，支持通配符*和?
func (h *pathHelper) IsMatchPattern(pattern string) bool {
	ok, _ := path.Match(pattern, h.path)
	return ok
}
