package prettyCode

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pancake-lee/pgo/client/common"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	ignore "github.com/sabhiram/go-gitignore"
)

// 当前只有优化分割线一个功能，后续可以在此增加更多代码美化功能

// --------------------------------------------------
const (
	paramNameRootDir = "dir"
	paramNameInclude = "include"
	paramNameExclude = "exclude"
	cacheKeyPrefix   = "tools.prettyCode."
)

// 默认排除的目录和文件模式
var defaultExcludeDirs = []string{
	".git", ".vscode", "node_modules", "bin", ".pb.go", "swagger"}
var defaultIncludeFileExts = []string{"go", "js", "ts"}

var paramSettingList = []common.ParamItem{
	{
		Name:    paramNameRootDir,
		Usage:   "root directory to process",
		Default: putil.GetCurDir(),
	}, {
		Name:    paramNameExclude,
		Usage:   "comma separated exclude directories",
		Default: strings.Join(defaultExcludeDirs, ","),
	}, {
		Name:    paramNameInclude,
		Usage:   "comma separated file extensions",
		Default: strings.Join(defaultIncludeFileExts, ","),
	}}

var Entrypoint = common.NewToolEntrypoint(common.ToolEntrypointOption{
	ToolName:       "prettyCode",
	Use:            "pretty",
	Aliases:        []string{"prettyCode"},
	Short:          "美化代码分割线注释",
	CacheKeyPrefix: cacheKeyPrefix,
	ParamList:      paramSettingList,
	Run:            Run,
})

// --------------------------------------------------
// 运行参数，定义的参数列表最终转换成当前程序使用的运行选项

type RunOptions struct {
	RootDir         string
	IncludeFileExts []string
	ExcludeDirs     []string
}

// cobra参数值转换为“当前程序的”运行选项
func convParamToRunOpt(values common.ParamMap) RunOptions {
	rootDir := values[paramNameRootDir]
	if rootDir == "" {
		rootDir = putil.GetCurDir()
	}

	return RunOptions{
		RootDir:         rootDir,
		IncludeFileExts: putil.StrToStrList(values[paramNameInclude], ","),
		ExcludeDirs:     putil.StrToStrList(values[paramNameExclude], ","),
	}
}

// --------------------------------------------------
func Run(values common.ParamMap) error {
	options := convParamToRunOpt(values)

	if options.RootDir == "" {
		return errors.New("root dir is empty")
	}
	plogger.Debugf("Processing files in: %s", options.RootDir)
	plogger.Debugf("Include extensions : %v", options.IncludeFileExts)
	plogger.Debugf("Exclude directories: %v", options.ExcludeDirs)

	// --------------------------------------------------
	// 初始化 gitignore 处理器
	initGitignore(options.RootDir)

	err := filepath.WalkDir(options.RootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if d.IsDir() {
			return nil
		}

		// 只处理特定类型的文件
		if !isIncludeExt(path, options.IncludeFileExts) {
			return nil
		}

		if isExcludeFile(path, options.RootDir) {
			return nil
		}

		if isExcludeDir(path, options.ExcludeDirs) {
			return nil
		}

		return processFile(path)
	})

	if err != nil {
		return fmt.Errorf("walk dir failed: %w", err)
	}

	return nil
}

// --------------------------------------------------
var gitignoreHandler *ignore.GitIgnore

// 初始化 gitignore 处理器
func initGitignore(rootDir string) {
	gitignoreHandler = nil
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
func isExcludeDir(path string, excludeDirs []string) bool {
	return slices.ContainsFunc(excludeDirs, func(excludeDir string) bool {
		return strings.HasPrefix(path, excludeDir)
	})
}

func isIncludeExt(path string, includeFileExts []string) bool {
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
