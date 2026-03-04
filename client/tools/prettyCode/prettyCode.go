package prettyCode

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pancake-lee/pgo/client/common"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	ignore "github.com/sabhiram/go-gitignore"
)

// 当前只有优化分割线一个功能，后续可以在此增加更多代码美化功能
func PrettyCode() {
	cachePath := pconfig.GetDefaultCachePath()

	includeFileExtsStr := common.GetCachedInput(cachePath,
		"tools.prettyCode.includeFileExts",
		"Include File Extensions (comma separated)",
		putil.StrListToStr(defaultIncludeFileExts, ","))

	excludeDirsStr := common.GetCachedInput(cachePath,
		"tools.prettyCode.excludeDirs",
		"Exclude Directories (comma separated)",
		putil.StrListToStr(defaultExcludeDirs, ","))

	// --------------------------------------------------
	includeFileExts = putil.StrToStrList(includeFileExtsStr, ",")
	excludeDirs = putil.StrToStrList(excludeDirsStr, ",")
	curDir := putil.GetCurDir()

	// --------------------------------------------------
	// 显示配置信息
	plogger.Debugf("Processing files in: %s", curDir)
	plogger.Debugf("Include extensions : %v", includeFileExts)
	plogger.Debugf("Exclude directories: %v", excludeDirs)

	// --------------------------------------------------
	// 初始化 gitignore 处理器
	initGitignore(curDir)

	err := filepath.WalkDir(curDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if d.IsDir() {
			return nil
		}

		// 只处理特定类型的文件
		if !isIncludeExt(path) {
			return nil
		}

		if isExcludeFile(path, curDir) {
			return nil
		}

		if isExcludeDir(path) {
			return nil
		}

		return processFile(path)
	})

	if err != nil {
		plogger.Debugf("Error walking directory: %v", err)
	}
}

// --------------------------------------------------
var gitignoreHandler *ignore.GitIgnore

// 初始化 gitignore 处理器
func initGitignore(rootDir string) {
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	_, err := os.Stat(gitignorePath)
	if err != nil {
		return
	}
	gitignoreHandler, err = ignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return
	}
	plogger.Debugf("Using .gitignore rules for file exclusion")
}

// 检查文件是否应该被排除
func isExcludeFile(filePath, rootDir string) bool {
	// 获取相对路径
	relPath, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		return false
	}

	// 使用 Unix 风格的路径分隔符
	relPath = filepath.ToSlash(relPath)

	// 如果有 gitignore 处理器，使用它
	if gitignoreHandler != nil {
		if gitignoreHandler.MatchesPath(relPath) {
			return true
		}
	}
	return false
}

// --------------------------------------------------
var defaultIncludeFileExts = []string{"go", "js", "ts"}

// 默认排除的目录和文件模式
var defaultExcludeDirs = []string{".git", ".vscode", "node_modules", "bin", ".pb.go", "swagger"}

// 运行时使用的排除规则
var includeFileExts []string
var excludeDirs []string

func isExcludeDir(path string) bool {
	return slices.ContainsFunc(excludeDirs, func(excludeDir string) bool {
		return strings.HasPrefix(path, excludeDir)
	})
}

func isIncludeExt(path string) bool {
	e := filepath.Ext(path)
	e = strings.TrimPrefix(e, ".")
	return slices.Contains(includeFileExts, e)
}

// --------------------------------------------------
func processFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	modified := false

	for scanner.Scan() {
		line := scanner.Text()

		newLine, isModified := processLine(line)
		if isModified {
			lines = append(lines, newLine)
			modified = true
		} else {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 如果文件有修改，写回文件
	if modified {
		plogger.Debugf("Updated dividers in: %s", filePath)

		output := strings.Join(lines, "\n") + "\n"
		err = os.WriteFile(filePath, []byte(output), 0644)
		if err != nil {
			plogger.Errorf("Failed to write file %s: %v", filePath, err)
			return err
		}
	}

	return nil
}

// 格式化分割线，返回新行和是否修改的标志
func processLine(line string) (string, bool) {

	otherRune := strings.IndexFunc(line, func(r rune) bool {
		return r != '-' && r != '/' && r != '*' && r != ' ' && r != '\t'
	})
	if otherRune != -1 {
		return line, false // 如果行中有其他字符，则不处理
	}

	lineIndex := strings.IndexRune(line, '-')
	if lineIndex == -1 {
		return line, false // 如果没有找到分割线，则不处理
	}

	indentEnd := strings.IndexFunc(line, func(r rune) bool {
		return r == '-' || r == '/' || r == '*'
	})
	if indentEnd == -1 {
		return line, false // 如果没有找到分割线的起始位置，则不处理
	}

	indent := line[:indentEnd] // 获取缩进部分

	trimmed := strings.ReplaceAll(line, "\t", "")
	trimmed = strings.ReplaceAll(trimmed, " ", "")

	// 匹配 // ------
	if strings.HasPrefix(trimmed, "//") {
		return fmt.Sprintf("%s// %s", indent, strings.Repeat("-", 50)), true
	}

	// 匹配 /*------------
	if strings.HasPrefix(trimmed, "/*") && !strings.HasSuffix(trimmed, "*/") {
		return fmt.Sprintf("%s/* %s", indent, strings.Repeat("-", 50)), true
	}

	// 匹配 ----------*/
	if !strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
		return fmt.Sprintf("%s%s */", indent, strings.Repeat("-", 50)), true
	}

	// 匹配 /*------*/
	if strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
		return fmt.Sprintf("%s/* %s */", indent, strings.Repeat("-", 50)), true
	}

	return line, false
}
