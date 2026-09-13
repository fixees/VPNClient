//go:build windows

package winutil

import (
	"fmt"
	"syscall"
	"unsafe"

	"myinternetvpn/client/internal/defaults"
)

// Global mutex so elevated / non-elevated launches still collide correctly.
const singleInstanceMutex = defaults.MutexName

var singleInstanceHandle syscall.Handle

// AcquireSingleInstance returns false when another instance already owns the mutex.
func AcquireSingleInstance() (bool, error) {
	name, err := syscall.UTF16PtrFromString(singleInstanceMutex)
	if err != nil {
		return true, err
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createMutex := kernel32.NewProc("CreateMutexW")
	getLastError := kernel32.NewProc("GetLastError")

	handle, _, callErr := createMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		if callErr != nil {
			return true, fmt.Errorf("CreateMutex: %w", callErr)
		}
		return true, fmt.Errorf("CreateMutex failed")
	}
	errno, _, _ := getLastError.Call()
	const errorAlreadyExists = 183 // ERROR_ALREADY_EXISTS
	if errno == errorAlreadyExists {
		_, _, _ = kernel32.NewProc("CloseHandle").Call(handle)
		return false, nil
	}
	singleInstanceHandle = syscall.Handle(handle)
	return true, nil
}

// ReleaseSingleInstance closes the instance mutex handle.
func ReleaseSingleInstance() {
	if singleInstanceHandle != 0 {
		_ = syscall.CloseHandle(singleInstanceHandle)
		singleInstanceHandle = 0
	}
}
