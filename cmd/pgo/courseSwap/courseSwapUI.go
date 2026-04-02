package courseSwap

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/pancake-lee/pgo/cmd/pgo/swagger"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// CourseSwapUIBuilder builds the course swap UI
type CourseSwapUIBuilder struct {
	window       fyne.Window
	logData      binding.String
	resultData   binding.UntypedList
	onCalculate  func(config InputConfig)
	onExecute    func(target *CourseInfo)
	onDelete     func(id int32)
	getHistory   func() ([]swagger.ApiCourseSwapRequestInfo, error)
	fileDialogFn func(initialPath string) string
}

// NewCourseSwapUIBuilder creates a new UI builder
func NewCourseSwapUIBuilder(
	window fyne.Window,
	logData binding.String,
	resultData binding.UntypedList,
) *CourseSwapUIBuilder {
	return &CourseSwapUIBuilder{
		window:     window,
		logData:    logData,
		resultData: resultData,
	}
}

// SetCallbacks sets the action callbacks
func (b *CourseSwapUIBuilder) SetCallbacks(
	onCalculate func(config InputConfig),
	onExecute func(target *CourseInfo),
	onDelete func(id int32),
	getHistory func() ([]swagger.ApiCourseSwapRequestInfo, error),
	fileDialogFn func(initialPath string) string,
) {
	b.onCalculate = onCalculate
	b.onExecute = onExecute
	b.onDelete = onDelete
	b.getHistory = getHistory
	b.fileDialogFn = fileDialogFn
}

// Build creates the course swap UI
func (b *CourseSwapUIBuilder) Build() fyne.CanvasObject {
	cache := LoadCache()

	// --------------------------------------------------
	// Teacher Select
	teacherSelect := widget.NewSelectEntry([]string{})
	teacherSelect.SetText(cache.Teacher)

	updateTeachers := func(path string) {
		if _, err := os.Stat(path); err != nil {
			plogger.LogErr(err)
			return
		}
		go func() {
			teachers, err := GetTeacherList(path)
			if err != nil {
				plogger.LogErr(err)
				return
			}
			teacherSelect.SetOptions(teachers)
		}()
	}

	if cache.Path != "" {
		updateTeachers(cache.Path)
	}

	// --------------------------------------------------
	// Path Entry with file dialog
	pathEntry := widget.NewEntry()
	pathEntry.SetText(cache.Path)

	var debounceTimer *time.Timer
	pathEntry.OnChanged = func(s string) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
			updateTeachers(s)
		})
	}

	browseBtn := widget.NewButton("...", func() {
		if b.fileDialogFn != nil {
			file := b.fileDialogFn(pathEntry.Text)
			if file != "" {
				pathEntry.SetText(file)
			}
		}
	})

	b.window.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
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
		b.openDatePicker(dateEntry.Text, func(s string) {
			dateEntry.SetText(s)
		})
	})
	dateContainer := container.NewBorder(nil, nil, nil, dateBtn, dateEntry)

	// --------------------------------------------------
	// Course Num Select
	var numList []string
	for i := 1; i <= CourseNumMax; i++ {
		numList = append(numList, fmt.Sprintf("%d", i))
	}

	courseNumSelect := widget.NewSelect(numList, nil)
	if cache.CourseNum > 0 && cache.CourseNum <= CourseNumMax {
		courseNumSelect.SetSelected(fmt.Sprintf("%d", cache.CourseNum))
	} else {
		courseNumSelect.SetSelected("1")
	}

	// --------------------------------------------------
	// History List
	historyData := binding.NewUntypedList()
	var updateHistory func()

	historyList := widget.NewListWithData(
		historyData,
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			delBtn := widget.NewButton("删除", nil)
			return container.NewBorder(nil, nil, nil, delBtn, label)
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			b.updateHistoryItem(i, o, &updateHistory, historyData)
		},
	)

	updateHistory = func() {
		if b.getHistory == nil {
			return
		}
		go func() {
			list, err := b.getHistory()
			if err != nil {
				return
			}
			items := b.formatHistoryItems(list)
			historyData.Set(items)
		}()
	}
	updateHistory()

	// --------------------------------------------------
	// IsOddWeek Checkbox
	now := time.Now()
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	monday := now.AddDate(0, 0, offset)
	mondayYMD := putil.TimeToStr(monday, "YYYYMMDD")

	isOddWeekCheck := widget.NewCheck(fmt.Sprintf("本周是单周则打勾（%v为周一的）", mondayYMD), nil)
	isOddWeekCheck.Checked = cache.IsOddWeek

	// --------------------------------------------------
	// Form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "课表", Widget: pathContainer},
			{Text: "教师", Widget: teacherSelect},
			{Text: "日期", Widget: dateContainer},
			{Text: "节次", Widget: courseNumSelect},
			{Text: "单双周", Widget: isOddWeekCheck},
		},
		OnSubmit: func() {
			cNum, _ := strconv.Atoi(courseNumSelect.Selected)
			config := InputConfig{
				Path:        pathEntry.Text,
				Teacher:     teacherSelect.Text,
				Date:        dateEntry.Text,
				CourseNum:   cNum,
				StorageType: "Local",
				IsOddWeek:   isOddWeekCheck.Checked,
			}
			SaveCache(config)

			if b.onCalculate != nil {
				b.onCalculate(config)
			}
		},
	}

	return container.NewBorder(
		container.NewVBox(form, widget.NewLabel("换课记录")),
		nil, nil, nil,
		historyList)
}

func (b *CourseSwapUIBuilder) updateHistoryItem(
	i binding.DataItem,
	o fyne.CanvasObject,
	updateHistory *func(),
	historyData binding.UntypedList,
) {
	val, _ := i.(binding.Untyped).Get()
	data, ok := val.(map[string]interface{})
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

	if label == nil || btn == nil {
		return
	}

	label.SetText(data["display"].(string))

	if data["expired"].(bool) {
		label.TextStyle = fyne.TextStyle{Italic: true}
	} else {
		label.TextStyle = fyne.TextStyle{}
	}

	btn.OnTapped = func() {
		id := data["id"].(int32)
		if b.onDelete != nil {
			b.onDelete(id)
			if *updateHistory != nil {
				(*updateHistory)()
			}
		}
	}
}

func (b *CourseSwapUIBuilder) formatHistoryItems(list []swagger.ApiCourseSwapRequestInfo) []interface{} {
	var items []interface{}
	today := putil.DateToStrDefault(time.Now())
	for _, info := range list {
		status := "[有效]"
		if info.SrcDate < today {
			status = "[过期]"
		}
		str := fmt.Sprintf(
			"[%s] [%s] [%s] [第%d节] [%v]班 [%v]课 -> "+
				"[%s] [%s] [第%d节] [%v]班 [%v]课",
			status,
			info.SrcTeacher, info.SrcDate, info.SrcCourseNum,
			info.SrcClass, info.SrcCourse,
			info.DstTeacher, info.DstDate, info.DstCourseNum,
			info.DstClass, info.DstCourse)

		items = append(items, map[string]interface{}{
			"id":      info.ID,
			"display": str,
			"expired": info.SrcDate < today,
		})
	}
	return items
}

func (b *CourseSwapUIBuilder) openDatePicker(current string, onSet func(string)) {
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
		}, b.window)
}
