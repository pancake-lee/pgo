//go:build windows

package main

import (
	"bufio"
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
	// windows下通过 client.exe cli 参数启动命令行模式
	isCli := false
	for _, arg := range os.Args[1:] {
		if arg == "cli" {
			isCli = true
			break
		}
	}

	if isCli {
		runCli()
	} else {
		runUI()
	}
}

func runUI() {
	a := app.New()
	w := a.NewWindow("PGO Client")
	w.Resize(fyne.NewSize(1000, 700))

	// --------------------------------------------------
	// 右上，日志区域
	logData := binding.NewString()
	captureOutput(logData)
	logEntry := newReadOnlyEntry()
	// Manual binding to avoid validation icon
	logData.AddListener(binding.NewDataListener(func() {
		val, _ := logData.Get()
		logEntry.SetText(val)
	}))
	logEntry.MultiLine = true

	logEntry.SetPlaceHolder("Logs will appear here...")
	// logEntry.Disable() // Keep enabled for normal text color
	logEntry.SetMinRowsVisible(10)

	// --------------------------------------------------
	// 右下，调课结果列表
	// Shared state for the list
	var currentConfig courseSwap.InputConfig

	// List for swap candidates
	courseData := binding.NewUntypedList()
	resultList := widget.NewListWithData(
		courseData,
		func() fyne.CanvasObject {
			return container.NewBorder(
				nil, nil, nil, widget.NewButton("确认", nil),
				widget.NewLabel("placeholder"))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			val, _ := i.(binding.Untyped).Get()
			course, ok := val.(*courseSwap.CourseInfo)
			if !ok {
				return
			}

			c := o.(*fyne.Container)
			var label *widget.Label
			var btn *widget.Button
			for _, obj := range c.Objects {
				if l, ok := obj.(*widget.Label); ok {
					label = l
				} else if b, ok := obj.(*widget.Button); ok {
					btn = b
				}
			}

			label.SetText(fmt.Sprintf("%v", course))

			btn.SetText("确认")
			btn.Enable()
			btn.OnTapped = func() {
				dialog.ShowConfirm("确认调课", "确定要与该课程调课吗？",
					func(ok bool) {
						if !ok {
							return
						}

						err := courseSwap.ExecuteSwap(currentConfig, course)
						if err != nil {
							dialog.ShowError(err, w)
							return
						}
						dialog.ShowInformation("成功", "调课请求已保存。", w)
						btn.SetText("已选")
						btn.Disable()
					}, w)
			}
		},
	)

	// Right side layout: Log at top, List fills rest
	// Add separator between Log and List
	var seqWidth float32 = 3
	sep1 := canvas.NewRectangle(color.Gray{Y: 128})
	sep1.SetMinSize(fyne.NewSize(0, seqWidth))
	rightTop := container.NewVBox(widget.NewLabel("日志输出"), logEntry, sep1)
	rightBottom := container.NewVBox(widget.NewLabel("结果输出"), resultList)
	rightContent := container.NewBorder(rightTop, nil, nil, nil, rightBottom)

	// --------------------------------------------------
	// 中间，子功能参数操作区域
	centerParam := container.NewStack()
	centerParam.Add(widget.NewLabel("请先从左侧菜单选择一个功能"))

	centerContent := container.NewVBox(
		widget.NewLabel("参数输入"), centerParam)

	centerSpacer := canvas.NewRectangle(color.Transparent)
	centerSpacer.SetMinSize(fyne.NewSize(400, 0))
	centerFixed := container.NewStack(centerSpacer, centerContent)

	// Add separator to the right of center
	sep2 := canvas.NewRectangle(color.Gray{Y: 128})
	sep2.SetMinSize(fyne.NewSize(seqWidth, 0))
	centerWithSep := container.NewBorder(nil, nil, nil, sep2, centerFixed)

	// --------------------------------------------------
	// 左侧，功能菜单区域
	btnCourseSwap := widget.NewButton("调课", func() {
		ui := makeCourseSwapUI(w, logData, courseData, &currentConfig)
		centerParam.Objects = []fyne.CanvasObject{ui}
		centerParam.Refresh()
	})

	leftMenu := container.NewVBox(
		widget.NewLabel("功能列表"),
		btnCourseSwap,
	)

	leftSpacer := canvas.NewRectangle(color.Transparent)
	leftSpacer.SetMinSize(fyne.NewSize(100, 0))
	leftFixed := container.NewStack(leftSpacer, leftMenu)
	// Add separator to the right of left
	sep3 := canvas.NewRectangle(color.Gray{Y: 128})
	sep3.SetMinSize(fyne.NewSize(seqWidth, 0))
	leftWithSep := container.NewBorder(nil, nil, nil, sep3, leftFixed)

	// --------------------------------------------------
	// Layout: Left(100) | Center(400) | Right(Rest)
	innerBorder := container.NewBorder(nil, nil, centerWithSep, nil, rightContent)
	rootBorder := container.NewBorder(nil, nil, leftWithSep, nil, innerBorder)

	w.SetContent(rootBorder)
	w.ShowAndRun()
}

