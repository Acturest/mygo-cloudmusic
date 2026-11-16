//go:build !windows

package app

// allowForeground does nothing here: the systems show a window another
// instance raises as they do any window.
func allowForeground() {}
