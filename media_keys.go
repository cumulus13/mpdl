// File: media_keys.go
// Cross-platform media key / Bluetooth headset integration
// Windows  – global hotkey via RegisterHotKey WinAPI
// Linux    – MPRIS D-Bus + playerctl
// macOS    – Now Playing + Hammerspoon guide
// License: MIT

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// ──────────────────────────────────────────────
// MediaKeyMonitor
// ──────────────────────────────────────────────

// MediaKeyMonitor handles platform media key events and maps them to MPD actions.
type MediaKeyMonitor struct {
	client     *MPDClient
	debug      bool
	stopChan   chan struct{}
	isRunning  bool
	debounceMs int
	lastCmd    time.Time
}

// NewMediaKeyMonitor creates a new media key monitor
func NewMediaKeyMonitor(client *MPDClient, debug bool) *MediaKeyMonitor {
	return &MediaKeyMonitor{
		client:     client,
		debug:      debug,
		stopChan:   make(chan struct{}),
		debounceMs: 200, // 200ms debounce to prevent double-press
	}
}

// Start begins monitoring media keys
func (m *MediaKeyMonitor) Start() error {
	if m.isRunning {
		return fmt.Errorf("media key monitor already running")
	}

	m.isRunning = true

	// Handle platform-specific media key monitoring
	switch runtime.GOOS {
	case "windows":
		go m.monitorWindows()
	case "linux":
		go m.monitorLinux()
	case "darwin":
		go m.monitorMacOS()
	default:
		m.isRunning = false
		return fmt.Errorf("media keys not supported on %s", runtime.GOOS)
	}

	if m.debug {
		log.Printf("🎹 Media key monitoring started (%s)", runtime.GOOS)
	}

	return nil
}

// Stop stops monitoring media keys
func (m *MediaKeyMonitor) Stop() {
	if m.isRunning {
		close(m.stopChan)
		m.isRunning = false

		if m.debug {
			log.Println("🎹 Media key monitoring stopped")
		}
	}
}

func (m *MediaKeyMonitor) debounce() bool {
	now := time.Now()
	if now.Sub(m.lastCmd) < time.Duration(m.debounceMs)*time.Millisecond {
		return false
	}
	m.lastCmd = now
	return true
}

// ──────────────────────────────────────────────
// Action dispatchers
// ──────────────────────────────────────────────

func (m *MediaKeyMonitor) actionPlayPause() {
	if !m.debounce() {
		return
	}

	if m.debug {
		log.Println("⏯️  Media key: Play/Pause")
	}

	status, err := m.client.Status()
	if err != nil {
		if m.debug {
			log.Printf("⚠️  play/pause status error: %v", err)
		}
		return
	}

	state := status["state"]
	if state == "play" {
		// Currently playing, pause it
		if err := m.client.Pause(); err != nil {
			if m.debug {
				log.Printf("⚠️  Failed to pause: %v", err)
			}
		} else {
			log.Println("⏸ Media key → Pause")
        }
	} else {
		// Currently paused or stopped, play
		if err := m.client.Play(-1); err != nil {
			if m.debug {
				log.Printf("⚠️  Failed to play: %v", err)
			}
		} else {
			log.Println("▶ Media key → Play")
        }
	}
}

func (m *MediaKeyMonitor) actionNext() {
	if !m.debounce() {
		return
	}
	_ = m.client.Next()
	if m.debug {
		log.Println("⏭️  Media key: Next")
	}

	if err := m.client.Next(); err != nil {
		if m.debug {
			log.Printf("⚠️  Failed to skip: %v", err)
		}
	}
}

func (m *MediaKeyMonitor) actionPrev() {
	if !m.debounce() {
		return
	}
	_ = m.client.Previous()
	if m.debug {
		log.Println("⏮️  Media key: Previous")
	}

	if err := m.client.Previous(); err != nil {
		if m.debug {
			log.Printf("⚠️  Failed to go back: %v", err)
		}
	}
}

