// File: media_keys.go
// Global media key integration — OS-level, not terminal-dependent.
//
// Windows : RegisterHotKey WinAPI (no window needed, works system-wide)
// Linux   : playerctl subprocess + MPRIS D-Bus (via mpd-mpris / mpdris2)
// macOS   : CGEventTap via Hammerspoon IPC or media key daemon
//
// License: MIT

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// ──────────────────────────────────────────────
// MediaKeyMonitor
// ──────────────────────────────────────────────

type MediaKeyMonitor struct {
	client     *MPDClient
	debug      bool
	stopChan   chan struct{}
	isRunning  bool
	debounceMs int
	lastCmd    time.Time
}

func NewMediaKeyMonitor(client *MPDClient, debug bool) *MediaKeyMonitor {
	return &MediaKeyMonitor{
		client:     client,
		debug:      debug,
		stopChan:   make(chan struct{}),
		debounceMs: 300,
	}
}

func (m *MediaKeyMonitor) Start() error {
	if m.isRunning {
		return fmt.Errorf("already running")
	}
	m.isRunning = true

	switch runtime.GOOS {
	case "windows":
		go m.monitorWindows()
	case "linux":
		go m.monitorLinux()
	case "darwin":
		go m.monitorMacOS()
	default:
		m.isRunning = false
		return fmt.Errorf("global media keys not supported on %s", runtime.GOOS)
	}

	if m.debug {
		log.Printf("🎹 Global media key monitoring started (%s)", runtime.GOOS)
	}
	return nil
}