func makeCourseSwapUI(w fyne.Window,
	logData binding.String,
	courseData binding.UntypedList,
	currentConfig *courseSwap.InputConfig,
) fyne.CanvasObject {
	// Load cache to pre-fill
	cache := courseSwap.LoadCache()

	// --------------------------------------------------
	// Teacher Select
	teacherSelect := widget.NewSelectEntry([]string{})
	teacherSelect.SetText(cache.Teacher)

	updateTeachers := func(path string) {
		if _, err := os.Stat(path); err != nil {
			plogger.LogErr(err)
			return
		}
		// File exists, load teachers in background
		go func() {
			teachers, err := courseSwap.GetTeacherList(path)
			if err != nil {
				plogger.LogErr(err)
				return
			}
			teacherSelect.SetOptions(teachers)
		}()
	}

	// Initial update
	if cache.Path != "" {
		updateTeachers(cache.Path)
	}

	// --------------------------------------------------
	pathEntry := widget.NewEntry() // 键盘直接输入
	pathEntry.SetText(cache.Path)

	// 防抖动机制
	var debounceTimer *time.Timer
	pathEntry.OnChanged = func(s string) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
			updateTeachers(s)
		})
	}

	// 使用fyne文件选择器，还是windows原生的
	useFyneFileDialog := false

	// 文件选择器输入
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

	// 拖拽输入
	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) > 0 {
			pathEntry.SetText(uris[0].Path())
		}
	})

	pathContainer := container.NewBorder(nil, nil, nil, browseBtn, pathEntry)

	// --------------------------------------------------
	// Date Picker
	dateEntry := widget.NewEntry()
	dateEntry.SetText(cache.Date)

	dateBtn := widget.NewButton("选择日期", func() {
		openDatePicker(w, dateEntry.Text, func(s string) {
			dateEntry.SetText(s)
		})
	})
	dateContainer := container.NewBorder(nil, nil, nil, dateBtn, dateEntry)

	// --------------------------------------------------
	// Course Num Select
	courseNumSelect := widget.NewSelect(
		[]string{"1", "2", "3", "4", "5", "6", "7"}, nil)
	if cache.CourseNum > 0 && cache.CourseNum <= 7 {
		courseNumSelect.SetSelected(fmt.Sprintf("%d", cache.CourseNum))
	} else {
		courseNumSelect.SetSelected("1")
	}

	// --------------------------------------------------
	// 用Form做布局

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "课表", Widget: pathContainer},
			{Text: "教师", Widget: teacherSelect},
			{Text: "日期", Widget: dateContainer},
			{Text: "节次", Widget: courseNumSelect},
		},
		OnSubmit: func() {
			cNum, _ := strconv.Atoi(courseNumSelect.Selected)
			config := courseSwap.InputConfig{
				Path:        pathEntry.Text,
				Teacher:     teacherSelect.Text,
				Date:        dateEntry.Text,
				CourseNum:   cNum,
				StorageType: "Local",
			}
			courseSwap.SaveCache(config)
			*currentConfig = config

			courseData.Set(nil) // 清空输出列表

			go func() { // Run in goroutine to not block UI
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

				// Update UI with candidates
				var items []interface{}
				for _, c := range courses {
					items = append(items, c)
				}
				courseData.Set(items)
			}() // goroutine
		}, // OnSubmit
	} // form

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

	content := container.NewHBox(
		yearSel, widget.NewLabel("年"),
		monthSel, widget.NewLabel("月"),
		daySel, widget.NewLabel("日"))

	dialog.ShowCustomConfirm("选择日期", "确定", "取消",
		content, func(ok bool) {
			if ok {
				onSet(fmt.Sprintf("%s%s%s", yearSel.Selected, monthSel.Selected, daySel.Selected))
			}
		}, w)
}

func captureOutput(logData binding.String) {
	r, w, err := os.Pipe()
	if err != nil {
		return
	}

	os.Stdout = w
	os.Stderr = w

	// Re-init logger to pick up new stdout
	plogger.SetJsonLog(false)
	plogger.InitConsoleLogger()

	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			current, _ := logData.Get()
			if len(current) > 10000 {
				current = current[len(current)-5000:]
			}
			logData.Set(current + text + "\n")
		}
	}()
}

// readOnlyEntry is a wrapper around widget.Entry that prevents editing
// but keeps the text color normal (unlike Disable()) and allows selection/copy.
type readOnlyEntry struct {
	widget.Entry
}

func newReadOnlyEntry() *readOnlyEntry {
	entry := &readOnlyEntry{}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *readOnlyEntry) TypedRune(r rune) {
	// Ignore typing
}

func (e *readOnlyEntry) TypedKey(key *fyne.KeyEvent) {
	// Allow navigation
	switch key.Name {
	case fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight,
		fyne.KeyPageUp, fyne.KeyPageDown, fyne.KeyHome, fyne.KeyEnd:
		e.Entry.TypedKey(key)
	}
	// Ignore editing keys (Backspace, Delete, Enter, etc.)
}

func (e *readOnlyEntry) TypedShortcut(shortcut fyne.Shortcut) {
	// Allow Copy
	if _, ok := shortcut.(*fyne.ShortcutCopy); ok {
		e.Entry.TypedShortcut(shortcut)
	}
	// Ignore Cut/Paste
}