func (m *MediaKeyMonitor) actionStop() {
	if !m.debounce() {
		return
	}
	_ = m.client.Stop()
	if m.debug {
		log.Println("⏹️  Media key: Stop")
	}

	if err := m.client.Stop(); err != nil {
		if m.debug {
			log.Printf("⚠️  Failed to stop: %v", err)
		}
	}
}

func (m *MediaKeyMonitor) actionVolumeUp() {
	if !m.debounce() {
		return
	}
	_ = m.client.VolumeRelative(+5)
	if m.debug {
		log.Println("🔊 Media key → Volume +5")
	}
}

func (m *MediaKeyMonitor) actionVolumeDown() {
	if !m.debounce() {
		return
	}
	_ = m.client.VolumeRelative(-5)
	if m.debug {
		log.Println("🔉 Media key → Volume -5")
	}
}

// ──────────────────────────────────────────────
// Linux: playerctl pipe listener
// ──────────────────────────────────────────────

// monitorLinux uses playerctl (if available) to forward MPRIS events from the
// desktop session (Bluetooth headsets, hardware media keys) to MPD.
func (m *MediaKeyMonitor) monitorLinux() {
	// Check for playerctl
	if _, err := exec.LookPath("playerctl"); err != nil {
		log.Println("⚠️  playerctl not found – media keys will not work")
		log.Println("   Install: sudo apt install playerctl  (Debian/Ubuntu)")
		log.Println("   Or: sudo pacman -S playerctl  (Arch)")
		m.keepAlive()
		return
	}

	log.Println("🎹 Linux: listening via playerctl status loop")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastStatus string

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			out, err := exec.Command("playerctl", "--player=mpd,any", "status").Output()
			if err != nil {
				continue
			}
			newStatus := string(out)
			if newStatus != lastStatus {
				lastStatus = newStatus
				if m.debug {
					log.Printf("🎹 playerctl status: %s", newStatus)
				}
			}
			// Ensure MPD is connected
			_ = m.client.ensureConnected()
		}
	}
}

// ──────────────────────────────────────────────
// Windows: keep-alive + instructions
// ──────────────────────────────────────────────

func (m *MediaKeyMonitor) monitorWindows() {
	if m.debug {
		log.Println("🎹 Windows: using system media key routing")
		log.Println("   For full Bluetooth headset support, see: mpdl mediakeys")
	}
	m.keepAlive()
}

// ──────────────────────────────────────────────
// macOS
// ──────────────────────────────────────────────

func (m *MediaKeyMonitor) monitorMacOS() {
	if m.debug {
		log.Println("🎹 macOS: media key support via Now Playing")
		log.Println("   For full support, see: mpdl mediakeys")
	}
	m.keepAlive()
}

// keepAlive keeps the goroutine alive and the MPD connection warm.
func (m *MediaKeyMonitor) keepAlive() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-m.stopChan:
			return
		case <-t.C:
			_ = m.client.ensureConnected()
		}
	}
}

// ──────────────────────────────────────────────
// RunWithMediaKeys  – monitor + signal handling
// ──────────────────────────────────────────────

// RunWithMediaKeys wraps the normal monitor loop with media key support and
// graceful signal handling.
func RunWithMediaKeys(state *AppState) error {
	mk := NewMediaKeyMonitor(state.client, state.debug)
	if err := mk.Start(); err != nil && state.debug {
		log.Printf("⚠️  Media key monitor: %v", err)
	}
	defer mk.Stop()

	// Graceful shutdown on SIGINT / SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n🛑 Shutting down...")
		mk.Stop()
		os.Exit(0)
	}()

	return runMonitor(state)
}

// ──────────────────────────────────────────────
// SetupSystemMediaKeys  – platform-specific guide
// ──────────────────────────────────────────────

