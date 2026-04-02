package pclient

import (
	"bufio"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// ResultItem represents an item in the result list (bottom of output panel)
// 提供字符串列表，字符串由业务代码自己编解码，底层直接存储/展示字符串即可
// 提供两个按钮：复制（点击后复制到粘贴板）和确认（逻辑由业务代码传入callback）
type ResultItem struct {
	Display   string
	OnConfirm func()
}

// OutputPanel is the right-side output panel
// Top: Log output (展示本程序日志)
// Bottom: Result list (本程序输出展示)
type OutputPanel struct {
	container     *fyne.Container
	logData       binding.String
	logEntry      *ReadOnlyEntry
	resultList    *widget.List
	resultItems   []ResultItem
	resultBinding binding.UntypedList
	window        fyne.Window
}

// NewOutputPanel creates a new output panel
func NewOutputPanel() *OutputPanel {
	p := &OutputPanel{
		logData:       binding.NewString(),
		resultItems:   make([]ResultItem, 0),
		resultBinding: binding.NewUntypedList(),
	}
	p.build()
	return p
}

// SetWindow sets the window reference for clipboard operations
func (p *OutputPanel) SetWindow(w fyne.Window) {
	p.window = w
}

func (p *OutputPanel) build() {
	p.logEntry = newReadOnlyEntry()
	p.logEntry.MultiLine = true
	p.logEntry.SetPlaceHolder("Logs will appear here...")
	p.logEntry.SetMinRowsVisible(10)

	p.logData.AddListener(binding.NewDataListener(func() {
		val, _ := p.logData.Get()
		p.logEntry.SetText(val)
	}))

	// Start capturing output
	p.startCapture()

	p.resultList = widget.NewListWithData(
		p.resultBinding,
		func() fyne.CanvasObject {
			copyBtn := widget.NewButton("复制", nil)
			confirmBtn := widget.NewButton("确认", nil)
			buttons := container.NewHBox(copyBtn, confirmBtn)
			return container.NewBorder(
				nil, nil, nil, buttons,
				widget.NewLabel("placeholder"))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			val, _ := i.(binding.Untyped).Get()
			item, ok := val.(ResultItem)
			if !ok {
				return
			}

			c := o.(*fyne.Container)
			var label *widget.Label
			var buttons *fyne.Container
			for _, obj := range c.Objects {
				if l, ok := obj.(*widget.Label); ok {
					label = l
				} else if bc, ok := obj.(*fyne.Container); ok {
					buttons = bc
				}
			}

			if label != nil {
				label.SetText(item.Display)
			}
			if buttons != nil && len(buttons.Objects) >= 2 {
				capturedItem := item
				capturedPanel := p
				if copyBtn, ok := buttons.Objects[0].(*widget.Button); ok {
					copyBtn.OnTapped = func() {
						if capturedPanel.window != nil {
							capturedPanel.window.Clipboard().SetContent(capturedItem.Display)
						}
					}
				}
				if confirmBtn, ok := buttons.Objects[1].(*widget.Button); ok {
					if capturedItem.OnConfirm != nil {
						confirmBtn.Enable()
						confirmBtn.OnTapped = func() {
							capturedItem.OnConfirm()
						}
					} else {
						confirmBtn.Disable()
					}
				}
			}
		},
	)

	sep := newSeparator()

	// Right side layout: Log at top, List fills rest
	p.container = container.NewBorder(
		container.NewVBox(widget.NewLabel("日志输出"), p.logEntry, sep),
		nil, nil, nil,
		container.NewBorder(widget.NewLabel("结果输出"), nil, nil, nil, p.resultList))
}

// startCapture redirects stdout/stderr to the log panel
func (p *OutputPanel) startCapture() {
	r, w, err := os.Pipe()
	if err != nil {
		return
	}

	os.Stdout = w
	os.Stderr = w

	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			current, _ := p.logData.Get()
			if len(current) > 100000 {
				current = current[len(current)-50000:]
			}
			p.logData.Set(current + text + "\n")
		}
	}()
}

// AppendLog appends a message to the log
func (p *OutputPanel) AppendLog(msg string) {
	current, _ := p.logData.Get()
	if len(current) > 100000 {
		current = current[len(current)-50000:]
	}
	p.logData.Set(current + msg + "\n")
}

// ClearLog clears the log
func (p *OutputPanel) ClearLog() {
	p.logData.Set("")
}

// SetResults sets the result list items
func (p *OutputPanel) SetResults(items []ResultItem) {
	p.resultItems = items
	resultAnys := make([]any, len(items))
	for i, item := range items {
		resultAnys[i] = item
	}
	p.resultBinding.Set(resultAnys)
}

// AppendResult adds a single result item
func (p *OutputPanel) AppendResult(item ResultItem) {
	p.resultItems = append(p.resultItems, item)
	resultAnys := make([]any, len(p.resultItems))
	for i, ri := range p.resultItems {
		resultAnys[i] = ri
	}
	p.resultBinding.Set(resultAnys)
}

// ClearResults clears the result list
func (p *OutputPanel) ClearResults() {
	p.resultItems = nil
	p.resultBinding.Set(nil)
}

// Container returns the underlying fyne container
func (p *OutputPanel) Container() fyne.CanvasObject {
	return p.container
}

// LogData returns the log binding for external log management
func (p *OutputPanel) LogData() binding.String {
	return p.logData
}
