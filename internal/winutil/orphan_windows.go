//go:build windows

package winutil

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const processTerminateOnly = 0x0001

var procTerminateProcess = modKernel32.NewProc("TerminateProcess")

// KillProcessesWithImagePath terminates processes whose full image path matches exePath
// (used to reap orphan mihomo.exe left after a hard kill of the UI).
func KillProcessesWithImagePath(exePath string) (int, error) {
	want, err := filepath.Abs(filepath.Clean(exePath))
	if err != nil {
		want = filepath.Clean(exePath)
	}
	want = strings.ToLower(want)
	if want == "" {
		return 0, fmt.Errorf("empty exe path")
	}

	snap, _, snapErr := procCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snap == 0 || snap == ^uintptr(0) {
		return 0, snapErr
	}
	defer procCloseHandle.Call(snap)

	var entry processEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))
	ret, _, _ := procProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return 0, nil
	}

	self := uint32(syscall.Getpid())
	killed := 0
	var firstErr error
	for {
		pid := entry.ProcessID
		if pid != 0 && pid != self {
			path := queryProcessPath(pid)
			if path != "" && strings.ToLower(filepath.Clean(path)) == want {
				if err := terminatePID(pid); err != nil {
					if firstErr == nil {
						firstErr = err
					}
				} else {
					killed++
				}
			}
		}
		ret, _, _ = procProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}
	return killed, firstErr
}

func terminatePID(pid uint32) error {
	h, _, err := procOpenProcess.Call(processTerminateOnly|processQueryLimited, 0, uintptr(pid))
	if h == 0 {
		return fmt.Errorf("OpenProcess(%d): %w", pid, err)
	}
	defer procCloseHandle.Call(h)
	ret, _, termErr := procTerminateProcess.Call(h, 1)
	if ret == 0 {
		return fmt.Errorf("TerminateProcess(%d): %w", pid, termErr)
	}
	return nil
}
