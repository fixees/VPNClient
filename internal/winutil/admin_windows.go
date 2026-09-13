//go:build windows

package winutil

import (
	"fmt"
	"os"
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
	cwdDir, err := os.Getwd()
	if err != nil {
		cwdDir = ""
	}
	argStr := joinArgs(args)
	var params *uint16
	if argStr != "" {
		params, err = syscall.UTF16PtrFromString(argStr)
		if err != nil {
			return err
		}
	}
	cwd, _ := syscall.UTF16PtrFromString(cwdDir)
	const swShownormal = 1
	return windows.ShellExecute(0, verb, exePtr, params, cwd, swShownormal)
}

// EnsureAdmin exits after launching an elevated copy when the process is not elevated.
// Returns true when the current process may continue as administrator.
func EnsureAdmin() (bool, error) {
	if IsAdmin() {
		return true, nil
	}
	args := os.Args[1:]
	if err := RelaunchAsAdmin(args...); err != nil {
		return false, fmt.Errorf("%w: %v", ErrNeedAdmin, err)
	}
	return false, nil
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

func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "" {
			continue
		}
		if needsQuote(a) {
			out = append(out, `"`+a+`"`)
			continue
		}
		out = append(out, a)
	}
	return joinSpace(out)
}

func needsQuote(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' {
			return true
		}
	}
	return false
}

func joinSpace(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	n := len(parts) - 1
	for _, p := range parts {
		n += len(p)
	}
	b := make([]byte, 0, n)
	for i, p := range parts {
		if i > 0 {
			b = append(b, ' ')
		}
		b = append(b, p...)
	}
	return string(b)
}
