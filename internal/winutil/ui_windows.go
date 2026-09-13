//go:build windows

package winutil

import (
	"syscall"
	"unsafe"
)

const (
	mbOK              = 0x00000000
	mbIconError       = 0x00000010
	mbIconInformation = 0x00000040
	mbTopmost         = 0x00040000
	mbSetForeground   = 0x00010000

	swRestore = 9
	swShow    = 5
)

// MessageBox shows a native Windows message dialog (works with -H windowsgui).
func MessageBox(title, text string, isError bool) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	flags := uintptr(mbOK | mbSetForeground | mbTopmost | mbIconInformation)
	if isError {
		flags = uintptr(mbOK | mbSetForeground | mbTopmost | mbIconError)
	}
	_, _, _ = proc.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), flags)
}

// ActivateMainWindow finds an existing app window by title and brings it to the foreground.
func ActivateMainWindow(windowTitle string) bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	findWindow := user32.NewProc("FindWindowW")
	showWindow := user32.NewProc("ShowWindow")
	setForeground := user32.NewProc("SetForegroundWindow")
	isIconic := user32.NewProc("IsIconic")
	bringToTop := user32.NewProc("BringWindowToTop")

	title, err := syscall.UTF16PtrFromString(windowTitle)
	if err != nil {
		return false
	}
	hwnd, _, _ := findWindow.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return false
	}
	iconic, _, _ := isIconic.Call(hwnd)
	if iconic != 0 {
		_, _, _ = showWindow.Call(hwnd, swRestore)
	} else {
		_, _, _ = showWindow.Call(hwnd, swShow)
	}
	_, _, _ = bringToTop.Call(hwnd)
	_, _, _ = setForeground.Call(hwnd)
	return true
}

// HandleSecondInstance activates the running UI and informs the user.
func HandleSecondInstance(windowTitle string) {
	_ = ActivateMainWindow(windowTitle)
	MessageBox(windowTitle, "Приложение уже запущено.\nОткрыто существующее окно.", false)
}