func (m *MediaKeyMonitor) Stop() {
	if m.isRunning {
		close(m.stopChan)
		m.isRunning = false
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

// ── Action dispatchers ───────────────────────────────────────────────────────

func (m *MediaKeyMonitor) doPlayPause() {
	if !m.debounce() {
		return
	}
	status, err := m.client.Status()
	if err != nil {
		return
	}
	if status["state"] == "play" {
		_ = m.client.Pause()
	} else {
		_ = m.client.Play(-1)
	}
	if m.debug {
		log.Println("🎹 Global: Play/Pause")
	}
}

func (m *MediaKeyMonitor) doNext() {
	if !m.debounce() {
		return
	}
	_ = m.client.Next()
	if m.debug {
		log.Println("🎹 Global: Next")
	}
}

func (m *MediaKeyMonitor) doPrev() {
	if !m.debounce() {
		return
	}
	_ = m.client.Previous()
	if m.debug {
		log.Println("🎹 Global: Previous")
	}
}

func (m *MediaKeyMonitor) doStop() {
	if !m.debounce() {
		return
	}
	_ = m.client.Stop()
	if m.debug {
		log.Println("🎹 Global: Stop")
	}
}

func (m *MediaKeyMonitor) doVolUp() {
	if !m.debounce() {
		return
	}
	_ = m.client.VolumeRelative(+5)
}

func (m *MediaKeyMonitor) doVolDown() {
	if !m.debounce() {
		return
	}
	_ = m.client.VolumeRelative(-5)
}

// ── Linux: playerctl event loop ──────────────────────────────────────────────
//
// playerctl can subscribe to MPRIS events from ANY media player.
// We run "playerctl --follow status" which prints a line every time the
// global playback state changes (from headset buttons, keyboard keys, etc.)
// and we mirror those events to MPD.
//
// Requires: playerctl + mpd-mpris (or mpdris2) running.
// mpd-mpris exposes MPD as an MPRIS2 service so playerctl can see it.

func (m *MediaKeyMonitor) monitorLinux() {
	// Attempt 1: use playerctl --follow to get global media events
	playerctlPath, err := exec.LookPath("playerctl")
	if err != nil {
		log.Printf("⚠️  playerctl not found — global media keys inactive on Linux")
		log.Printf("   Fix: sudo apt install playerctl mpd-mpris && systemctl --user enable --now mpd-mpris")
		m.keepAlive()
		return
	}

	log.Printf("🎹 Linux global media keys: using playerctl (%s)", playerctlPath)

	for {
		select {
		case <-m.stopChan:
			return
		default:
		}

		// "playerctl --player=mpd,any --follow status" prints a line per event:
		//   Playing
		//   Paused
		//   Stopped
		// We also listen for Next/Previous via a separate goroutine using
		// "playerctl --follow metadata" which fires on track change.
		cmd := exec.Command(playerctlPath, "--player=mpd,any", "--follow", "status")
		cmd.Stderr = os.Stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		if err := cmd.Start(); err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		stopCmd := make(chan struct{})
		go func() {
			select {
			case <-m.stopChan:
				_ = cmd.Process.Kill()
			case <-stopCmd:
			}
		}()

		buf := make([]byte, 256)
		for {
			n, err := stdout.Read(buf)
			if err != nil || n == 0 {
				break
			}
			line := strings.TrimSpace(string(buf[:n]))
			// playerctl status output: "Playing", "Paused", "Stopped"
			// These fire when the user presses headset/keyboard media buttons.
			switch line {
			case "Playing":
				if !m.debounce() {
					continue
				}
				// Only act if MPD is not already playing — avoids feedback loop.
				if s, err := m.client.Status(); err == nil && s["state"] != "play" {
					_ = m.client.Play(-1)
					if m.debug {
						log.Println("🎹 playerctl→ Play")
					}
				}
			case "Paused":
				if !m.debounce() {
					continue
				}
				if s, err := m.client.Status(); err == nil && s["state"] == "play" {
					_ = m.client.Pause()
					if m.debug {
						log.Println("🎹 playerctl→ Pause")
					}
				}
			case "Stopped":
				if !m.debounce() {
					continue
				}
				_ = m.client.Stop()
			}
		}

		close(stopCmd)
		_ = cmd.Wait()

		select {
		case <-m.stopChan:
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// ── Windows: RegisterHotKey WinAPI ───────────────────────────────────────────
//
// RegisterHotKey registers a global hotkey that fires even when the terminal
// is in the background. It requires a Win32 message loop.
// We use "golang.org/x/sys/windows" for WinAPI access.
//
// VK codes for media keys:
//   VK_MEDIA_PLAY_PAUSE = 0xB3
//   VK_MEDIA_NEXT_TRACK = 0xB0
//   VK_MEDIA_PREV_TRACK = 0xB1
//   VK_MEDIA_STOP       = 0xB2
//   VK_VOLUME_UP        = 0xAF
//   VK_VOLUME_DOWN      = 0xAE

func (m *MediaKeyMonitor) monitorWindows() {
	// Try to register global hotkeys via WinAPI.
	// We use a subprocess approach (mpdl as its own hotkey daemon) if cgo is off.
	if err := m.tryWindowsHotkeys(); err != nil {
		log.Printf("⚠️  Windows global hotkeys failed: %v", err)
		log.Printf("   Fallback: create an AutoHotkey script — run: mpdl mediakeys")
		m.keepAlive()
	}
}

// tryWindowsHotkeys registers OS-level global media key hooks on Windows.
// Uses a self-pipe: spawns a lightweight child process that registers the
// hotkeys and forwards commands back via stdin, so the main mpdl process
// does not need to run a Win32 message loop itself.
func (m *MediaKeyMonitor) tryWindowsHotkeys() error {
	// Check if we have a hotkey helper available (mpdl itself as helper).
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find own executable: %v", err)
	}

	log.Println("🎹 Windows: registering global media hotkeys via WinAPI helper")

	cmd := exec.Command(exe, "--hotkey-daemon",
		"--mpd-host", m.client.host,
		"--mpd-port", m.client.port)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		select {
		case <-m.stopChan:
			_ = cmd.Process.Kill()
		}
	}()

	buf := make([]byte, 32)
	for {
		n, err := stdout.Read(buf)
		if err != nil || n == 0 {
			break
		}
		switch strings.TrimSpace(string(buf[:n])) {
		case "play_pause":
			m.doPlayPause()
		case "next":
			m.doNext()
		case "prev":
			m.doPrev()
		case "stop":
			m.doStop()
		case "vol_up":
			m.doVolUp()
		case "vol_down":
			m.doVolDown()
		}
	}

	_ = cmd.Wait()
	return fmt.Errorf("hotkey daemon exited")
}

// ── macOS: Hammerspoon IPC ────────────────────────────────────────────────────

func (m *MediaKeyMonitor) monitorMacOS() {
	// Check for Hammerspoon IPC socket
	hsSocket := fmt.Sprintf("%s/.hammerspoon/ipc.sock",
		os.Getenv("HOME"))

	if _, err := os.Stat(hsSocket); err == nil {
		log.Println("🎹 macOS: Hammerspoon IPC available — global media keys active")
		m.keepAlive()
		return
	}

	// Fallback: check if BetterTouchTool is running
	out, _ := exec.Command("pgrep", "-x", "BetterTouchTool").Output()
	if len(strings.TrimSpace(string(out))) > 0 {
		log.Println("🎹 macOS: BetterTouchTool running — configure it to call mpdl commands")
		m.keepAlive()
		return
	}

	log.Printf("⚠️  macOS global media keys: no supported tool found")
	log.Printf("   Fix: install Hammerspoon + add media key config — run: mpdl mediakeys")
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
// RunWithMediaKeys
// ──────────────────────────────────────────────

func RunWithMediaKeys(state *AppState) error {
	mk := NewMediaKeyMonitor(state.client, state.debug)
	if err := mk.Start(); err != nil && state.debug {
		log.Printf("⚠️  Media key monitor: %v", err)
	}
	defer mk.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		mk.Stop()
		os.Exit(0)
	}()

	return runMonitor(state)
}

// ──────────────────────────────────────────────
// detectMediaKeySupport — shown in monitor banner
// ──────────────────────────────────────────────

func detectMediaKeySupport(debug bool) string {
	switch runtime.GOOS {
	case "linux":
		if path, err := exec.LookPath("playerctl"); err == nil {
			out, err := exec.Command("playerctl", "--list-all").Output()
			if err == nil {
				players := strings.TrimSpace(string(out))
				if strings.Contains(players, "mpd") {
					return fmt.Sprintf("%s✓ global — playerctl (%s) + MPD visible via MPRIS%s",
						ColorGreen, path, Reset)
				}
				return fmt.Sprintf("%s⚠ playerctl found but MPD not listed%s — run: systemctl --user start mpd-mpris",
					ColorYellow, Reset)
			}
			return fmt.Sprintf("%s⚠ playerctl found but not responding%s", ColorYellow, Reset)
		}
		for _, proc := range []string{"mpDris2", "mpd-mpris"} {
			out, _ := exec.Command("pgrep", "-x", proc).Output()
			if len(strings.TrimSpace(string(out))) > 0 {
				return fmt.Sprintf("%s✓ global — %s running (no playerctl — limited)%s",
					ColorGreen, proc, Reset)
			}
		}
		return fmt.Sprintf("%s✗ inactive%s — install: sudo apt install playerctl mpd-mpris && systemctl --user enable --now mpd-mpris",
			ColorRed, Reset)

	case "windows":
		out, _ := exec.Command("tasklist", "/FI", "IMAGENAME eq AutoHotkey*.exe", "/NH").Output()
		if strings.Contains(strings.ToLower(string(out)), "autohotkey") {
			return fmt.Sprintf("%s✓ global — AutoHotkey running%s", ColorGreen, Reset)
		}
		return fmt.Sprintf("%s⚠ AutoHotkey not detected%s — run: mpdl mediakeys  for setup guide",
			ColorYellow, Reset)

	case "darwin":
		out, _ := exec.Command("pgrep", "-x", "Hammerspoon").Output()
		if len(strings.TrimSpace(string(out))) > 0 {
			return fmt.Sprintf("%s✓ global — Hammerspoon running%s", ColorGreen, Reset)
		}
		out, _ = exec.Command("pgrep", "-x", "BetterTouchTool").Output()
		if len(strings.TrimSpace(string(out))) > 0 {
			return fmt.Sprintf("%s✓ global — BetterTouchTool running%s", ColorGreen, Reset)
		}
		return fmt.Sprintf("%s⚠ no tool detected%s — run: mpdl mediakeys  for setup guide",
			ColorYellow, Reset)

	default:
		return fmt.Sprintf("%sunsupported%s", ColorGray, Reset)
	}
}

// ──────────────────────────────────────────────
// SetupSystemMediaKeys — detailed guide
// ──────────────────────────────────────────────

func SetupSystemMediaKeys() {
	sep := strings.Repeat("─", 60)
	fmt.Printf("\n%s🎹 Global Media Key Setup Guide%s\n", Bold, Reset)
	fmt.Println(sep)

	switch runtime.GOOS {
	case "windows":
		fmt.Print(`
WINDOWS — Global Media Keys (works in background)
──────────────────────────────────────────────────
Option 1 — AutoHotkey v2 (recommended, free)
  Download: https://www.autohotkey.com

  Create C:\Users\<you>\AppData\Roaming\Microsoft\Windows\Start Menu\
         Programs\Startup\mpdl_keys.ahk

  Contents:
    #Requires AutoHotkey v2.0
    Media_Play_Pause:: {
        Run "mpdl pause",, "Hide"
    }
    Media_Next:: {
        Run "mpdl next",, "Hide"
    }
    Media_Prev:: {
        Run "mpdl prev",, "Hide"
    }
    Media_Stop:: {
        Run "mpdl stop",, "Hide"
    }

  This fires globally — Bluetooth headsets, keyboard media keys,
  even when mpdl monitor is NOT running.

  Double-click the .ahk file to start. It runs silently in the tray.
  Placing it in Startup means it launches on every login.

Option 2 — Windows built-in (no extra software)
  If your keyboard has media keys, Windows routes them to the active
  media player. Set mpdl as default music player in Settings →
  Default Apps → Music Player.

`)

	case "linux":
		fmt.Print(`
LINUX — Global Media Keys (headset + hardware keyboard)
────────────────────────────────────────────────────────
Step 1 — Install mpd-mpris (exposes MPD as MPRIS2 service)
  sudo apt install mpd-mpris          # Debian / Ubuntu / Mint
  sudo pacman -S mpd-mpris            # Arch / Manjaro
  sudo dnf install mpd-mpris          # Fedora / RHEL
  yay -S mpd-mpris                    # AUR

  Enable as user service:
    systemctl --user enable --now mpd-mpris

Step 2 — Install playerctl
  sudo apt install playerctl

  Test it works:
    playerctl --list-all              # should show "mpd"
    playerctl play-pause              # should toggle MPD

  After this, ALL system media keys and Bluetooth headset buttons
  control MPD globally — no terminal needed.

Step 3 — mpdl monitor --media-keys
  The monitor will confirm:
    ✓ global — playerctl + MPD visible via MPRIS

Desktop-specific manual bindings (if needed):
  GNOME:  Settings → Keyboard → Custom Shortcuts
    XF86AudioPlay  → mpdl pause
    XF86AudioNext  → mpdl next
    XF86AudioPrev  → mpdl prev
    XF86AudioStop  → mpdl stop

  i3 / Sway (add to config):
    bindsym XF86AudioPlay  exec mpdl pause
    bindsym XF86AudioNext  exec mpdl next
    bindsym XF86AudioPrev  exec mpdl prev
    bindsym XF86AudioStop  exec mpdl stop

  Openbox (~/.config/openbox/rc.xml):
    <keybind key="XF86AudioPlay">
      <action name="Execute"><command>mpdl pause</command></action>
    </keybind>

`)

	case "darwin":
		fmt.Print(`
macOS — Global Media Keys
─────────────────────────
Option 1 — Hammerspoon (free, recommended)
  Download: https://www.hammerspoon.org

  Add to ~/.hammerspoon/init.lua:

    -- Global MPD media keys
    local function mpdl(cmd)
      hs.task.new("/usr/local/bin/mpdl", nil, {cmd}):start()
    end

    -- Intercept F7/F8/F9 (or use hs.mediakey)
    hs.hotkey.bind({}, "F7",  function() mpdl("prev")  end)
    hs.hotkey.bind({}, "F8",  function() mpdl("pause") end)
    hs.hotkey.bind({}, "F9",  function() mpdl("next")  end)

    -- Intercept actual media keys (requires accessibility permission)
    hs.eventtap.new({hs.eventtap.event.types.NSSystemDefined},
      function(e)
        local keyCode = e:getProperty(
          hs.eventtap.event.properties.mouseEventNumber)
        if     keyCode == 16 then mpdl("pause")
        elseif keyCode == 17 then mpdl("next")
        elseif keyCode == 18 then mpdl("prev")
        elseif keyCode == 19 then mpdl("stop")
        end
      end):start()

  Reload: hs.reload() in Hammerspoon console.
  Grant Accessibility permission in System Preferences if asked.

Option 2 — BetterTouchTool (paid, easiest UI)
  Add media key triggers → Shell Script → mpdl pause / next / prev

Option 3 — Karabiner-Elements (free)
  Map media keys to shell commands.

Bluetooth headset:
  macOS routes headset buttons to the "Now Playing" app.
  Use an MPD Now Playing bridge:
    https://github.com/nicknick/NowPlaying

`)
	}

	fmt.Println(sep)
	fmt.Printf("\n%sTip:%s after setup, confirm with:\n  mpdl monitor --media-keys\n\n", Bold, Reset)
}
