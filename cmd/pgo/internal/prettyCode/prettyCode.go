package prettyCode

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pancake-lee/pgo/pkg/util"
	"github.com/spf13/cobra"
)

var includeFileExtSet = map[string]bool{
	".go": true,
	".js": true,
	".ts": true,
}

// TODO 通过.gitignore 文件以及参数来排除目录和文件
var excludeDirs = []string{".git", ".vscode", "node_module", "bin", ".pb.go", "swagger"}

var PrettyCode = &cobra.Command{
	Use:   "pretty",
	Short: "pretty code",
	Long:  "Standardize all dividers to 50 characters in length.",
	Run:   run,
}

func run(cmd *cobra.Command, args []string) {
	curDir := util.GetCurDir()

	fmt.Printf("Processing files in: %s\n", curDir)

	err := filepath.WalkDir(curDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if d.IsDir() {
			return nil
		}

		// 只处理特定类型的文件
		ext := filepath.Ext(path)
		if !includeFileExtSet[ext] {
			return nil
		}

		// 跳过某些目录
		if isExcludeDirs(path) {
			return nil
		}

		return processFile(path)
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
	}
}

func isExcludeDirs(path string) bool {
	for _, excludeDir := range excludeDirs {
		if strings.Contains(path, excludeDir) {
			return true
		}
	}
	return false
}

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
		fmt.Printf("Updated dividers in: %s\n", filePath)

		output := strings.Join(lines, "\n") + "\n"
		err = os.WriteFile(filePath, []byte(output), 0644)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
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
