package devops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type DeployConfig struct {
	Files   map[string]string `json:"files"`
	Exclude []string          `json:"exclude"`
}

type DeployConflict struct {
	LocalPath  string
	RemotePath string
	LocalMD5   string
	RemoteMD5  string
}

func DeployCli() {
	cachePath := pconfig.GetDefaultCachePath()
	pthird.Interact.Infof("using cache file: %v", cachePath)

	sshHost := pclient.GetCachedParam(cachePath, "deploy.ssh.host", "SSH Host", "127.0.0.1")
	sshPort := pclient.GetCachedParam(cachePath, "deploy.ssh.port", "SSH Port", "22")
	sshUser := pclient.GetCachedParam(cachePath, "deploy.ssh.user", "SSH User", "root")
	sshPass := pclient.GetCachedParam(cachePath, "deploy.ssh.pass", "SSH Password", "")
	remoteRoot := pclient.GetCachedParam(cachePath, "deploy.ssh.dir", "Remote Root Dir", "/root/pgo")

	host := fmt.Sprintf("%s:%s", sshHost, sshPort)
	pthird.Interact.Infof("Connecting to %s@%s...", sshUser, host)
	sshCli, err := pthird.NewSSHClient(sshUser, sshPass, host)
	if err != nil {
		pthird.Interact.Errorf("SSH Connection failed: %v", err)
		return
	}

	err = sshCli.InitSftp()
	if err != nil {
		sshCli.Close()
		pthird.Interact.Errorf("SFTP Initialization failed: %v", err)
		return
	}

	// --------------------------------------------------
	sel := pthird.Interact.NewSelector("Deploy Menu")
	sel.Reg("First Time Deployment", func() {
		firstTimeDeploy(sshCli, remoteRoot)
	})
	sel.Reg("Update Service Process", func() {
		updateServiceProcess(sshCli, remoteRoot)
	})
	sel.Run()
}

