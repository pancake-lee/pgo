//go:build windows

package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/pancake-lee/pgo/client/courseSwap"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func runApp() {
	// Check if "cli" command is present in args
	// Note: os.Args[0] is the executable path
	isCli := false
	for _, arg := range os.Args[1:] {
		if arg == "cli" {
			isCli = true
			break
		}
	}

	if isCli {
		if err := rootCmd.Execute(); err != nil {
			plogger.Error(err)
			os.Exit(1)
		}
	} else {
		// Run UI directly without Cobra
		runUI()
	}
}

func runUI() {
	a := app.New()
	w := a.NewWindow("PGO Client")

	// Function selection
	label := widget.NewLabel("请选择功能:")

	// Output area
	outputData := binding.NewString()
	output := widget.NewEntryWithData(outputData)
	output.MultiLine = true
	output.SetPlaceHolder("Output will appear here...")
	output.Disable() // Read-only

	btnCourseSwap := widget.NewButton("调课 (Course Swap)", func() {
		output.SetText("Starting Course Swap...\n")
		showCourseSwapForm(w, output)
	})

	content := container.NewVBox(
		label,
		btnCourseSwap,
		widget.NewLabel("Output:"),
		container.NewGridWrap(fyne.NewSize(600, 400), output),
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}

func showCourseSwapForm(parent fyne.Window, output *widget.Entry) {
	// Load cache to pre-fill
	cache := courseSwap.LoadCache()

	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("Excel Path")
	pathEntry.SetText(cache.Path)

	teacherEntry := widget.NewEntry()
	teacherEntry.SetPlaceHolder("Teacher Name")
	teacherEntry.SetText(cache.Teacher)

	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("Date (YYYYMMDD)")
	dateEntry.SetText(cache.Date)

	courseNumEntry := widget.NewEntry()
	courseNumEntry.SetPlaceHolder("Course Num (1-7)")
	if cache.CourseNum > 0 {
		courseNumEntry.SetText(fmt.Sprintf("%d", cache.CourseNum))
	}

	storageTypeSelect := widget.NewSelect([]string{"Local", "Cloud"}, nil)
	if cache.StorageType != "" {
		storageTypeSelect.SetSelected(cache.StorageType)
	} else {
		storageTypeSelect.SetSelected("Local")
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Excel Path", Widget: pathEntry},
			{Text: "Teacher", Widget: teacherEntry},
			{Text: "Date", Widget: dateEntry},
			{Text: "Course Num", Widget: courseNumEntry},
			{Text: "Storage Type", Widget: storageTypeSelect},
		},
		OnSubmit: func() {
			// Construct config
			// We need to convert string to int for CourseNum
			var cNum int
			fmt.Sscanf(courseNumEntry.Text, "%d", &cNum)

			config := courseSwap.InputConfig{
				Path:        pathEntry.Text,
				Teacher:     teacherEntry.Text,
				Date:        dateEntry.Text,
				CourseNum:   cNum,
				StorageType: storageTypeSelect.Selected,
			}

			// Save cache
			courseSwap.SaveCache(config)

			output.SetText("Running Course Swap...\n")

			// Run in goroutine to not block UI
			go func() {
				courseSwap.CourseSwap(config)
				plogger.Infof("\nCourse Swap Finished.")
			}()
		},
	}

	w := fyne.CurrentApp().NewWindow("Course Swap Input")
	w.SetContent(form)
	w.Resize(fyne.NewSize(400, 300))
	w.Show()
}
