package devops

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pancake-lee/pgo/cmd/pgo/common"
	"github.com/pancake-lee/pgo/pkg/putil"
)

const (
	paramNameSrcRoot = "srcRoot"
	paramNameDstRoot = "dstRoot"
	cacheKeyPrefix   = "devops.initProj."
)

var initProjParamSettingList = []common.ParamItem{{
	Name:    paramNameSrcRoot,
	Usage:   "source project root",
	Default: ".",
}, {
	Name:    paramNameDstRoot,
	Usage:   "destination project root",
	Default: "../in3",
}}

var InitProjEntrypoint = common.NewToolEntrypoint(common.ToolEntrypointOption{
	ToolName:       "initProj",
	Use:            "initProj",
	Aliases:        []string{"init-proj"},
	Short:          "从当前仓库抽取新项目初始化骨架",
	CacheKeyPrefix: cacheKeyPrefix,
	ParamList:      initProjParamSettingList,
	Run:            RunInitProj,
})

type initProjRunOptions struct {
	SrcRoot string
	DstRoot string
}

type InitProjConfig struct {
	Files   map[string]string `json:"files"`
	Exclude []string          `json:"exclude"`
}

type initCopyTask struct {
	SrcPath string
	DstPath string
	SrcRel  string
	DstRel  string
}

func CICli() {
	sel := putil.Interact.NewSelector("Devops CI")
	sel.Reg("Make", MakeCli)
	sel.Reg("Init Project", InitProjCli)
	sel.Loop()
}

func InitProjCli() {
	InitProjEntrypoint.RunInteractive()
}

func RunInitProj(values common.ParamMap) error {
	opt := convParamToRunOpt(values)
	return runInitProject(opt)
}

func convParamToRunOpt(values common.ParamMap) initProjRunOptions {
	srcRoot := strings.TrimSpace(values[paramNameSrcRoot])
	if srcRoot == "" {
		srcRoot = "."
	}

	dstRoot := strings.TrimSpace(values[paramNameDstRoot])
	if dstRoot == "" {
		dstRoot = "../in3"
	}

	return initProjRunOptions{
		SrcRoot: srcRoot,
		DstRoot: dstRoot,
	}
}

// --------------------------------------------------
func runInitProject(opt initProjRunOptions) error {
	srcAbs, err := filepath.Abs(opt.SrcRoot)
	if err != nil {
		return fmt.Errorf("resolve srcRoot failed: %w", err)
	}

	dstAbs, err := filepath.Abs(opt.DstRoot)
	if err != nil {
		return fmt.Errorf("resolve dstRoot failed: %w", err)
	}

	if srcAbs == dstAbs {
		return fmt.Errorf("dstRoot can not equal srcRoot: %s", dstAbs)
	}

	if err = ensureSafeTargetPath(dstAbs); err != nil {
		return err
	}

	if err = os.MkdirAll(dstAbs, 0755); err != nil {
		return fmt.Errorf("create dstRoot failed: %w", err)
	}

	putil.Interact.Infof("init project from %s -> %s", srcAbs, dstAbs)
	moduleName, err := inferModuleNameFromPath(dstAbs)
	if err != nil {
		return err
	}

	configPath := filepath.Join(srcAbs, "deploy", "initProj.json")
	cfg, err := loadInitProjConfig(configPath)
	if err != nil {
		return err
	}

	taskList, err := buildCopyTaskList(srcAbs, dstAbs, cfg)
	if err != nil {
		return err
	}

	conflictList, err := collectConflictList(dstAbs, taskList)
	if err != nil {
		return err
	}
	if len(conflictList) > 0 {
		putil.Interact.Warnf("found conflict files in destination, please delete them manually:")
		for _, conflictPath := range conflictList {
			putil.Interact.Warnf("  %s", conflictPath)
		}
		return fmt.Errorf("conflict files detected: %d", len(conflictList))
	}

	for _, task := range taskList {
		err = copyFile(task.SrcPath, task.DstPath, moduleName)
		if err != nil {
			return fmt.Errorf("copy failed [%s -> %s]: %w", task.SrcRel, task.DstRel, err)
		}
	}

	err = initTargetGoModule(dstAbs, moduleName)
	if err != nil {
		return err
	}

	putil.Interact.Infof("init project done: %s", dstAbs)
	return nil
}

func ensureSafeTargetPath(dstAbs string) error {
	clean := filepath.Clean(dstAbs)
	if clean == "/" || clean == "." || clean == "" {
		return fmt.Errorf("unsafe dstRoot: %s", dstAbs)
	}
	return nil
}

func inferModuleNameFromPath(dstAbs string) (string, error) {
	moduleName := filepath.Base(filepath.Clean(dstAbs))
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" || moduleName == "." || moduleName == string(filepath.Separator) {
		return "", fmt.Errorf("invalid module name from dstRoot: %s", dstAbs)
	}
	return moduleName, nil
}

