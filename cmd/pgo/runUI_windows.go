//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"fyne.io/fyne/v2"
	"github.com/pancake-lee/pgo/cmd/pgo/courseSwap"
	"github.com/pancake-lee/pgo/cmd/pgo/swagger"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func runUI() {
	app := pclient.NewApp("PGO Client")

	// Register course swap page
	app.RegisterPage("调课", buildCourseSwapPage)

	// Show window and run
	app.ShowAndRun()
}

// buildCourseSwapPage builds the course swap center panel content
func buildCourseSwapPage(app *pclient.App) fyne.CanvasObject {
	outputPanel := app.OutputPanel()

	// Create course swap UI builder
	builder := courseSwap.NewCourseSwapUIBuilder(
		app.Window(),
		outputPanel.LogData(),
		nil, // resultData no longer managed externally
	)

	// Set callbacks
	builder.SetCallbacks(
		// onCalculate - handle form submission
		func(config courseSwap.InputConfig) {
			outputPanel.ClearResults()
			go func() {
				mgr, err := courseSwap.CalculateSwapCandidates(config)
				if err != nil {
					plogger.LogErr(err)
					return
				}

				courses := mgr.GetCourses()
				if len(courses) == 0 {
					plogger.Errorf("找不到合适的调课候选")
					return
				}

				plogger.Infof("找到了合适的调课候选[%v]个，展示列表中...", len(courses))

				var items []pclient.ResultItem
				for _, c := range courses {
					course := c
					items = append(items, pclient.ResultItem{
						Display: fmt.Sprintf("%v", course),
						OnConfirm: func() {
							cache := courseSwap.LoadCache()
							err := courseSwap.ExecuteSwap(cache, course)
							if err != nil {
								plogger.LogErr(err)
							} else {
								plogger.Infof("调课成功: %v", course)
								outputPanel.ClearResults()
							}
						},
					})
				}
				outputPanel.SetResults(items)
			}()
		},
		// onExecute - handle result item click
		func(target *courseSwap.CourseInfo) {
			// Execution is handled via result list confirm button
		},
		// onDelete - handle history item delete
		func(id int32) {
			err := courseSwap.DeleteSwapHistory("Local", id)
			if err != nil {
				plogger.LogErr(err)
			}
		},
		// getHistory - get swap history
		func() ([]swagger.ApiCourseSwapRequestInfo, error) {
			return courseSwap.GetSwapHistory("Local")
		},
		// fileDialogFn - native file dialog
		openNativeFileDialog,
	)

	return builder.Build()
}

// openNativeFileDialog opens Windows native file dialog
func openNativeFileDialog(initialPath string) string {
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

	cmd := exec.Command("powershell", "-NoProfile", "-Command", cmdStr)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
