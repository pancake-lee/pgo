//go:build windows

package pclient

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// OpenNativeFileDialog opens Windows native file dialog
func OpenNativeFileDialog(initialPath string) string {
	var dir string
	if initialPath != "" {
		if fi, err := os.Stat(initialPath); err == nil {
			if fi.IsDir() {
				dir = initialPath
			} else {
				dir = filepath.Dir(initialPath)
			}
		} else {
			dir = filepath.Dir(initialPath)
		}
	}

	psDir := ""
	if dir != "" {
		psDir = strings.ReplaceAll(dir, "'", "''")
	}

	cmdStr := fmt.Sprintf("& { [System.Reflection.Assembly]::LoadWithPartialName('System.windows.forms') | Out-Null; $OpenFileDialog = New-Object System.Windows.Forms.OpenFileDialog; $InitDir = '%s'; if ($InitDir -and (Test-Path $InitDir)) { $OpenFileDialog.InitialDirectory = $InitDir }; $OpenFileDialog.ShowDialog() | Out-Null; $OpenFileDialog.FileName }", psDir)

	out, err := putil.Exec("powershell", "-NoProfile", "-Command", cmdStr)
	if err != nil {
		plogger.LogErr(err)
		return ""
	}
	return out
}
