//go:build windows

package app

import "syscall"

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
)

// allowForeground passes this launch's right to bring a window to the
// front on to the instance it asks to show itself: Windows otherwise only
// flashes the icon of a window another process raises in the background.
func allowForeground() {
	const asfwAny = 0xFFFFFFFF
	_, _, _ = procAllowSetForegroundWindow.Call(asfwAny)
}
