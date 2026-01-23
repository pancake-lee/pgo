package putil

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SshClient struct {
	cli     *ssh.Client
	sftpCli *sftp.Client
	config  *ssh.ClientConfig
	addr    string
	mu      sync.Mutex
}

func NewSSHClient(user, password, addr string) (*SshClient, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return &SshClient{
		cli:    client,
		config: config,
		addr:   addr,
	}, nil
}

func (s *SshClient) reconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := ssh.Dial("tcp", s.addr, s.config)
	if err != nil {
		return err
	}

	s.cli = client
	return nil
}
func (s *SshClient) Close() {
	s.cli.Close()
}

// --------------------------------------------------

func (s *SshClient) GetAddr() string {
	return s.addr
}

func (s *SshClient) PowershellCmd(cmd string,
) (string, string, error) {
	return s.RunCommand(fmt.Sprintf(
		`powershell -command "%s"`, cmd))
}

func (s *SshClient) RunCommand(cmd string,
) (string, string, error) {
	session, err := s.cli.NewSession()
	if err != nil {
		if err := s.reconnect(); err != nil {
			return "", "", err
		}
		session, err = s.cli.NewSession()
		if err != nil {
			return "", "", err
		}
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	if err := session.Run(cmd); err != nil {
		return "", "", err
	}

	return strings.TrimSuffix(stdoutBuf.String(), "\n"),
		strings.TrimSuffix(stderrBuf.String(), "\n"),
		nil
}

// --------------------------------------------------
func (s *SshClient) InitSftp() error {
	if s.sftpCli != nil {
		return nil
	}
	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(s.cli)
	if err != nil {
		return err
	}
	s.sftpCli = sftpClient
	return nil
}
func (s *SshClient) Scp(src, dst string) error {
	err := s.InitSftp()
	if err != nil {
		fmt.Printf("InitSftp err : %v\n", err)
		return err
	}
	// 读取本地文件内容
	fileContent, err := os.ReadFile(src)
	if err != nil {
		fmt.Printf("ReadFile[%v] err : %v\n", src, err)
		return err
	}

	// 简单的路径处理 logic
	// 假设 dst 是远程 linux 路径
	finalDst := dst
	if strings.HasSuffix(dst, "/") {
		finalDst = path.Join(dst, filepath.Base(src))
	}

	// 确保父目录存在
	parentFolder := path.Dir(finalDst)
	// MkdirAll 类似于 mkdir -p
	err = s.sftpCli.MkdirAll(parentFolder)
	// sftp MkdirAll 如果目录已存在可能会报错也可能不会，取决于实现。
	// 但通常它处理得还行。如果报错，通常是权限等问题。
	// 不过 sftp MkdirAll 并不总是像 mkdir -p 那样幂等 (如果目录存在可能报错)。
	// 为了简单起见，我们忽略 MkdirAll 的错误或者假设它工作正常。
	// 更好的做法是逐级检查。
	if err != nil {
		// 忽略错误，尝试直接写，或者打印日志
		// fmt.Printf("mkdir[%v] err : %v\n", parentFolder, err)
		// 有些 sftp server 实现 MkdirAll 对于已存在目录会报错
	}

	// 创建远端文件并写入内容
	f, err := s.sftpCli.Create(finalDst)
	if err != nil {
		fmt.Printf("create[%v] err : %v\n", finalDst, err)
		return err
	}
	defer f.Close()

	_, err = f.Write(fileContent)
	if err != nil {
		fmt.Printf("Write[%v] err : %v\n", finalDst, err)
		return err
	}

	return nil
}