func firstTimeDeploy(sshCli *pthird.SshClient, remoteRoot string) {
	dstRootPath := putil.NewPath(remoteRoot)
	var conflictList []DeployConflict

	cfg, err := loadDeployConfig()
	if err != nil {
		return
	}

	pthird.Interact.Infof("Starting deployment check...")

	for src, dst := range cfg.Files {
		if cfg.ShouldExclude(src) {
			pthird.Interact.Infof("Skipped by exclude rule: %s", src)
			continue
		}

		// Handle wildcard patterns
		srcPath := putil.NewPathS(src)
		if srcPath.HasGlob() {
			// Expand wildcard and deploy each matched file
			matches, err := srcPath.Glob()
			if err != nil {
				pthird.Interact.Warnf("Glob error for %s: %v", src, err)
				continue
			}
			if len(matches) == 0 {
				pthird.Interact.Warnf("No files matched pattern: %s", src)
				continue
			}

			// Calculate the base prefix from the wildcard pattern (directory part before the glob)
			globBase := srcPath.ToFile().GetPath()
			if srcPath.IsDir() {
				globBase = srcPath.GetPath()
			}

			for _, matchPath := range matches {
				matchInfo, err := os.Stat(matchPath)
				if err != nil {
					pthird.Interact.Warnf("Skipping %s: %v", matchPath, err)
					continue
				}
				if matchInfo.IsDir() {
					continue
				}

				// Calculate relative path from glob base
				relPath := putil.NewPathS(matchPath).CutPrefix(filepath.Dir(globBase)).ToFile().GetPath()
				fullDstPath := dstRootPath.Clone().Join(dst).Join(relPath)

				conflict, err := deployOneFile(sshCli, matchPath, fullDstPath.GetPath())
				if conflict != nil {
					conflictList = append(conflictList, *conflict)
				}
				if err != nil {
					pthird.Interact.Errorf("Failed to deploy file %s: %v", matchPath, err)
				}
			}
			continue
		}

		// Handle directory recursion or single file
		srcPath = putil.NewPathS(src)
		info, err := os.Stat(srcPath.GetPath())
		if err != nil {
			pthird.Interact.Warnf("Skipping %s: %v", srcPath.GetPath(), err)
			continue
		}

		if info.IsDir() {
			// Recursive copy
			err := filepath.Walk(srcPath.GetPath(),
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if cfg.ShouldExclude(path) {
						pthird.Interact.Infof("Skipped by exclude rule: %s", path)
						if info.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}

					if info.IsDir() {
						return nil
					}

					ph := putil.NewPathS(path).CutPrefix(srcPath.GetPath())
					dstPath := dstRootPath.Clone().Join(dst).Join(ph.GetPath())
					conflict, err := deployOneFile(sshCli, path, dstPath.GetPath())
					if conflict != nil {
						conflictList = append(conflictList, *conflict)
					}
					return err
				})
			if err != nil {
				pthird.Interact.Errorf("Failed to deploy directory %s: %v",
					srcPath.GetPath(), err)
				return
			}
		} else {
			// Single file
			if cfg.ShouldExclude(srcPath.GetPath()) {
				pthird.Interact.Infof("Skipped by exclude rule: %s", srcPath.GetPath())
				continue
			}

			dstPath := dstRootPath.Clone().Join(dst)
			conflict, err := deployOneFile(sshCli, srcPath.GetPath(), dstPath.GetPath())
			if conflict != nil {
				conflictList = append(conflictList, *conflict)
			}
			if err != nil {
				pthird.Interact.Errorf("Failed to deploy file %s: %v",
					srcPath.GetPath(), err)
				return
			}
		}
	}

	if len(conflictList) == 0 {
		pthird.Interact.Infof("All files deployed successfully.")
	} else {
		pthird.Interact.Warnf("Deployment completed with %d skipped files (remote exists and MD5 mismatch):", len(conflictList))
		for _, c := range conflictList {
			pthird.Interact.Warnf("SKIPPED: %s -> %s", c.LocalPath, c.RemotePath)
			pthird.Interact.Warnf("  Remote: %s", c.RemoteMD5)
			pthird.Interact.Warnf("  Local : %s", c.LocalMD5)
		}
	}

	// 5. Run Docker Compose
	autoRun := pthird.Interact.
		Input("Run 'docker-compose up -d' automatically? (y/n) [n]: ")

	cmdStr := fmt.Sprintf(`cd %s && `+
		`find . -name "*.sh" -exec chmod +x {} \; && `+
		`docker-compose up -d`,
		dstRootPath.GetPath())

	if strings.ToLower(autoRun) == "y" {
		pthird.Interact.Infof("Executing: %s", cmdStr)
		stdout, stderr, err := sshCli.RunCommand(cmdStr)
		if err != nil {
			pthird.Interact.Errorf("Command failed: %v\nStderr: %s", err, stderr)
		} else {
			pthird.Interact.Infof("Success:\n%s", stdout)
		}
	} else {
		pthird.Interact.Infof("Manual run command:\n%s", cmdStr)
	}
}

func loadDeployConfig() (*DeployConfig, error) {
	deployFile := putil.NewPathS("deploy/deploy.json").GetPath()
	if _, err := os.Stat(deployFile); os.IsNotExist(err) {
		pthird.Interact.Errorf("Deploy config not found: %s", deployFile)
		return nil, err
	}

	content, err := os.ReadFile(deployFile)
	if err != nil {
		pthird.Interact.Errorf("Failed to read deploy config: %v", err)
		return nil, err
	}
	var cfg DeployConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		pthird.Interact.Errorf("Failed to parse deploy config: %v", err)
		return nil, err
	}

	if cfg.Files == nil {
		cfg.Files = map[string]string{}
	}
	if cfg.Exclude == nil {
		cfg.Exclude = []string{}
	}

	return &cfg, nil
}

