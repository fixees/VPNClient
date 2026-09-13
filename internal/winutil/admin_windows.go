//go:build windows

package winutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// IsAdmin reports whether the current process has an elevated admin token.
func IsAdmin() bool {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	type tokenElevation struct {
		TokenIsElevated uint32
	}
	var elevation tokenElevation
	var outLen uint32
	err = windows.GetTokenInformation(
		token,
		windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elevation)),
		uint32(unsafe.Sizeof(elevation)),
		&outLen,
	)
	return err == nil && elevation.TokenIsElevated != 0
}

// RelaunchAsAdmin restarts the current executable with a UAC elevation prompt.
func RelaunchAsAdmin(args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	argStr := strings.Join(args, " ")
	var params *uint16
	if argStr != "" {
		params, err = syscall.UTF16PtrFromString(argStr)
		if err != nil {
			return err
		}
	}
	cwd, _ := syscall.UTF16PtrFromString("")
	var show int32 = 1
	return windows.ShellExecute(0, verb, exePtr, params, cwd, show)
}

// RequireAdminForTUN returns ErrNeedAdmin when TUN is requested without elevation.
func RequireAdminForTUN(tunEnabled bool) error {
	if !tunEnabled {
		return nil
	}
	if IsAdmin() {
		return nil
	}
	return fmt.Errorf("%w: TUN mode requires elevation", ErrNeedAdmin)
}

func runNetsh(args ...string) error {
	cmd := exec.Command("netsh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh %s: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
