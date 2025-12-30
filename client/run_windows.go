//go:build windows

package main

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
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
	w.Resize(fyne.NewSize(1000, 700))

	// Output area (Right)
	outputData := binding.NewString()
	output := widget.NewEntryWithData(outputData)
	output.MultiLine = true
	output.SetPlaceHolder("Output will appear here...")
	output.Disable() // Read-only

	// Center area (Function Content)
	centerContent := container.NewStack()
	centerContent.Add(widget.NewLabel("Please select a function from the left."))

	centerSpacer := canvas.NewRectangle(color.Transparent)
	centerSpacer.SetMinSize(fyne.NewSize(400, 0))
	centerFixed := container.NewStack(centerSpacer, centerContent)

	// Left area (Function List)
	btnCourseSwap := widget.NewButton("调课 (Course Swap)", func() {
		ui := makeCourseSwapUI(w, output)
		centerContent.Objects = []fyne.CanvasObject{ui}
		centerContent.Refresh()
	})

	leftMenu := container.NewVBox(
		widget.NewLabel("Functions"),
		btnCourseSwap,
	)

	leftSpacer := canvas.NewRectangle(color.Transparent)
	leftSpacer.SetMinSize(fyne.NewSize(100, 0))
	leftFixed := container.NewStack(leftSpacer, leftMenu)

	// Layout: Left(100) | Center(400) | Right(Rest)
	// Inner: Center(Left) | Right(Center)
	innerBorder := container.NewBorder(nil, nil, centerFixed, nil, output)
	// Outer: Left(Left) | Inner(Center)
	rootBorder := container.NewBorder(nil, nil, leftFixed, nil, innerBorder)

	w.SetContent(rootBorder)
	w.ShowAndRun()
}

func makeCourseSwapUI(w fyne.Window, output *widget.Entry) fyne.CanvasObject {
	// Load cache to pre-fill
	cache := courseSwap.LoadCache()

	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("Excel Path")
	pathEntry.SetText(cache.Path)

	// Teacher Select
	teacherSelect := widget.NewSelectEntry([]string{})
	teacherSelect.PlaceHolder = "Select or Type Teacher"
	teacherSelect.SetText(cache.Teacher)

	// Helper to update teacher list
	updateTeachers := func(path string) {
		if _, err := os.Stat(path); err == nil {
			// File exists, load teachers in background
			go func() {
				teachers, err := courseSwap.GetTeacherList(path)
				if err == nil {
					teacherSelect.SetOptions(teachers)
				}
			}()
		}
	}

	// Initial update
	if cache.Path != "" {
		updateTeachers(cache.Path)
	}

	var debounceTimer *time.Timer
	pathEntry.OnChanged = func(s string) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
			updateTeachers(s)
		})
	}

	// Hardcoded setting: use Windows native dialog by default (false means use native)
	useFyneFileDialog := false

	browseBtn := widget.NewButton("...", func() {
		if !useFyneFileDialog {
			file, err := openNativeFileDialog(pathEntry.Text)
			if err == nil && file != "" {
				pathEntry.SetText(file)
				// OnChanged will trigger updateTeachers
			}
		} else {
			dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err == nil && reader != nil {
					pathEntry.SetText(reader.URI().Path())
				}
			}, w)
		}
	})

	pathContainer := container.NewBorder(nil, nil, nil, browseBtn, pathEntry)

	// Date Picker
	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("Date (YYYYMMDD)")
	dateEntry.SetText(cache.Date)
	dateEntry.Disable() // Read-only

	dateBtn := widget.NewButton("Select Date", func() {
		openDatePicker(w, dateEntry.Text, func(s string) {
			dateEntry.SetText(s)
		})
	})
	dateContainer := container.NewBorder(nil, nil, nil, dateBtn, dateEntry)

	// Course Num Select
	courseNumSelect := widget.NewSelect([]string{"1", "2", "3", "4", "5", "6", "7"}, nil)
	if cache.CourseNum > 0 {
		courseNumSelect.SetSelected(fmt.Sprintf("%d", cache.CourseNum))
	} else {
		courseNumSelect.SetSelected("1")
	}

	// storageTypeSelect := widget.NewSelect([]string{"Local", "Cloud"}, nil)
	// if cache.StorageType != "" {
	// 	storageTypeSelect.SetSelected(cache.StorageType)
	// } else {
	// 	storageTypeSelect.SetSelected("Local")
	// }

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "课表", Widget: pathContainer},
			{Text: "教师", Widget: teacherSelect},
			{Text: "日期", Widget: dateContainer},
			{Text: "节次", Widget: courseNumSelect},
			// {Text: "存储", Widget: storageTypeSelect},
		},
		OnSubmit: func() {
			// Construct config
			cNum, _ := strconv.Atoi(courseNumSelect.Selected)

			config := courseSwap.InputConfig{
				Path:        pathEntry.Text,
				Teacher:     teacherSelect.Text,
				Date:        dateEntry.Text,
				CourseNum:   cNum,
				StorageType: "Local",
				// StorageType: storageTypeSelect.Selected,
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

	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) > 0 {
			pathEntry.SetText(uris[0].Path())
		}
	})

	return form
}

func openNativeFileDialog(initialPath string) (string, error) {
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
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func openDatePicker(w fyne.Window, current string, onSet func(string)) {
	now := time.Now()
	if current != "" {
		if t, err := time.Parse("20060102", current); err == nil {
			now = t
		}
	}

	years := make([]string, 10)
	for i := 0; i < 10; i++ {
		years[i] = fmt.Sprintf("%d", now.Year()-2+i)
	}
	yearSel := widget.NewSelect(years, nil)
	yearSel.SetSelected(fmt.Sprintf("%d", now.Year()))

	months := make([]string, 12)
	for i := 1; i <= 12; i++ {
		months[i-1] = fmt.Sprintf("%02d", i)
	}
	monthSel := widget.NewSelect(months, nil)
	monthSel.SetSelected(fmt.Sprintf("%02d", now.Month()))

	days := make([]string, 31)
	for i := 1; i <= 31; i++ {
		days[i-1] = fmt.Sprintf("%02d", i)
	}
	daySel := widget.NewSelect(days, nil)
	daySel.SetSelected(fmt.Sprintf("%02d", now.Day()))

	content := container.NewHBox(yearSel, widget.NewLabel("Y"), monthSel, widget.NewLabel("M"), daySel, widget.NewLabel("D"))

	dialog.ShowCustomConfirm("Select Date", "OK", "Cancel", content, func(ok bool) {
		if ok {
			onSet(fmt.Sprintf("%s%s%s", yearSel.Selected, monthSel.Selected, daySel.Selected))
		}
	}, w)
}
