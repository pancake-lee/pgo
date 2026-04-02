//go:build windows

package pclient

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// ReadOnlyEntry is a widget.Entry that prevents editing
// but keeps normal text color and allows selection/copy.
type ReadOnlyEntry struct {
	widget.Entry
}

func newReadOnlyEntry() *ReadOnlyEntry {
	entry := &ReadOnlyEntry{}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *ReadOnlyEntry) TypedRune(r rune) {
	// Ignore typing
}

func (e *ReadOnlyEntry) TypedKey(key *fyne.KeyEvent) {
	// Allow navigation
	switch key.Name {
	case fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight,
		fyne.KeyPageUp, fyne.KeyPageDown, fyne.KeyHome, fyne.KeyEnd:
		e.Entry.TypedKey(key)
	}
	// Ignore editing keys (Backspace, Delete, Enter, etc.)
}

func (e *ReadOnlyEntry) TypedShortcut(shortcut fyne.Shortcut) {
	// Allow Copy
	if _, ok := shortcut.(*fyne.ShortcutCopy); ok {
		e.Entry.TypedShortcut(shortcut)
	}
	// Ignore Cut/Paste
}

// newSeparator creates a horizontal gray separator line
func newSeparator() fyne.CanvasObject {
	sep := canvas.NewRectangle(color.Gray{Y: 128})
	sep.SetMinSize(fyne.NewSize(0, 3))
	return sep
}

// newVSpacer creates a vertical spacer
func newVSpacer(height float32) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, height))
	return spacer
}
