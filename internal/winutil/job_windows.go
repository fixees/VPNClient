//go:build windows

package winutil

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows job object that kills all assigned children when this process exits
// (Task Manager / crash / hard kill of the UI). Prevents orphan mihomo.exe.
var (
	childJobOnce sync.Once
	childJob     windows.Handle
	childJobErr  error
)

func ensureChildKillJob() (windows.Handle, error) {
	childJobOnce.Do(func() {
		h, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			childJobErr = fmt.Errorf("CreateJobObject: %w", err)
			return
		}
		var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
		info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
		ret, err := windows.SetInformationJobObject(
			h,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		)
		if ret == 0 {
			_ = windows.CloseHandle(h)
			if err == nil {
				err = fmt.Errorf("SetInformationJobObject failed")
			}
			childJobErr = fmt.Errorf("SetInformationJobObject: %w", err)
			return
		}
		childJob = h
	})
	return childJob, childJobErr
}

// AssignProcessToChildKillJob ties pid to a job that dies with this process.
func AssignProcessToChildKillJob(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid")
	}
	job, err := ensureChildKillJob()
	if err != nil {
		return err
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("OpenProcess(%d): %w", pid, err)
	}
	defer windows.CloseHandle(h)
	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		return fmt.Errorf("AssignProcessToJobObject(%d): %w", pid, err)
	}
	return nil
}
