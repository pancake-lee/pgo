package pclient

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// PageBuilder is a function that builds the center panel content for a page.
// It receives the App so business code can access OutputPanel, Window, etc.
type PageBuilder func(app *App) fyne.CanvasObject

// App represents the main GUI application
// 左侧：功能列表
// 中间：可切换的业务内容（参数输入 + 数据列表）
// 右侧：日志输出 + 结果输出
type App struct {
	window          fyne.Window
	leftContainer   *fyne.Container // Contains menu buttons
	centerContainer *fyne.Container // Swappable center content
	outputPanel     *OutputPanel
}

// NewApp creates a new GUI application
func NewApp(title string) *App {
	a := app.New()
	w := a.NewWindow(title)
	w.Resize(fyne.NewSize(1500, 700))

	guiApp := &App{
		window:          w,
		leftContainer:   container.NewVBox(),
		centerContainer: container.NewStack(),
		outputPanel:     NewOutputPanel(),
	}

	guiApp.outputPanel.SetWindow(w)
	guiApp.buildLayout()
	return guiApp
}

// Window returns the fyne window
func (a *App) Window() fyne.Window {
	return a.window
}

// AddMenuItem adds a menu item to the left panel
func (a *App) AddMenuItem(name string, onClick func()) {
	btn := widget.NewButton(name, onClick)
	a.leftContainer.Add(btn)
}

// RegisterPage registers a named page with the left menu.
// When the user clicks the menu item, the PageBuilder is called
// and its result is displayed in the center panel.
func (a *App) RegisterPage(name string, builder PageBuilder) {
	a.AddMenuItem(name, func() {
		content := builder(a)
		a.SetCenterContent(content)
	})
}

// SetCenterContent sets the center panel content
func (a *App) SetCenterContent(content fyne.CanvasObject) {
	a.centerContainer.Objects = []fyne.CanvasObject{content}
	a.centerContainer.Refresh()
}

// OutputPanel returns the output panel for log/result management
func (a *App) OutputPanel() *OutputPanel {
	return a.outputPanel
}

// ShowAndRun shows the window and runs the app
func (a *App) ShowAndRun() {
	a.window.ShowAndRun()
}

func (a *App) buildLayout() {
	// Left: menu header + buttons
	leftHeader := widget.NewLabel("功能列表")
	leftSpacer := newVSpacer(100)
	leftContent := container.NewVBox(leftHeader, leftSpacer)
	leftWithSep := container.NewBorder(nil, nil, nil, newSeparator(), leftContent)

	// Center: swappable content with separator
	centerWithSep := container.NewBorder(nil, nil, nil, newSeparator(), a.centerContainer)

	// Right: output panel
	rightContent := a.outputPanel.Container()

	// Root: Left | Center | Right
	rootBorder := container.NewBorder(nil, nil, leftWithSep, nil,
		container.NewBorder(nil, nil, centerWithSep, nil, rightContent))

	a.window.SetContent(rootBorder)
}
