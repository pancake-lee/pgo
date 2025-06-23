package prettyCode

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pancake-lee/pgo/pkg/util"
	"github.com/spf13/cobra"
)

var PrettyCode = &cobra.Command{
	Use:   "pretty",
	Short: "pretty code",
	Long:  "Standardize all dividers to 50 characters in length.",
	Run:   run,
}

func run(cmd *cobra.Command, args []string) {
	curDir := util.GetCurDir()

	fmt.Printf("Processing files in: %s\n", curDir)

	err := filepath.WalkDir(curDir, processFile)
	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
	}
}

var includeFileExtSet = map[string]bool{
	".go": true,
	".js": true,
	".ts": true,
}

// TODO 通过.gitignore 文件以及参数来排除目录和文件
var excludeDirs = []string{".git", ".vscode", "node_module", "bin", ".pb.go", "swagger"}

func isExcludeDirs(path string) bool {
	for _, excludeDir := range excludeDirs {
		if strings.Contains(path, excludeDir) {
			return true
		}
	}
	return false
}

func processFile(path string, d fs.DirEntry, err error) error {
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

	return formatDividersInFile(path)
}

func formatDividersInFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	modified := false

	// 匹配分割线的正则表达式
	// 匹配 // ---- 或 # ---- 或 /* ---- */ 等形式
	dividerRegex := regexp.MustCompile(`^(\s*)(//|#|\*|/\*)\s*-{2,}(\s*\*/)?(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		if dividerRegex.MatchString(line) {
			matches := dividerRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				indent := matches[1]       // 前置空格
				commentStart := matches[2] // 注释开始符号
				// 注释结束符号（如 */）已匹配但未使用，下面直接拼接/**/就好了

				// 生成新的分割线
				var newLine string
				if commentStart == "/*" {
					newLine = fmt.Sprintf("%s/* %s */", indent, strings.Repeat("-", 50))
				} else {
					newLine = fmt.Sprintf("%s%s %s", indent, commentStart, strings.Repeat("-", 50))
				}

				lines = append(lines, newLine)
				modified = true
			} else {
				lines = append(lines, line)
			}
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
