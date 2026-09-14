package winutil

// RunningApp is a unique process image suitable for PROCESS-NAME routing.
type RunningApp struct {
	Name string `json:"name"` // e.g. ayugram.exe
	Path string `json:"path"` // full path when available
	PID  uint32 `json:"pid"`
}
