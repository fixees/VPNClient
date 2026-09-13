package winutil

import "errors"

var (
	ErrNotWindows = errors.New("windows-only feature")
	ErrNeedAdmin  = errors.New("administrator privileges required")
)
