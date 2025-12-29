//go:build windows

package main

import (
	"fmt"
	"io"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/pancake-lee/pgo/client/courseSwap"
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
			fmt.Println(err)
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

	// Redirect stdout to capture output
	// Note: This is a simple redirection for demonstration.
	// Real-time capturing might require a pipe and a goroutine.
	r, wPipe, _ := os.Pipe()
	originalStdout := os.Stdout
	os.Stdout = wPipe

	// Capture output in a goroutine
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				current, _ := outputData.Get()
				outputData.Set(current + string(buf[:n]))
			}
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(originalStdout, "Error reading stdout: %v\n", err)
				}
				break
			}
		}
	}()

	btnCourseSwap := widget.NewButton("调课 (Course Swap)", func() {
		output.SetText("Starting Course Swap...\n")

		// Mock input for now or use a dialog to get input
		// Since courseSwap.CourseSwap() uses putil.Interact.Input which reads from Stdin,
		// we need to adapt courseSwap to accept config or mock the interaction.
		// For this task, "读取缓存或者输入必要参数后运行程序"
		// We can try to load cache and run if possible, or pop up a dialog.

		// Ideally, we should refactor courseSwap to take an interface for input/output.
		// But for now, let's assume we can just run it and it might block on Stdin if we don't handle it.
		// To avoid blocking, we might need to change how inputParams works or provide a UI implementation of Interact.

		// Let's try to run it in a goroutine so UI doesn't freeze,
		// BUT inputParams reads from Stdin.
		// We need to override putil.Interact to use UI dialogs.

		// Let's show a form for Course Swap Params
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

	// Restore stdout
	wPipe.Close()
	os.Stdout = originalStdout
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
				// We need to capture stdout again inside the goroutine if we want it to show up?
				// The global os.Stdout redirect should work for the process.

				courseSwap.Run(config)

				// Append "Done"
				// output.SetText(output.Text + "\nDone.\n") // Accessing UI from goroutine needs care?
				// Fyne widgets are generally thread-safe for SetText? No, usually need driver.
				// But for simplicity let's rely on the stdout capture goroutine to update UI.
				fmt.Println("\nCourse Swap Finished.")
			}()
		},
	}

	w := fyne.CurrentApp().NewWindow("Course Swap Input")
	w.SetContent(form)
	w.Resize(fyne.NewSize(400, 300))
	w.Show()
}
