package putil

import (
	"strings"
	"testing"
)

func TestPathHelper_Scene1_OS_System(t *testing.T) {
	// 1. Windows Absolute
	// 默认非 relative
	p := NewPath("/a/b/c").SetOS("windows")
	if got := p.GetPath(); got != "C:\\a\\b\\c" {
		t.Errorf("Win Abs 1 failed: want C:\\a\\b\\c, got %s", got)
	}

	// 补全 C:\
	p = NewPath("program/files").SetOS("windows")
	if got := p.GetPath(); got != "C:\\program\\files" {
		t.Errorf("Win Abs 2 failed: want C:\\program\\files, got %s", got)
	}

	// 保留盘符
	p = NewPath("D:\\game\\data").SetOS("windows")
	if got := p.GetPath(); got != "D:\\game\\data" {
		t.Errorf("Win Abs 3 failed: want D:\\game\\data, got %s", got)
	}

	// 2. Windows Relative
	p = NewPath("/a/b/c").SetOS("windows").SetIsRel(true)
	if got := p.GetPath(); got != "a\\b\\c" {
		t.Errorf("Win Rel 1 failed: want a\\b\\c, got %s", got)
	}

	// 3. Linux Absolute
	// 默认 Linux, 默认非 Rel
	p = NewPath("etc/nginx")
	if got := p.GetPath(); got != "/etc/nginx" {
		t.Errorf("Linux Abs 1 failed: want /etc/nginx, got %s", got)
	}

	p = NewPath("/var/log/")
	if got := p.GetPath(); got != "/var/log/" {
		t.Errorf("Linux Abs 2 failed: want /var/log/, got %s", got)
	}

	// 4. Linux Relative
	p = NewPath("/usr/local/bin").SetIsRel(true)
	if got := p.GetPath(); got != "usr/local/bin" {
		t.Errorf("Linux Rel 1 failed: want usr/local/bin, got %s", got)
	}
}

func TestPathHelper_Scene2_SpecialProtocols(t *testing.T) {
	// Http
	url := "http://example.com/image.png"
	p := NewPath(url)
	if got := p.GetPath(); got != url {
		t.Errorf("Http failed: want %s, got %s", url, got)
	}

	// SMB / UNC
	// Note: FixWinSlash preserves "\\" if they are consecutive.
	unc := "\\\\server\\share\\file.txt"
	// NewPath calls FixWinSlash.

	p = NewPath(unc)
	// Expect behavior: Double slash preserved at start, others become forward slash internally,
	// but GetPath returns res as-is if it has "//" prefix.

	got := p.GetPath()
	// UNC should start with \\ because FixWinSlash preserves it and GetPath respects it
	if !strings.HasPrefix(got, "\\\\") {
		t.Errorf("UNC check failed, got %s", got)
	}
}

func TestPathHelper_Scene3_LinuxManipulation(t *testing.T) {
	// 默认 Linux
	p := NewPath("/home/user/docs")

	// 1. ToFile / ToDir / IsDir / IsFile
	p.ToDir()
	if !p.IsDir() {
		t.Error("ToDir failed to set dir flag (slash)")
	}
	if p.GetPath() != "/home/user/docs/" {
		t.Errorf("ToDir output wrong: %s", p.GetPath())
	}

	p.ToFile()
	if !p.IsFile() {
		t.Error("ToFile failed")
	}
	if p.GetPath() != "/home/user/docs" {
		t.Errorf("ToFile output wrong: %s", p.GetPath())
	}

	// 2. Join & Parent & Base
	p.Join("photos", "2023", "aaa", "..")
	// now /home/user/docs/photos/2023
	if !strings.HasSuffix(p.GetPath(), "2023") {
		t.Errorf("Join failed: %s", p.GetPath())
	}

	parent := p.Parent() // should be /home/user/docs/photos
	if !strings.HasSuffix(parent, "photos") {
		t.Errorf("Parent failed: %s", parent)
	}

	base := p.Base() // 2023
	if base != "2023" {
		t.Errorf("Base failed: %s", base)
	}

	// 3. Cut / Get First/Last
	// p is /home/user/docs/photos/2023
	first := p.GetFirst() // home
	if first != "home" {
		t.Errorf("GetFirst want home, got %s", first)
	}

	cutFirst := p.CutFirst() // returns home, path becomes user/docs/photos/2023
	if cutFirst != "home" {
		t.Errorf("CutFirst return wrong: %s", cutFirst)
	}
	// Note: p internal path is now "user/docs/photos/2023" (relative internally?)
	// But GetPath adds "/" if !isRel and linux.
	if p.GetPath() != "/user/docs/photos/2023" {
		t.Errorf("After CutFirst path wrong: %s", p.GetPath())
	}

	last := p.GetLast() // 2023
	if last != "2023" {
		t.Errorf("GetLast return wrong: %s", last)
	}

	cutLast := p.CutLast() // returns 2023, path becomes /user/docs/photos
	if cutLast != "2023" {
		t.Errorf("CutLast return wrong: %s", cutLast)
	}
	if p.GetPath() != "/user/docs/photos" {
		t.Errorf("After CutLast path wrong: %s", p.GetPath())
	}
}

func TestPathHelper_Scene4_Manipulation_Advanced(t *testing.T) {
	p := NewPath("/a/b/c/d/e")

	// CutByDepth
	p.CutByDepth(2) // remove a, b. -> c/d/e
	if p.GetPath() != "/c/d/e" {
		t.Errorf("CutByDepth failed: %s", p.GetPath())
	}

	// IsParent / Child
	master := NewPath("/var/www")
	child := "/var/www/html/index.html"
	if !master.IsParentPathOf(child) {
		t.Error("IsParentPathOf failed positive case")
	}
	if master.IsParentPathOf("/var/lib") {
		t.Error("IsParentPathOf failed negative case")
	}

	// Split
	dir, file := NewPath("/var/log/syslog").Split()
	if file != "syslog" {
		t.Error("Split file wrong")
	}
	// filepath.Split behaves differently depending on trailing slash, but NewPath cleans slashes usually via internal logic? No, NewPath just stores.
	// Split calls filepath.Split(h.path).
	// h.path is "/var/log/syslog"
	// dir should be "/var/log/"
	if dir != "/var/log/" {
		t.Errorf("Split dir wrong: %s", dir)
	}
}

func TestPathHelper_Scene5_FileType(t *testing.T) {
	img := NewPath("photo.JPG") // Case check
	if !img.IsImage() {
		t.Error("IsImage failed for JPG")
	}
	if img.MIME() != "image/jpeg" {
		t.Errorf("MIME failed: %s", img.MIME())
	}

	vid := NewPath("movie.mp4")
	if !vid.IsVideo() {
		t.Error("IsVideo failed")
	}
}

// 补充：测试 CaseSensitive
func TestPathHelper_CaseSensitive(t *testing.T) {
	p := NewPath("/My/Path").SetCaseSensitive(true)
	if got := p.GetPath(); got != "/my/path" {
		t.Errorf("CaseSensitive failed: %s", got)
	}
}