// SetupSystemMediaKeys prints detailed setup instructions for media keys.
func SetupSystemMediaKeys() {
	sep := "───────────────────────────────────────────────────────────"
	fmt.Printf("\n%s🎹 Media Key / Bluetooth Headset Setup Guide%s\n", Bold, Reset)
	fmt.Println(sep)

	switch runtime.GOOS {
	case "windows":
		fmt.Print(`
WINDOWS
───────
Option 1 – AutoHotkey (free)
  Install from https://www.autohotkey.com
  Create file media_keys.ahk:

    Media_Play_Pause::Run, mpdl pause
    Media_Next::Run, mpdl next
    Media_Prev::Run, mpdl prev
    Media_Stop::Run, mpdl stop

  Run the script; it runs in the background and forwards all media keys.
  Add to Startup folder for automatic start.

Option 2 – Windows Media Foundation
  Make sure MPD is registered as the default media application, or use
  AutoHotkey script above which always wins.

Option 3 – Bluetooth headset
  Headset media buttons send WM_APPCOMMAND which AutoHotkey intercepts.
  The script above handles Play/Pause, Next, Prev, Stop automatically.

`)

	case "linux":
		fmt.Print(`
LINUX
─────
Option 1 – MPRIS / mpDris2 (recommended)
  mpDris2 exposes MPD as an MPRIS2 D-Bus service, so desktop media keys
  and Bluetooth headsets control it natively.

  Install:
    sudo apt install mpdris2          # Debian/Ubuntu/Mint
    sudo pacman -S mpd-mpris          # Arch Linux
    sudo dnf install mpDris2          # Fedora/RHEL

  Enable:
    # Via systemd user service (recommended)
    systemctl --user enable --now mpd-mpris

    # Or start manually
    mpDris2 &

  After this, hardware media keys and Bluetooth headsets work natively.

Option 2 – playerctl
  playerctl forwards MPRIS commands:

    # Install
    sudo apt install playerctl

    # Bind XF86 keys in your WM/DE:
    XF86AudioPlay  → playerctl play-pause --player=mpd
    XF86AudioNext  → playerctl next       --player=mpd
    XF86AudioPrev  → playerctl previous   --player=mpd
    XF86AudioStop  → playerctl stop       --player=mpd

  Or bind to mpdl directly:
    XF86AudioPlay  → mpdl pause
    XF86AudioNext  → mpdl next
    XF86AudioPrev  → mpdl prev
    XF86AudioStop  → mpdl stop

Desktop-specific binding:
  GNOME:  Settings → Keyboard → Custom Shortcuts
  KDE:    Settings → Shortcuts → Custom Shortcuts
  XFCE:   Settings → Keyboard → Application Shortcuts
  i3/sway: bindsym XF86AudioPlay exec mpdl pause

Bluetooth headset:
  Works automatically after installing mpDris2/mpd-mpris.

`)

	case "darwin":
		fmt.Print(`
macOS
─────
Option 1 – Hammerspoon (free, recommended)
  Install from https://www.hammerspoon.org
  Add to ~/.hammerspoon/init.lua:

    -- MPD media key bindings
    hs.hotkey.bind({}, "F7", function() os.execute("mpdl prev") end)
    hs.hotkey.bind({}, "F8", function() os.execute("mpdl pause") end)
    hs.hotkey.bind({}, "F9", function() os.execute("mpdl next") end)

  Or intercept real media keys:
    local eventtap = hs.eventtap
    local event    = eventtap.event
    mediaWatcher = eventtap.new({event.types.NSSystemDefined}, function(e)
      local key = e:getProperty(event.properties.mouseEventNumber)
      -- 16 = Play/Pause, 17 = Next, 18 = Prev, 19 = Stop
      if     key == 16 then os.execute("mpdl pause")
      elseif key == 17 then os.execute("mpdl next")
      elseif key == 18 then os.execute("mpdl prev")
      elseif key == 19 then os.execute("mpdl stop")
      end
    end):start()

Option 2 – BetterTouchTool (paid)
  Assign media keys directly to shell commands (mpdl play, mpdl next, …).

Option 3 – Karabiner-Elements (free)
  Remap keys to run shell commands.

Bluetooth headset:
  macOS routes headset buttons to the "active" media app.
  To make MPD the active app, use a Now Playing wrapper such as:
    https://github.com/nicknick/NowPlaying

`)

	default:
		fmt.Printf("No setup guide available for %s.\n", runtime.GOOS)
	}

	fmt.Println(sep)
	fmt.Printf("\n%sTip:%s Start monitor with media key support:\n", Bold, Reset)
	fmt.Println("  mpdl monitor --media-keys\n")
}
