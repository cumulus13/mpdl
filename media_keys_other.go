//go:build !windows

// File: media_keys_other.go
// Stub so media_keys.go's call to monitorWindowsNative compiles on
// non-Windows platforms. It is never actually invoked at runtime there —
// MediaKeyMonitor.Start() dispatches by runtime.GOOS before calling
// monitorWindows() at all — but Go still needs the symbol to exist for
// the build to succeed on Linux/macOS.
//
// License: MIT

package main

import "fmt"

func (m *MediaKeyMonitor) monitorWindowsNative() error {
	return fmt.Errorf("windows hotkeys unavailable on %s", "this platform")
}
