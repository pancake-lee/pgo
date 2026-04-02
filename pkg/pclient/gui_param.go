package pclient

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// DataItem represents an item in the data list (bottom of param panel)
// 提供字符串列表，字符串由业务代码自己编解码，底层直接存储/展示字符串即可
// 提供两个按钮：修改和删除，逻辑由业务代码传入callback
type DataItem struct {
	ID       any
	Display  string
	OnEdit   func(id any)
	OnDelete func(id any)
}

// ParamPanel is the center panel for parameters
// Top: Form inputs (字符串输入、日期选择、下拉选择)
// Bottom: Data list with edit/delete (展示当前已存在数据)
type ParamPanel struct {
	container   *fyne.Container
	placeholder *widget.Label
	formItems   []*widget.FormItem
	form        *widget.Form
	dataList    *widget.List
	dataItems   []DataItem
	dataBinding binding.UntypedList
}

// FormInput represents a single form input field
type FormInput struct {
	Label  string
	Widget fyne.CanvasObject
}

// NewParamPanel creates a new param panel
func NewParamPanel() *ParamPanel {
	p := &ParamPanel{
		dataItems:   make([]DataItem, 0),
		dataBinding: binding.NewUntypedList(),
	}
	p.placeholder = widget.NewLabel("请先从左侧菜单选择一个功能")
	p.container = container.NewStack(p.placeholder)
	return p
}

// SetForm sets the top form section with form inputs
func (p *ParamPanel) SetForm(inputs []FormInput, onSubmit func()) {
	p.formItems = make([]*widget.FormItem, len(inputs))
	for i, input := range inputs {
		p.formItems[i] = &widget.FormItem{Text: input.Label, Widget: input.Widget}
	}
	p.form = &widget.Form{Items: p.formItems, OnSubmit: onSubmit}
	p.updateContainer()
}

// SetDataList sets the bottom data list section
// 提供两个按钮：修改和删除
func (p *ParamPanel) SetDataList(items []DataItem) {
	p.dataItems = items
	p.dataList = widget.NewListWithData(
		p.dataBinding,
		func() fyne.CanvasObject {
			editBtn := widget.NewButton("修改", nil)
			delBtn := widget.NewButton("删除", nil)
			buttons := container.NewHBox(editBtn, delBtn)
			return container.NewBorder(
				nil, nil, nil, buttons,
				widget.NewLabel("placeholder"))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			val, _ := i.(binding.Untyped).Get()
			item, ok := val.(DataItem)
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
				if editBtn, ok := buttons.Objects[0].(*widget.Button); ok {
					if item.OnEdit != nil {
						editBtn.Show()
						capturedItem := item
						editBtn.OnTapped = func() {
							capturedItem.OnEdit(capturedItem.ID)
						}
					} else {
						editBtn.Hide()
					}
				}
				if delBtn, ok := buttons.Objects[1].(*widget.Button); ok {
					if item.OnDelete != nil {
						delBtn.Show()
						capturedItem := item
						delBtn.OnTapped = func() {
							capturedItem.OnDelete(capturedItem.ID)
						}
					} else {
						delBtn.Hide()
					}
				}
			}
		},
	)
	p.RefreshDataList()
	p.updateContainer()
}

// RefreshDataList refreshes the data list from dataItems
func (p *ParamPanel) RefreshDataList() {
	items := make([]any, len(p.dataItems))
	for i, item := range p.dataItems {
		items[i] = item
	}
	p.dataBinding.Set(items)
}

// UpdateDataList updates the data list with new items
func (p *ParamPanel) UpdateDataList(items []DataItem) {
	p.dataItems = items
	p.RefreshDataList()
}

// Container returns the underlying fyne container
func (p *ParamPanel) Container() fyne.CanvasObject {
	return p.container
}

// SetContent replaces the placeholder with actual content
func (p *ParamPanel) SetContent(content fyne.CanvasObject) {
	p.container.Objects = []fyne.CanvasObject{content}
	p.container.Refresh()
}

func (p *ParamPanel) updateContainer() {
	if p.form == nil {
		return
	}

	sep := newSeparator()
	var content fyne.CanvasObject
	if p.dataList != nil {
		content = container.NewBorder(
			container.NewVBox(p.form, sep, widget.NewLabel("记录列表")),
			nil, nil, nil,
			p.dataList)
	} else {
		content = container.NewVBox(p.form)
	}
	p.container.Objects = []fyne.CanvasObject{content}
	p.container.Refresh()
}

// Clear clears the panel back to placeholder
func (p *ParamPanel) Clear() {
	p.container.Objects = []fyne.CanvasObject{p.placeholder}
	p.container.Refresh()
}
