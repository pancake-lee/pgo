package devops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type DeployConfig struct {
	Files map[string]string `json:"files"`
}

func DeployCli() {
	deployFile := putil.NewPathS("deploy/deploy.json").GetPath()
	if _, err := os.Stat(deployFile); os.IsNotExist(err) {
		putil.Interact.Errorf("Deploy config not found: %s", deployFile)
		return
	}

	// 1. Load Deploy Config
	content, err := os.ReadFile(deployFile)
	if err != nil {
		putil.Interact.Errorf("Failed to read deploy config: %v", err)
		return
	}
	var cfg DeployConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		putil.Interact.Errorf("Failed to parse deploy config: %v", err)
		return
	}

	// 2. Interactive Inputs & Cache
	cachePath := pconfig.GetDefaultCachePath()

	sshHost := getCachedInput(cachePath, "deploy.ssh.host", "SSH Host", "127.0.0.1")
	sshPort := getCachedInput(cachePath, "deploy.ssh.port", "SSH Port", "22")
	sshUser := getCachedInput(cachePath, "deploy.ssh.user", "SSH User", "root")
	sshPass := getCachedInput(cachePath, "deploy.ssh.pass", "SSH Password", "")
	remoteRoot := getCachedInput(cachePath, "deploy.ssh.root", "Remote Root Dir", "/root/pgo")
	dstRootPath := putil.NewPath(remoteRoot)

	// 3. Connect SSH
	host := fmt.Sprintf("%s:%s", sshHost, sshPort)
	putil.Interact.Infof("Connecting to %s@%s...", sshUser, host)
	sshCli, err := putil.NewSSHClient(sshUser, sshPass, host)
	if err != nil {
		putil.Interact.Errorf("SSH Connection failed: %v", err)
		return
	}
	defer sshCli.Close()

	// Initialize SFTP
	if err := sshCli.InitSftp(); err != nil {
		putil.Interact.Errorf("SFTP Initialization failed: %v", err)
		return
	}

	// 4. Process Files
	putil.Interact.Infof("Starting deployment check...")

	for src, dst := range cfg.Files {
		// Handle directory recursion or single file
		srcPath := putil.NewPathS(src)
		info, err := os.Stat(srcPath.GetPath())
		if err != nil {
			putil.Interact.Warnf("Skipping %s: %v", srcPath, err)
			continue // Or error out?
		}

		if info.IsDir() {
			// Recursive copy
			err := filepath.Walk(srcPath.GetPath(),
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() {
						return nil
					}

					ph := putil.NewPathS(path).CutPrefix(srcPath.GetPath())
					dstPath := dstRootPath.Clone().Join(dst).Join(ph.GetPath())
					return deployOneFile(sshCli, path, dstPath.GetPath())
				})
			if err != nil {
				putil.Interact.Errorf("Failed to deploy directory %s: %v",
					srcPath.GetPath(), err)
				return
			}
		} else {
			// Single file
			dstPath := dstRootPath.Clone().Join(dst)
			err := deployOneFile(sshCli, srcPath.GetPath(), dstPath.GetPath())
			if err != nil {
				putil.Interact.Errorf("Failed to deploy file %s: %v",
					srcPath.GetPath(), err)
				return
			}
		}
	}

	putil.Interact.Infof("All files deployed successfully.")

	// 5. Run Docker Compose
	autoRun := putil.Interact.
		Input("Run 'docker-compose up -d' automatically? (y/n) [n]: ")

	cmdStr := fmt.Sprintf(`cd %s && `+
		`find . -name "*.sh" -exec chmod +x {} \; && `+
		`docker-compose up -d`,
		dstRootPath.GetPath())

	if strings.ToLower(autoRun) == "y" {
		putil.Interact.Infof("Executing: %s", cmdStr)
		stdout, stderr, err := sshCli.RunCommand(cmdStr)
		if err != nil {
			putil.Interact.Errorf("Command failed: %v\nStderr: %s", err, stderr)
		} else {
			putil.Interact.Infof("Success:\n%s", stdout)
		}
	} else {
		putil.Interact.Infof("Manual run command:\n%s", cmdStr)
	}
}

func getCachedInput(cachePath, key, prompt, layout string) string {
	cachedVal := pconfig.GetCacheValue(cachePath, key)
	defaultVal := layout
	if cachedVal != "" {
		defaultVal = cachedVal
	}

	inputPrompt := fmt.Sprintf("%s (Default: %s)", prompt, defaultVal)
	val := putil.Interact.Input(inputPrompt)

	if val == "" {
		val = defaultVal
	}

	if val != cachedVal {
		pconfig.SetCacheValue(cachePath, key, val)
	}
	return val
}

func deployOneFile(sshCli *putil.SshClient, localPath, remotePath string) error {
	putil.Interact.Infof("Checking %s -> %s", localPath, remotePath)

	// Check if remote file exists and MD5
	// cmd: md5sum <file> | awk '{print $1}'
	// Note: md5sum might output "hash  filename".

	md5Cmd := fmt.Sprintf("md5sum '%s'", remotePath)
	stdout, _, err := sshCli.RunCommand(md5Cmd)

	remoteExists := false
	remoteMd5 := ""

	if err == nil {
		// Parse MD5
		parts := strings.Fields(stdout)
		if len(parts) > 0 {
			remoteMd5 = parts[0]
			remoteExists = true
		}
	}

	if remoteExists {
		localMd5, err := putil.GetFileMd5(localPath)
		if err != nil {
			return fmt.Errorf("local md5 fail: %w", err)
		}

		if localMd5 == remoteMd5 {
			putil.Interact.Infof("  Skipped (MD5 Match)")
			return nil
		}

		// MD5 Mismatch - Prompt Conflict
		putil.Interact.Warnf("CONFLICT: MD5 Mismatch for %s", remotePath)
		putil.Interact.Warnf("Remote: %s", remoteMd5)
		putil.Interact.Warnf("Local : %s", localMd5)
		putil.Interact.Errorf("Interrupting deployment as per configuration.")
		return fmt.Errorf("conflict detected for %s", remotePath)
	}

	// Copy file
	putil.Interact.Infof("  Copying...")
	if err := sshCli.Scp(localPath, remotePath); err != nil {
		return err
	}

	return nil
}