func (d *DeployConfig) ShouldExclude(srcPath string) bool {
	normSrcPath := putil.NewPath(srcPath).SetIsRel(true).GetPath()
	for _, rule := range d.Exclude {
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

func matchExcludeRule(srcPath, rule string) bool {
	hasGlob := strings.ContainsAny(rule, "*?[")
	if hasGlob {
		matched, err := filepath.Match(rule, srcPath)
		if err == nil && matched {
			return true
		}
	}

	prefix := strings.TrimSuffix(rule, "/")
	if srcPath == prefix {
		return true
	}
	if strings.HasPrefix(srcPath, prefix+"/") {
		return true
	}

	return false
}

func deployOneFile(sshCli *pthird.SshClient, localPath, remotePath string) (*DeployConflict, error) {
	pthird.Interact.Infof("Checking %s -> %s", localPath, remotePath)

	md5Cmd := fmt.Sprintf("md5sum '%s'", remotePath)
	stdout, _, err := sshCli.RunCommand(md5Cmd)

	remoteExists := false
	remoteMd5 := ""

	if err == nil {
		parts := strings.Fields(stdout)
		if len(parts) > 0 {
			remoteMd5 = parts[0]
			remoteExists = true
		}
	}

	if remoteExists {
		localMd5, err := putil.GetFileMd5(localPath)
		if err != nil {
			return nil, fmt.Errorf("local md5 fail: %w", err)
		}

		if localMd5 == remoteMd5 {
			pthird.Interact.Infof("  Skipped (MD5 Match)")
			return nil, nil
		}

		pthird.Interact.Warnf("  Skipped (Remote exists and MD5 mismatch)")
		return &DeployConflict{
			LocalPath:  localPath,
			RemotePath: remotePath,
			LocalMD5:   localMd5,
			RemoteMD5:  remoteMd5,
		}, nil
	}

	pthird.Interact.Infof("  Copying...")
	if err := sshCli.Scp(localPath, remotePath); err != nil {
		return nil, err
	}

	return nil, nil
}

// --------------------------------------------------
func updateServiceProcess(sshCli *pthird.SshClient, remoteRoot string) {
	dstRootPath := putil.NewPath(remoteRoot)

	cfg, err := loadDeployConfig()
	if err != nil {
		return
	}

	// Filter bin files
	var options []string
	// Collect keys
	for src := range cfg.Files {
		if strings.HasPrefix(src, "bin/") ||
			strings.HasPrefix(src, "./bin/") {
			options = append(options, src)
		}
	}
	sort.Strings(options)

	if len(options) == 0 {
		pthird.Interact.Warnf("No service process configurations found (starting with bin/)")
		return
	}
	var selectedSrc string
	sel := pthird.Interact.NewSelector("Select service to update")
	for _, opt := range options {
		o := opt
		sel.Reg(o, func() { selectedSrc = o })
	}
	sel.Run()

	if selectedSrc == "" {
		return
	}

	selectedDst := cfg.Files[selectedSrc]

	srcPath := putil.NewPathS(selectedSrc)
	dstPath := dstRootPath.Clone().Join(selectedDst)

	pthird.Interact.Infof("Updating %s -> %s",
		srcPath.GetPath(), dstPath.GetPath())

	// Check if local exists
	_, err = os.Stat(srcPath.GetPath())
	if os.IsNotExist(err) {
		pthird.Interact.Errorf("Local file not found: %s", srcPath.GetPath())
		return
	}

	// Force copy (overwrite) for update
	// 区别于deployOneFile需要检测MD5，防止自动部署覆盖手动修改
	// 更新服务，是主动覆盖文件，更新程序，修改文件后由pm2 watch自动重启服务

	// Fix: remove remote file first to avoid "Text file busy"
	sshCli.RunCommand(fmt.Sprintf("rm -f '%s'", dstPath.GetPath()))

	err = sshCli.Scp(srcPath.GetPath(), dstPath.GetPath())
	if err != nil {
		pthird.Interact.Errorf("Failed to copy file: %v", err)
		return
	}

	pthird.Interact.Infof("Update successful!")
}
