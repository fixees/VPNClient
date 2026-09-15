//go:build windows

package winutil

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"myinternetvpn/client/internal/defaults"

	"golang.org/x/sys/windows"
)

// Global mutex so elevated / non-elevated launches still collide correctly.
const singleInstanceMutex = defaults.MutexName

const errorAlreadyExists = 183 // ERROR_ALREADY_EXISTS

var singleInstanceHandle windows.Handle

// InstanceAlreadyRunning reports whether another process owns the single-instance mutex.
// Safe to call before elevation (OpenMutex does not create / take ownership).
func InstanceAlreadyRunning() bool {
	name, err := windows.UTF16PtrFromString(singleInstanceMutex)
	if err != nil {
		return false
	}
	const desiredAccess = windows.SYNCHRONIZE | windows.MUTEX_MODIFY_STATE
	h, err := windows.OpenMutex(desiredAccess, false, name)
	if err != nil || h == 0 {
		return false
	}
	_ = windows.CloseHandle(h)
	return true
}

// AcquireSingleInstance returns false when another instance already owns the mutex.
// On CreateMutex failure it fail-closes (returns false) so a second UI cannot start.
func AcquireSingleInstance() (bool, error) {
	name, err := windows.UTF16PtrFromString(singleInstanceMutex)
	if err != nil {
		return false, err
	}
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			_ = windows.CloseHandle(h)
		}
		return false, nil
	}
	if err != nil {
		// Some Go/Windows combos surface already-exists only via Errno on a non-nil handle.
		if errno, ok := err.(syscall.Errno); ok && errno == errorAlreadyExists {
			if h != 0 {
				_ = windows.CloseHandle(h)
			}
			return false, nil
		}
		if h != 0 {
			_ = windows.CloseHandle(h)
		}
		return false, fmt.Errorf("CreateMutex: %w", err)
	}
	if h == 0 {
		return false, fmt.Errorf("CreateMutex failed")
	}
	singleInstanceHandle = h
	return true, nil
}

// ReleaseSingleInstance closes the instance mutex handle.
func ReleaseSingleInstance() {
	if singleInstanceHandle != 0 {
		_ = windows.CloseHandle(singleInstanceHandle)
		singleInstanceHandle = 0
	}
}

// ActivateExistingClient brings the already-running app to the foreground.
func ActivateExistingClient(windowTitle string) bool {
	titles := []string{
		strings.TrimSpace(windowTitle),
		defaults.WindowTitle,
		defaults.ProductName,
		defaults.ProductClientName,
		"MyInternetVPN",
	}
	seen := map[string]struct{}{}
	for _, t := range titles {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		if ActivateMainWindow(t) {
			return true
		}
	}
	exes := []string{
		defaults.ProductExe,
		"MyInternetVPN.exe",
		"MyInternetVPN-fixed.exe",
		"MoyVPN-Client.exe",
	}
	return ActivateByExeNames(exes)
}

// ActivateByExeNames finds a visible/hidden top-level window owned by one of the exe basenames.
func ActivateByExeNames(exes []string) bool {
	want := map[string]struct{}{}
	for _, e := range exes {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		want[e] = struct{}{}
	}
	if len(want) == 0 {
		return false
	}

	user32 := windows.NewLazySystemDLL("user32.dll")
	enumWindows := user32.NewProc("EnumWindows")
	isWindowVisible := user32.NewProc("IsWindowVisible")
	getWindowTextW := user32.NewProc("GetWindowTextW")
	getWindowTextLengthW := user32.NewProc("GetWindowTextLengthW")
	getWindowThreadProcessId := user32.NewProc("GetWindowThreadProcessId")
	showWindow := user32.NewProc("ShowWindow")
	setForeground := user32.NewProc("SetForegroundWindow")
	isIconic := user32.NewProc("IsIconic")
	bringToTop := user32.NewProc("BringWindowToTop")
	allowSetForeground := user32.NewProc("AllowSetForegroundWindow")

	type result struct {
		hwnd uintptr
	}
	var found result

	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var pid uint32
		_, _, _ = getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid == 0 {
			return 1
		}
		base, ok := processBaseName(pid)
		if !ok {
			return 1
		}
		if _, hit := want[base]; !hit {
			return 1
		}
		// Prefer windows that have a title (main UI), skip tool/tray ghost HWNDs with empty titles when possible.
		length, _, _ := getWindowTextLengthW.Call(hwnd)
		visible, _, _ := isWindowVisible.Call(hwnd)
		if length == 0 && visible == 0 {
			return 1
		}
		if length > 0 {
			buf := make([]uint16, length+1)
			_, _, _ = getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
			title := windows.UTF16ToString(buf)
			// Skip obvious non-main chrome if we can.
			if strings.EqualFold(title, "MSCTFIME UI") || strings.EqualFold(title, "Default IME") {
				return 1
			}
		}
		found.hwnd = hwnd
		return 0 // stop
	})

	_, _, _ = enumWindows.Call(cb, 0)
	if found.hwnd == 0 {
		return false
	}

	_, _, _ = allowSetForeground.Call(uintptr(^uint32(0))) // ASFW_ANY
	iconic, _, _ := isIconic.Call(found.hwnd)
	if iconic != 0 {
		_, _, _ = showWindow.Call(found.hwnd, swRestore)
	} else {
		_, _, _ = showWindow.Call(found.hwnd, swShow)
	}
	_, _, _ = bringToTop.Call(found.hwnd)
	_, _, _ = setForeground.Call(found.hwnd)
	return true
}

func processBaseName(pid uint32) (string, bool) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", false
	}
	defer windows.CloseHandle(h)

	var buf [windows.MAX_PATH]uint16
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", false
	}
	path := windows.UTF16ToString(buf[:size])
	return strings.ToLower(filepath.Base(path)), true
}