func loadInitProjConfig(configPath string) (*InitProjConfig, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read initProj config failed [%s]: %w", configPath, err)
	}

	var cfg InitProjConfig
	if err = json.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse initProj config failed [%s]: %w", configPath, err)
	}

	if cfg.Files == nil {
		cfg.Files = map[string]string{}
	}
	if cfg.Exclude == nil {
		cfg.Exclude = []string{}
	}

	if len(cfg.Files) == 0 {
		return nil, fmt.Errorf("initProj config files is empty [%s]", configPath)
	}

	return &cfg, nil
}

func (cfg *InitProjConfig) ShouldExclude(srcPath string) bool {
	normSrcPath := putil.NewPath(srcPath).SetIsRel(true).GetPath()
	for _, rule := range cfg.Exclude {
		normRule := putil.NewPath(rule).SetIsRel(true).GetPath()
		if normRule == "" {
			continue
		}
		if matchExcludeRule(normSrcPath, normRule) {
			return true
		}
	}
	return false
}

func buildCopyTaskList(srcRoot, dstRoot string, cfg *InitProjConfig) ([]initCopyTask, error) {
	taskList := make([]initCopyTask, 0)

	keyList := make([]string, 0, len(cfg.Files))
	for srcRel := range cfg.Files {
		keyList = append(keyList, srcRel)
	}
	sort.Strings(keyList)

	for _, srcRel := range keyList {
		dstRel := cfg.Files[srcRel]

		if cfg.ShouldExclude(srcRel) {
			putil.Interact.Infof("skip by exclude rule: %s", srcRel)
			continue
		}

		srcPath := filepath.Join(srcRoot, srcRel)
		dstPath := filepath.Join(dstRoot, dstRel)

		isDir := putil.NewPathS(srcRel).IsDir() || putil.NewPathS(dstRel).IsDir()
		if !isDir {
			srcInfo, err := os.Stat(srcPath)
			if err != nil {
				return nil, err
			}
			isDir = srcInfo.IsDir()
		}

		if !isDir {
			taskList = append(taskList, initCopyTask{
				SrcPath: srcPath,
				DstPath: dstPath,
				SrcRel:  srcRel,
				DstRel:  dstRel,
			})
			continue
		}

		err := filepath.WalkDir(srcPath, func(curSrcPath string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}

			relInDir, err := filepath.Rel(srcPath, curSrcPath)
			if err != nil {
				return err
			}
			if relInDir == "." {
				return nil
			}

			curSrcRel := putil.NewPathS(srcRel).Join(relInDir).GetPath()
			if cfg.ShouldExclude(curSrcRel) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if d.IsDir() {
				return nil
			}

			curDstPath := filepath.Join(dstPath, relInDir)
			curDstRel := putil.NewPathS(dstRel).Join(relInDir).GetPath()

			taskList = append(taskList, initCopyTask{
				SrcPath: curSrcPath,
				DstPath: curDstPath,
				SrcRel:  curSrcRel,
				DstRel:  curDstRel,
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return taskList, nil
}

func collectConflictList(dstRoot string, taskList []initCopyTask) ([]string, error) {
	conflictPathMap := make(map[string]struct{})
	seenDstPathMap := make(map[string]struct{})

	for _, task := range taskList {
		if _, ok := seenDstPathMap[task.DstPath]; ok {
			conflictPathMap[task.DstPath] = struct{}{}
			continue
		}
		seenDstPathMap[task.DstPath] = struct{}{}

		_, err := os.Stat(task.DstPath)
		if err == nil {
			conflictPathMap[task.DstPath] = struct{}{}
			continue
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat dst file failed [%s]: %w", task.DstPath, err)
		}
	}

	conflictList := make([]string, 0, len(conflictPathMap))
	for absPath := range conflictPathMap {
		relPath, err := filepath.Rel(dstRoot, absPath)
		if err != nil {
			relPath = absPath
		}
		conflictList = append(conflictList, relPath)
	}
	sort.Strings(conflictList)
	return conflictList, nil
}

func copyFile(srcPath, dstPath, moduleName string) error {
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("source is directory: %s", srcPath)
	}

	err = os.MkdirAll(filepath.Dir(dstPath), 0755)
	if err != nil {
		return err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dstPath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		srcInfo.Mode().Perm())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return rewriteCopiedFileImport(dstPath, moduleName)
}

func rewriteCopiedFileImport(filePath, moduleName string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(contentBytes)
	nextContent := content
	nextContent = strings.ReplaceAll(nextContent,
		"github.com/pancake-lee/pgo/internal",
		moduleName+"/internal")
	nextContent = strings.ReplaceAll(nextContent,
		"github.com/pancake-lee/pgo/api",
		moduleName+"/api")

	if nextContent == content {
		return nil
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(nextContent), info.Mode().Perm())
}

func initTargetGoModule(dstRoot, moduleName string) error {
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Dir = dstRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod init failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = dstRoot
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	putil.Interact.Infof("go module initialized: %s", moduleName)
	return nil
}
