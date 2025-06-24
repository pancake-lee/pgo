package prettyCode

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pancake-lee/pgo/pkg/logger"
	"github.com/pancake-lee/pgo/pkg/util"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/cobra"
)

var PrettyCode = &cobra.Command{
	Use:   "pretty",
	Short: "pretty code",
	Long:  "Standardize all dividers to 50 characters in length.",
	Run:   run,
}

func init() {
	// 添加命令行参数
	PrettyCode.Flags().StringSliceVar(&excludeDirs, "exclude", defaultExcludeDirs, "Directories to exclude")
	PrettyCode.Flags().StringSliceVar(&includeFileExts, "include", defaultIncludeFileExts, "File extensions to include")
}

func run(cmd *cobra.Command, args []string) {
	curDir := util.GetCurDir()

	logger.Debugf("Processing files in: %s", curDir)

	// 显示配置信息
	logger.Debugf("Include extensions : %v", includeFileExts)
	logger.Debugf("Exclude directories: %v", excludeDirs)

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
		logger.Debugf("Error walking directory: %v", err)
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
	logger.Debugf("Using .gitignore rules for file exclusion")
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
var defaultIncludeFileExts = []string{".go", ".js", ".ts"}

// 默认排除的目录和文件模式
var defaultExcludeDirs = []string{".git", ".vscode", "node_modules", "bin", ".pb.go", "swagger"}

// 运行时使用的排除规则
var includeFileExts []string
var excludeDirs []string

func isExcludeDir(path string) bool {
	return slices.ContainsFunc(excludeDirs, func(excludeDir string) bool {
		return strings.Contains(path, excludeDir)
	})
}

func isIncludeExt(path string) bool {
	e := filepath.Ext(path)
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
		logger.Debugf("Updated dividers in: %s", filePath)

		output := strings.Join(lines, "\n") + "\n"
		err = os.WriteFile(filePath, []byte(output), 0644)
		if err != nil {
			logger.Errorf("Failed to write file %s: %v", filePath, err)
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
