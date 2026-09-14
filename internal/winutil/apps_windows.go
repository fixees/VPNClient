//go:build windows

package winutil

import (
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modKernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = modKernel32.NewProc("Process32FirstW")
	procProcess32NextW           = modKernel32.NewProc("Process32NextW")
	procOpenProcess              = modKernel32.NewProc("OpenProcess")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = modKernel32.NewProc("QueryFullProcessImageNameW")
)

const (
	th32csSnapProcess  = 0x00000002
	processQueryLimited = 0x1000
)

type processEntry32W struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

// ListRunningApps returns unique .exe images currently running (Windows).
func ListRunningApps() ([]RunningApp, error) {
	snap, _, err := procCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snap == 0 || snap == ^uintptr(0) {
		return nil, err
	}
	defer procCloseHandle.Call(snap)

	var entry processEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))
	ret, _, _ := procProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, nil
	}

	byName := map[string]RunningApp{}
	for {
		name := strings.TrimSpace(syscall.UTF16ToString(entry.ExeFile[:]))
		if name != "" && !isNoiseProcess(name) {
			key := strings.ToLower(name)
			path := queryProcessPath(entry.ProcessID)
			cur, exists := byName[key]
			if !exists || (cur.Path == "" && path != "") {
				byName[key] = RunningApp{Name: name, Path: path, PID: entry.ProcessID}
			}
		}
		ret, _, _ = procProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	out := make([]RunningApp, 0, len(byName))
	for _, a := range byName {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func queryProcessPath(pid uint32) string {
	if pid == 0 {
		return ""
	}
	h, _, _ := procOpenProcess.Call(processQueryLimited, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	ret, _, _ := procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return ""
	}
	return filepath.Clean(syscall.UTF16ToString(buf[:size]))
}

func isNoiseProcess(name string) bool {
	n := strings.ToLower(name)
	switch n {
	case "system", "idle", "registry", "smss.exe", "csrss.exe", "wininit.exe",
		"services.exe", "lsass.exe", "svchost.exe", "fontdrvhost.exe",
		"dwm.exe", "conhost.exe", "sihost.exe", "taskhostw.exe",
		"runtimebroker.exe", "searchhost.exe", "startmenuexperiencehost.exe",
		"textinputhost.exe", "ctfmon.exe", "securityhealthservice.exe":
		return true
	}
	return false
}
