//go:build windows

package courseSwap

import (
	"fmt"

	"fyne.io/fyne/v2"
	"github.com/pancake-lee/pgo/cmd/pgo/swagger"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

// BuildPage builds the course swap center panel content for RegisterPage
func BuildPage(app *pclient.App) fyne.CanvasObject {
	outputPanel := app.OutputPanel()

	builder := NewCourseSwapUIBuilder(
		app.Window(),
		outputPanel.LogData(),
		nil,
	)

	builder.SetCallbacks(
		// onCalculate
		func(config InputConfig) {
			outputPanel.ClearResults()
			go func() {
				mgr, err := CalculateSwapCandidates(config)
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
							cache := LoadCache()
							err := ExecuteSwap(cache, course)
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
		// onExecute
		func(target *CourseInfo) {},
		// onDelete
		func(id int32) {
			err := DeleteSwapHistory("Local", id)
			if err != nil {
				plogger.LogErr(err)
			}
		},
		// getHistory
		func() ([]swagger.ApiCourseSwapRequestInfo, error) {
			return GetSwapHistory("Local")
		},
		// fileDialogFn
		pclient.OpenNativeFileDialog,
	)

	return builder.Build()
}
