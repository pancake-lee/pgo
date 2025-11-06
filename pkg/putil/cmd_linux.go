//go:build linux

package putil

import (
	"errors"
	"os/exec"
)

func getBash() string {
	return `/bin/sh`
}
func execDefaultSetting(cmd *exec.Cmd) {
}

// Deprecated: 废弃，linux上应该用systemctl或pm2托管
func ExecBG(command string) error {
	return errors.New("not implemented")
}
