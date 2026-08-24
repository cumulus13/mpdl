//go:build windows

// File: media_keys_windows.go
// Real WinAPI RegisterHotKey implementation for global media keys on
// Windows. Runs entirely in-process — no subprocess, no AutoHotkey.
//
// RegisterHotKey does not need a visible window, but Win32 message
// queues are thread-affine, so the registering goroutine is pinned to
// one OS thread for its whole lifetime via runtime.LockOSThread().
//
// License: MIT

package main

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Virtual-key codes for the standard multimedia keys.
const (
	vkMediaPlayPause = 0xB3
	vkMediaNextTrack = 0xB0
	vkMediaPrevTrack = 0xB1
	vkMediaStop      = 0xB2
	vkVolumeUp       = 0xAF
	vkVolumeDown     = 0xAE
)

const wmHotkey = 0x0312

// Hotkey IDs — arbitrary, just need to be unique per RegisterHotKey call
// on this thread.
const (
	hkPlayPause = 1
	hkNext      = 2
	hkPrev      = 3
	hkStop      = 4
	hkVolUp     = 5
	hkVolDown   = 6
)

var (
	user32             = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey = user32.NewProc("RegisterHotKey")
	procUnregisterHK   = user32.NewProc("UnregisterHotKey")
	procGetMessageW    = user32.NewProc("GetMessageW")
	procPostThreadMsg  = user32.NewProc("PostThreadMessageW")
)

// win32Msg mirrors the Win32 MSG struct layout for GetMessageW.
type win32Msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// monitorWindowsNative registers the global media-key hotkeys and blocks,
// dispatching WM_HOTKEY messages to the corresponding action, until
// m.stopChan is closed. Returns an error only if registration could not
// proceed at all (e.g. running under a restricted session).
func (m *MediaKeyMonitor) monitorWindowsNative() error {
	done := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		threadID := windows.GetCurrentThreadId()

		type hotkey struct {
			id int
			vk uintptr
			fn func()
		}
		hotkeys := []hotkey{
			{hkPlayPause, vkMediaPlayPause, m.doPlayPause},
			{hkNext, vkMediaNextTrack, m.doNext},
			{hkPrev, vkMediaPrevTrack, m.doPrev},
			{hkStop, vkMediaStop, m.doStop},
			{hkVolUp, vkVolumeUp, m.doVolUp},
			{hkVolDown, vkVolumeDown, m.doVolDown},
		}

		registered := 0
		for _, hk := range hotkeys {
			r, _, _ := procRegisterHotKey.Call(0, uintptr(hk.id), 0, hk.vk)
			if r == 0 {
				if m.debug {
					log.Printf("⚠️  RegisterHotKey failed for id=%d (vk=0x%X) — key may already be claimed by another app", hk.id, hk.vk)
				}
				continue
			}
			registered++
		}
		if registered == 0 {
			done <- fmt.Errorf("could not register any global hotkeys")
			return
		}
		log.Printf("🎹 Windows: %d/%d global media hotkeys registered (no AHK needed)", registered, len(hotkeys))
		done <- nil

		// Stop-watcher: post WM_QUIT to this thread's message queue so
		// GetMessageW below unblocks when the monitor is stopped.
		stopWatch := make(chan struct{})
		go func() {
			select {
			case <-m.stopChan:
				const wmQuit = 0x0012
				_, _, _ = procPostThreadMsg.Call(uintptr(threadID), wmQuit, 0, 0)
			case <-stopWatch:
			}
		}()
		defer close(stopWatch)
		defer func() {
			for _, hk := range hotkeys {
				_, _, _ = procUnregisterHK.Call(0, uintptr(hk.id))
			}
		}()

		byID := make(map[uintptr]func())
		for _, hk := range hotkeys {
			byID[uintptr(hk.id)] = hk.fn
		}

		var msg win32Msg
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 { // 0 = WM_QUIT, -1 = error
				return
			}
			if msg.Message == wmHotkey {
				if fn, ok := byID[msg.WParam]; ok {
					fn()
				}
			}
		}
	}()

	return <-done
}
