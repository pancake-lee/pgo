package putil

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

// 命令行交互
type _interact struct{}

var Interact _interact

func (_interact) MustInput(msg string) (value string) {
	for i := 0; i < 3; i++ {
		value = Interact.Input(msg)
		if value != "" {
			Interact.Debugf("Your input: %v", value)
			break
		} else {
			Interact.Warnf("No input! retry[%v] ...", i+1)
		}
	}
	if value == "" {
		Interact.Warnf("No input! exit")
		os.Exit(1)
	}
	return value
}
func (_interact) Input(msg string) (value string) {
	value, _ = pterm.DefaultInteractiveTextInput.
		WithDefaultText(msg).Show()
	return value
}

func (_interact) MustConfirm(msg string) {
	result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultText(msg).WithDefaultValue(true).Show()
	if result {
		Interact.Debugf("Confirmed")
	} else {
		Interact.Warnf("Exited")
		os.Exit(1)
	}
}

// --------------------------------------------------
type _selector struct {
	msg       string
	options   []string
	callbacks map[string]func()
}

func (_interact) NewSelector(msg string) *_selector {
	return &_selector{
		msg:       msg,
		callbacks: make(map[string]func()),
	}
}

func (s *_selector) Reg(itemMsg string, cb func()) {
	s.options = append(s.options, itemMsg)
	s.callbacks[itemMsg] = cb
}

func (s *_selector) Run() {
	if len(s.options) == 0 {
		Interact.Warnf("No options registered")
		return
	}

	selected, _ := pterm.DefaultInteractiveSelect.
		WithDefaultText(s.msg).
		WithOptions(s.options).
		Show()

	Interact.Debugf("Selected: %s", selected)

	if cb, ok := s.callbacks[selected]; ok && cb != nil {
		cb()
	}
}

func (s *_selector) Loop() {
	if len(s.options) == 0 {
		Interact.Warnf("No options registered")
		return
	}

	backOpt := "回到上一级 (Back)"
	// create a new slice with 'Back' option
	opts := append(s.options, backOpt)

	for {
		selected, _ := pterm.DefaultInteractiveSelect.
			WithDefaultText(s.msg).
			WithMaxHeight(10).
			WithOptions(opts).
			Show()

		if selected == backOpt {
			return
		}

		Interact.Debugf("Selected: %s", selected)

		if cb, ok := s.callbacks[selected]; ok && cb != nil {
			cb()
		}
	}
}

// --------------------------------------------------

func (_interact) Warnf(msg string, args ...interface{}) {
	pterm.Println(pterm.Yellow(fmt.Sprintf(msg, args...)))
}
func (_interact) Errorf(msg string, args ...interface{}) {
	pterm.Println(pterm.Red(fmt.Sprintf(msg, args...)))
}

func (_interact) Debugf(msg string, args ...interface{}) {
	pterm.Println(pterm.Cyan(fmt.Sprintf(msg, args...)))
}

func (_interact) Infof(msg string, args ...interface{}) {
	pterm.Println(pterm.Green(fmt.Sprintf(msg, args...)))
}

func (_interact) PrintLine() {
	msg := "---------------------------------------------"
	pterm.Println(pterm.Cyan(msg))
}
