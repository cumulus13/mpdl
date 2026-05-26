// File: main.go
// Project: mpdl - Advanced MPD CLI client
// Author: Hadi Cahyadi <cumulus13@gmail.com>
// Date: 2026-02-04
// Description: Production-ready MPD CLI with monitoring, playlist management,
//              MPC-compatible commands, filter/regex/wildcard support, and media keys
// License: MIT

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cumulus13/go-gntp"
	"github.com/fhs/gompd/v2/mpd"
	"golang.org/x/term"
)

// Version is injected at build time via -ldflags
var Version = "1.0.0"

// ──────────────────────────────────────────────
// ANSI colour palette
// ──────────────────────────────────────────────
const (
	Reset        = "\033[0m"
	BgBlue       = "\033[44m"
	FgIndex      = "\033[33m"
	FgTitle      = "\033[93m"
	FgArtist     = "\033[96m"
	FgAlbum      = "\033[105m"
	FgDate       = "\033[92m"
	PlayingBg    = "\033[48;5;17m"
	PlayingLabel = "\033[97;41m"
	ColorCyan    = "\033[96m"
	ColorYellow  = "\033[93m"
	ColorOrange  = "\033[38;5;216m"
	ColorBlue    = "\033[94m"
	ColorGreen   = "\033[92m"
	ColorRed     = "\033[91m"
	ColorGray    = "\033[90m"
	ColorWhite   = "\033[97m"
	Bold         = "\033[1m"
	Icon         = "🎵"
)

// ──────────────────────────────────────────────
// Config
// ──────────────────────────────────────────────

// Config represents the full application configuration.
type Config struct {
	MPD struct {
		Host       string `toml:"host"        json:"host"        yaml:"host"`
		Port       string `toml:"port"        json:"port"        yaml:"port"`
		Password   string `toml:"password"    json:"password"    yaml:"password"`
		Timeout    int    `toml:"timeout"     json:"timeout"     yaml:"timeout"`
		MusicRoot  string `toml:"music_root"  json:"music_root"  yaml:"music_root"`
		ConfigPath string `toml:"config_path" json:"config_path" yaml:"config_path"`
	} `toml:"mpd" json:"mpd" yaml:"mpd"`

	GNTP struct {
		Host     string `toml:"host"      json:"host"      yaml:"host"`
		Port     int    `toml:"port"      json:"port"      yaml:"port"`
		Password string `toml:"password"  json:"password"  yaml:"password"`
		IconMode string `toml:"icon_mode" json:"icon_mode" yaml:"icon_mode"`
		Enabled  bool   `toml:"enabled"   json:"enabled"   yaml:"enabled"`
	} `toml:"gntp" json:"gntp" yaml:"gntp"`

	Display struct {
		ShowAlbumArt bool `toml:"show_album_art"   json:"show_album_art"   yaml:"show_album_art"`
		UseColor     bool `toml:"use_color"        json:"use_color"        yaml:"use_color"`
		ShowProgress bool `toml:"show_progress"    json:"show_progress"    yaml:"show_progress"`
	} `toml:"display" json:"display" yaml:"display"`
}

// NewConfig returns a Config populated with sensible defaults.
func NewConfig() *Config {
	cfg := &Config{}
	cfg.MPD.Host = "localhost"
	cfg.MPD.Port = "6600"
	cfg.MPD.Timeout = 10
	cfg.MPD.MusicRoot = getMusicRootDefault()
	cfg.MPD.ConfigPath = getMPDConfigDefault()
	cfg.GNTP.Host = "localhost"
	cfg.GNTP.Port = 23053
	cfg.GNTP.IconMode = "binary"
	cfg.GNTP.Enabled = true
	cfg.Display.ShowAlbumArt = true
	cfg.Display.UseColor = true
	cfg.Display.ShowProgress = true // progress bar in monitor mode on by default
	return cfg
}

func getMusicRootDefault() string {
	switch runtime.GOOS {
	case "windows":
		return "C:/Musics"
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Music")
	}
}

func getMPDConfigDefault() string {
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "mpd", "mpd.conf")
		}
		return "C:/mpd/mpd.conf"
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "mpd", "mpd.conf")
	}
}

// LoadConfig loads the config file (multi-format) then overlays env vars.
func LoadConfig(configPath string) (*Config, error) {
	cfg := NewConfig()
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := LoadConfigFromFile(configPath, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config: %v", err)
			}
		}
	}
	// Environment variable overrides
	if v := os.Getenv("MPD_HOST"); v != "" {
		cfg.MPD.Host = v
	}
	if v := os.Getenv("MPD_PORT"); v != "" {
		cfg.MPD.Port = v
	}
	if v := os.Getenv("MPD_PASSWORD"); v != "" {
		cfg.MPD.Password = v
	}
	if v := os.Getenv("MPD_TIMEOUT"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.MPD.Timeout = t
		}
	}
	if v := os.Getenv("MPD_MUSIC_ROOT"); v != "" {
		cfg.MPD.MusicRoot = v
	}
	return cfg, nil
}

// ──────────────────────────────────────────────
// MPDClient
// ──────────────────────────────────────────────

// MPDClient wraps gompd with auto-reconnect logic.
type MPDClient struct {
	host     string
	port     string
	password string
	client   *mpd.Client
	config   *Config
}

func NewMPDClient(host, port, password string, cfg *Config) (*MPDClient, error) {
	m := &MPDClient{host: host, port: port, password: password, config: cfg}
	if err := m.connect(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *MPDClient) connect() error {
	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	c, err := mpd.DialAuthenticated("tcp", addr, m.password)
	if err != nil {
		return fmt.Errorf("cannot connect to MPD at %s: %v", addr, err)
	}
	m.client = c
	return nil
}

func (m *MPDClient) reconnect() error {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
	for i := 0; i < 5; i++ {
		if err := m.connect(); err == nil {
			if err := m.client.Ping(); err == nil {
				return nil
			}
			m.client.Close()
			m.client = nil
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return fmt.Errorf("failed to reconnect after 5 attempts")
}

func (m *MPDClient) ensureConnected() error {
	if m.client == nil {
		return m.connect()
	}
	if err := m.client.Ping(); err != nil {
		return m.reconnect()
	}
	return nil
}

func (m *MPDClient) Close() {
	if m.client != nil {
		m.client.Close()
	}
}

// normalizePath strips the music root prefix so MPD gets a relative path.
func (m *MPDClient) normalizePath(path string) string {
	if path == "" {
		return ""
	}
	path = filepath.ToSlash(path)
	musicRoot := strings.TrimSuffix(filepath.ToSlash(m.config.MPD.MusicRoot), "/")

	pLow := strings.ToLower(path)
	mLow := strings.ToLower(musicRoot)

	if strings.HasPrefix(pLow, mLow+"/") {
		path = path[len(musicRoot)+1:]
	} else if pLow == mLow {
		path = ""
	} else if idx := strings.Index(pLow, mLow); idx >= 0 {
		path = strings.TrimPrefix(path[idx+len(musicRoot):], "/")
	}

	// Strip Windows drive letter
	if len(path) >= 2 && path[1] == ':' {
		path = path[2:]
	}
	path = strings.TrimPrefix(path, "/")
	path = strings.ReplaceAll(path, "\\", "/")
	return path
}

// ──────────────────────────────────────────────
// Playback
// ──────────────────────────────────────────────

func (m *MPDClient) Play(pos int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	if pos >= 0 {
		return m.client.Play(pos)
	}
	return m.client.Play(-1)
}

func (m *MPDClient) Pause() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	status, err := m.client.Status()
	if err != nil {
		return err
	}
	return m.client.Pause(status["state"] != "pause")
}

func (m *MPDClient) Stop() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Stop()
}

func (m *MPDClient) Next() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Next()
}

func (m *MPDClient) Previous() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Previous()
}

// Seek seeks within the current song. seconds may be negative for rewind.
func (m *MPDClient) Seek(seconds float64, relative bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	d := time.Duration(seconds * float64(time.Second))
	return m.client.SeekCur(d, relative)
}

// ──────────────────────────────────────────────
// Playlist mutation
// ──────────────────────────────────────────────

func (m *MPDClient) Add(path string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Add(m.normalizePath(path))
}

// queueEntry holds the data needed to sort and reorder the MPD queue.
// Defined at package level so sortQueueEntries and reorderQueue can reference it.
type queueEntry struct {
	id    int // MPD song ID (stable across Move operations)
	disc  int
	track int
	file  string // used as tie-breaker
}

// AddAndPlay clears the queue, adds path (file or folder), sorts the resulting
// queue by Disc→Track tag order, then plays from position 0.
//
// Why sorting is necessary
// ────────────────────────
// MPD's "add <dir>" appends files in filesystem/directory-walk order, which is
// almost always alphabetical by filename — not by the Track/Disc metadata tags.
// A folder like:
//
//	02 - Song Two.flac        ← added first  (pos 0)
//	01 - Song One.flac        ← added second (pos 1)
//
// would start at track 2 without this reorder step.
func (m *MPDClient) AddAndPlay(path string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}

	// Step 1 – clear queue and add the requested path atomically.
	cl := m.client.BeginCommandList()
	cl.Clear()
	cl.Add(m.normalizePath(path))
	if err := cl.End(); err != nil {
		return fmt.Errorf("addplay clear+add: %v", err)
	}

	// Step 2 – read back what was just added.
	playlist, err := m.client.PlaylistInfo(-1, -1)
	if err != nil {
		return fmt.Errorf("addplay playlist read: %v", err)
	}
	if len(playlist) == 0 {
		return fmt.Errorf("addplay: no tracks found at %q", path)
	}

	// Step 3 – build a sortable slice keyed by MPD song ID.
	entries := make([]queueEntry, 0, len(playlist))
	for _, song := range playlist {
		id, _ := strconv.Atoi(song["Id"])
		entries = append(entries, queueEntry{
			id:    id,
			disc:  parseTrackNum(song["Disc"]),
			track: parseTrackNum(song["Track"]),
			file:  song["file"],
		})
	}

	// Step 4 – sort: Disc asc → Track asc → filename asc.
	sortQueueEntries(entries)

	// Step 5 – reorder the queue in MPD to match the sorted order.
	// Non-fatal: if reordering fails we still play from position 0,
	// which may not be track #1 but is better than an error exit.
	if err := m.reorderQueue(entries); err != nil {
		log.Printf("⚠️  addplay: reorder failed (playing anyway): %v", err)
	}

	// Step 6 – play from the first position (Disc 1, Track 1).
	return m.client.Play(0)
}

// parseTrackNum parses MPD tag strings like "1", "01", "1/12" → 1.
// Returns 0 on empty/invalid input so untagged files sort before tagged ones.
func parseTrackNum(s string) int {
	if s == "" {
		return 0
	}
	// Strip "/total" suffix: "3/12" → "3"
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// sortQueueEntries sorts in place: Disc asc, Track asc, file asc.
// Uses insertion sort — correct and allocation-free; fast for ≤ a few thousand tracks.
func sortQueueEntries(e []queueEntry) {
	for i := 1; i < len(e); i++ {
		for j := i; j > 0 && queueLess(e[j], e[j-1]); j-- {
			e[j], e[j-1] = e[j-1], e[j]
		}
	}
}

// queueLess returns true when a should sort before b.
func queueLess(a, b queueEntry) bool {
	if a.disc != b.disc {
		return a.disc < b.disc
	}
	if a.track != b.track {
		return a.track < b.track
	}
	return a.file < b.file
}

// reorderQueue issues Move commands to bring the MPD queue into the order
// described by entries.  It maintains a local posOf map so it can correctly
// account for how each Move shifts the positions of neighbouring songs —
// avoiding a full PlaylistInfo round-trip after every Move.
// reorderQueue issues MoveID commands to bring the MPD queue into the order
// described by entries (sorted by disc/track).
//
// We use MoveID (move by stable song ID) rather than Move (move by position)
// because MoveID's target is always the final position — MPD re-indexes
// everything after each call, so we never need to track position drift ourselves.
func (m *MPDClient) reorderQueue(entries []queueEntry) error {
	// Issue one MoveID per song. MPD handles all position re-indexing internally.
	// We iterate in order 0, 1, 2, … so each song is placed at its correct
	// final position before we move on to the next.
	for targetPos, e := range entries {
		if err := m.client.MoveID(e.id, targetPos); err != nil {
			return fmt.Errorf("moveid id=%d to pos %d: %v", e.id, targetPos, err)
		}
	}
	return nil
}

// Insert adds song(s) right after the currently playing position.
func (m *MPDClient) Insert(path string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	status, err := m.client.Status()
	if err != nil {
		return err
	}
	insertPos := 0
	if s, ok := status["song"]; ok {
		if n, err := strconv.Atoi(s); err == nil {
			insertPos = n + 1
		}
	}

	// AddID returns the song ID of the newly added song, so we can MoveID it
	// to the right position without needing a full PlaylistInfo round-trip.
	id, err := m.client.AddID(m.normalizePath(path), -1)
	if err != nil {
		return err
	}
	// MoveID(songid, targetPos) — 2 args, moves by stable ID.
	return m.client.MoveID(id, insertPos)
}

// Delete removes playlist entries by position number (1-based), range, or path pattern.
// Supported formats:
//   - "3"        – delete position 3
//   - "3-7"      – delete positions 3 through 7
//   - "0"        – delete currently playing song (mpc compat)
//   - "/regex/"  – delete all matching title/artist/file
//   - "glob*"    – delete all entries whose file/title match the glob
//   - anything else treated as substring match against file path
func (m *MPDClient) Delete(arg string) (int, error) {
	if err := m.ensureConnected(); err != nil {
		return 0, err
	}

	// ── by numeric position ──────────────────────────────────────────
	if pos, err := strconv.Atoi(arg); err == nil {
		// mpc compat: 0 = currently playing
		if pos == 0 {
			status, err := m.client.Status()
			if err != nil {
				return 0, err
			}
			if s, ok := status["song"]; ok {
				if n, err := strconv.Atoi(s); err == nil {
					pos = n + 1
				}
			}
		}
		if pos < 1 {
			return 0, fmt.Errorf("invalid position: %s", arg)
		}
		// gompd Delete takes 0-based start,end
		if err := m.client.Delete(pos-1, pos); err != nil {
			return 0, err
		}
		return 1, nil
	}

	// ── by range  "from-to"  (1-based inclusive) ────────────────────
	if parts := strings.SplitN(arg, "-", 2); len(parts) == 2 {
		from, err1 := strconv.Atoi(parts[0])
		to, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil && from >= 1 && to >= from {
			count := to - from + 1
			if err := m.client.Delete(from-1, to); err != nil {
				return 0, err
			}
			return count, nil
		}
	}

	// ── by pattern (regex / glob / substring) ───────────────────────
	return m.deleteByPattern(arg)
}

func (m *MPDClient) deleteByPattern(pattern string) (int, error) {
	playlist, err := m.client.PlaylistInfo(-1, -1)
	if err != nil {
		return 0, err
	}

	// Build matcher
	var matcher func(song mpd.Attrs) bool

	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) >= 2 {
		// /regex/ literal
		inner := pattern[1 : len(pattern)-1]
		re, err := regexp.Compile("(?i)" + inner)
		if err != nil {
			return 0, fmt.Errorf("invalid regex %q: %v", inner, err)
		}
		matcher = func(song mpd.Attrs) bool {
			for _, field := range []string{"Title", "Artist", "Album", "file"} {
				if re.MatchString(song[field]) {
					return true
				}
			}
			return false
		}
	} else if strings.ContainsAny(pattern, "*?[") {
		// glob wildcard
		pat := strings.ToLower(pattern)
		matcher = func(song mpd.Attrs) bool {
			for _, field := range []string{"Title", "Artist", "Album", "file"} {
				if ok, _ := filepath.Match(pat, strings.ToLower(song[field])); ok {
					return true
				}
				// also match against base of file path
				if ok, _ := filepath.Match(pat, strings.ToLower(filepath.Base(song[field]))); ok {
					return true
				}
			}
			return false
		}
	} else {
		// substring / normalised-path match (original behaviour)
		norm := strings.ToLower(m.normalizePath(pattern))
		matcher = func(song mpd.Attrs) bool {
			songNorm := strings.ToLower(m.normalizePath(song["file"]))
			return songNorm == norm ||
				strings.ToLower(filepath.Dir(songNorm)) == norm ||
				strings.Contains(songNorm, norm) ||
				strings.Contains(strings.ToLower(song["Title"]), norm) ||
				strings.Contains(strings.ToLower(song["Artist"]), norm)
		}
	}

	// Collect IDs to delete (work by ID, safe from re-indexing)
	var ids []int
	for _, song := range playlist {
		if matcher(song) {
			if id, err := strconv.Atoi(song["Id"]); err == nil {
				ids = append(ids, id)
			}
		}
	}
	for _, id := range ids {
		_ = m.client.DeleteID(id)
	}
	return len(ids), nil
}

func (m *MPDClient) Clear() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Clear()
}

func (m *MPDClient) Shuffle() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Shuffle(-1, -1)
}

// Move moves a single song from position from to position to (both 0-based).
// gompd's Move(start, end, pos) takes a range [start,end), so we pass start+1
// as end to move exactly one song.
func (m *MPDClient) Move(from, to int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Move(from, from+1, to)
}

func (m *MPDClient) Crop() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	status, err := m.client.Status()
	if err != nil {
		return err
	}
	currentPos := -1
	if s, ok := status["song"]; ok {
		fmt.Sscanf(s, "%d", &currentPos)
	}
	if currentPos < 0 {
		return fmt.Errorf("no song is currently playing")
	}
	playlist, err := m.client.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	for i := len(playlist) - 1; i >= 0; i-- {
		if i != currentPos {
			_ = m.client.Delete(i, i+1)
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// Volume / Options
// ──────────────────────────────────────────────

func (m *MPDClient) Volume(vol int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.SetVolume(vol)
}

// VolumeRelative adjusts volume by delta (+/-).
func (m *MPDClient) VolumeRelative(delta int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	status, err := m.client.Status()
	if err != nil {
		return err
	}
	cur := 0
	fmt.Sscanf(status["volume"], "%d", &cur)
	newVol := cur + delta
	if newVol < 0 {
		newVol = 0
	}
	if newVol > 100 {
		newVol = 100
	}
	return m.client.SetVolume(newVol)
}

func (m *MPDClient) Random(enable bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Random(enable)
}

func (m *MPDClient) Repeat(enable bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Repeat(enable)
}

func (m *MPDClient) Single(enable bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Single(enable)
}

func (m *MPDClient) Consume(enable bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Consume(enable)
}

// Crossfade sets the crossfade duration in seconds.
// gompd v2 does not expose Crossfade as a typed method; we use the low-level
// Command() helper which sends the raw MPD protocol command.
func (m *MPDClient) Crossfade(seconds int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("crossfade %d", seconds).OK()
}

// ──────────────────────────────────────────────
// Database / Search
// ──────────────────────────────────────────────

func (m *MPDClient) Update(path string) (int, error) {
	if err := m.ensureConnected(); err != nil {
		return 0, err
	}
	return m.client.Update(path)
}

// Rescan forces a full re-read of tags (slower than Update but catches tag edits).
func (m *MPDClient) Rescan(path string) (int, error) {
	if err := m.ensureConnected(); err != nil {
		return 0, err
	}
	return m.client.Rescan(path)
}

// ── Queued song ─────────────────────────────────────────────────────────────

// NextSong returns the next song in the queue (the one that will play after current).
func (m *MPDClient) NextSong() (mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	status, err := m.client.Status()
	if err != nil {
		return nil, err
	}
	nextPos := -1
	if s, ok := status["nextsong"]; ok {
		nextPos, _ = strconv.Atoi(s)
	}
	if nextPos < 0 {
		return mpd.Attrs{}, nil
	}
	playlist, err := m.client.PlaylistInfo(nextPos, nextPos+1)
	if err != nil {
		return nil, err
	}
	if len(playlist) == 0 {
		return mpd.Attrs{}, nil
	}
	return playlist[0], nil
}

// ── ReplayGain ───────────────────────────────────────────────────────────────

// SetReplayGain sets the replay gain mode: off, track, album, auto.
func (m *MPDClient) SetReplayGain(mode string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("replay_gain_mode %s", mode).OK()
}

// GetReplayGain returns the current replay gain mode.
func (m *MPDClient) GetReplayGain() (string, error) {
	if err := m.ensureConnected(); err != nil {
		return "", err
	}
	attrs, err := m.client.Command("replay_gain_status").Attrs()
	if err != nil {
		return "", err
	}
	return attrs["replay_gain_mode"], nil
}

// ── MixRamp ──────────────────────────────────────────────────────────────────

func (m *MPDClient) SetMixRampDB(db float64) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("mixrampdb %f", db).OK()
}

func (m *MPDClient) GetMixRampDB() (string, error) {
	if err := m.ensureConnected(); err != nil {
		return "", err
	}
	s, err := m.client.Status()
	if err != nil {
		return "", err
	}
	return s["mixrampdb"], nil
}

func (m *MPDClient) SetMixRampDelay(seconds float64) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("mixrampdelay %f", seconds).OK()
}

func (m *MPDClient) GetMixRampDelay() (string, error) {
	if err := m.ensureConnected(); err != nil {
		return "", err
	}
	s, err := m.client.Status()
	if err != nil {
		return "", err
	}
	return s["mixrampdelay"], nil
}

// ── SeekThrough ──────────────────────────────────────────────────────────────

// SeekThrough seeks forward/backward by duration across track boundaries.
func (m *MPDClient) SeekThrough(seconds float64) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	// Get current elapsed and song duration
	status, err := m.client.Status()
	if err != nil {
		return err
	}
	elapsed, _ := strconv.ParseFloat(status["elapsed"], 64)
	target := elapsed + seconds

	// Walk forward/backward through playlist
	playlist, err := m.client.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	curPos, _ := strconv.Atoi(status["song"])

	if seconds >= 0 {
		// Forward
		pos := curPos
		remaining := target - elapsed
		for pos < len(playlist) {
			dur, _ := strconv.ParseFloat(playlist[pos]["duration"], 64)
			songElapsed := 0.0
			if pos == curPos {
				songElapsed = elapsed
				dur -= elapsed
			}
			if remaining <= dur {
				if err := m.client.Play(pos); err != nil {
					return err
				}
				seekTo := songElapsed + remaining
				return m.client.SeekCur(time.Duration(seekTo*float64(time.Second)), false)
			}
			remaining -= dur
			pos++
		}
		// Past end — play last song, seek to end
		return m.client.Play(len(playlist) - 1)
	}

	// Backward
	pos := curPos
	remaining := -target // how far back we still need to go
	for pos >= 0 {
		dur, _ := strconv.ParseFloat(playlist[pos]["duration"], 64)
		if pos == curPos {
			dur = elapsed
		}
		if remaining <= dur {
			if err := m.client.Play(pos); err != nil {
				return err
			}
			seekTo := dur - remaining
			return m.client.SeekCur(time.Duration(seekTo*float64(time.Second)), false)
		}
		remaining -= dur
		pos--
	}
	// Past beginning — play track 0 from start
	return m.client.Play(0)
}

// ── Tag listing ──────────────────────────────────────────────────────────────

// ListTags lists all values for a tag type, optionally filtered and grouped.
// func (m *MPDClient) ListTags(tagType string, filter []string, groupBy []string) ([]mpd.Attrs, error) {
// 	if err := m.ensureConnected(); err != nil {
// 		return nil, err
// 	}
// 	args := []string{tagType}
// 	args = append(args, filter...)
// 	for _, g := range groupBy {
// 		args = append(args, "group", g)
// 	}
// 	return m.client.List(tagType, filter...)
// }

// Fix ListTags method (line 890-901)
// Replace the entire method with:
func (m *MPDClient) ListTags(tagType string, filter []string, groupBy []string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	// gompd v2 List returns []string, we need to convert to []mpd.Attrs
	args := []string{tagType}
	args = append(args, filter...)
	for _, g := range groupBy {
		args = append(args, "group", g)
	}
	
	values, err := m.client.List(args...)
	if err != nil {
		return nil, err
	}
	
	// Convert []string to []mpd.Attrs
	var result []mpd.Attrs
	for _, v := range values {
		result = append(result, mpd.Attrs{tagType: v})
	}
	return result, nil
}

// ── Directory listing ────────────────────────────────────────────────────────

// ListDirectory lists files and directories at path.
func (m *MPDClient) ListDirectory(path string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.ListInfo(path)
}

// ListDirs lists only subdirectories at path.
func (m *MPDClient) ListDirs(path string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	all, err := m.client.ListInfo(path)
	if err != nil {
		return nil, err
	}
	var dirs []mpd.Attrs
	for _, item := range all {
		if _, ok := item["directory"]; ok {
			dirs = append(dirs, item)
		}
	}
	return dirs, nil
}

// ── Playlist item management ─────────────────────────────────────────────────

func (m *MPDClient) AddToPlaylist(playlist, uri string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.PlaylistAdd(playlist, uri)
}

func (m *MPDClient) DeleteFromPlaylist(playlist string, pos int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.PlaylistDelete(playlist, pos)
}

func (m *MPDClient) MoveInPlaylist(playlist string, from, to int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.PlaylistMove(playlist, from, to)
}

func (m *MPDClient) RenamePlaylist(from, to string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.PlaylistRename(from, to)
}

func (m *MPDClient) ClearPlaylist(playlist string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	// gompd exposes this as a raw command
	return m.client.Command("playlistclear %s", playlist).OK()
}

// ── Album art / pictures ─────────────────────────────────────────────────────

// AlbumArtBytes returns raw album art bytes for a song URI.
func (m *MPDClient) AlbumArtBytes(uri string) ([]byte, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	data, err := m.client.ReadPicture(uri)
	if err != nil || len(data) == 0 {
		data, err = m.client.AlbumArt(uri)
	}
	return data, err
}

// ── Sticker ──────────────────────────────────────────────────────────────────

func (m *MPDClient) StickerGet(uri, key string) (string, error) {
	if err := m.ensureConnected(); err != nil {
		return "", err
	}
	attrs, err := m.client.Command("sticker get song %s %s", uri, key).Attrs()
	if err != nil {
		return "", err
	}
	return attrs["sticker"], nil
}

func (m *MPDClient) StickerSet(uri, key, value string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("sticker set song %s %s %s", uri, key, value).OK()
}

func (m *MPDClient) StickerDelete(uri, key string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("sticker delete song %s %s", uri, key).OK()
}

func (m *MPDClient) StickerList(uri string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Command("sticker list song %s", uri).AttrsList("sticker")
}

func (m *MPDClient) StickerFind(dir, key string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Command("sticker find song %s %s", dir, key).AttrsList("sticker")
}

// ── Output toggle ────────────────────────────────────────────────────────────

func (m *MPDClient) ToggleOutput(id int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("toggleoutput %d", id).OK()
}

// ── Client-to-client messaging ───────────────────────────────────────────────

func (m *MPDClient) Channels() ([]string, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	attrs, err := m.client.Command("channels").AttrsList("channel")
	if err != nil {
		return nil, err
	}
	var ch []string
	for _, a := range attrs {
		if c, ok := a["channel"]; ok {
			ch = append(ch, c)
		}
	}
	return ch, nil
}

func (m *MPDClient) SendMessage(channel, message string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("sendmessage %s %s", channel, message).OK()
}

// ── Partitions ───────────────────────────────────────────────────────────────

func (m *MPDClient) ListPartitions() ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Command("listpartitions").AttrsList("partition")
}

func (m *MPDClient) NewPartition(name string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("newpartition %s", name).OK()
}

func (m *MPDClient) DeletePartition(name string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("delpartition %s", name).OK()
}

// ── Mounts ───────────────────────────────────────────────────────────────────

func (m *MPDClient) ListMounts() ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Command("listmounts").AttrsList("mount")
}

func (m *MPDClient) Mount(path, uri string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("mount %s %s", path, uri).OK()
}

func (m *MPDClient) Unmount(path string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Command("unmount %s", path).OK()
}

func (m *MPDClient) ListNeighbors() ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Command("listneighbors").AttrsList("neighbor")
}

// ── Idle ─────────────────────────────────────────────────────────────────────

// WaitIdle blocks until one of the given MPD subsystem events fires,
// then returns the list of changed subsystems.
func (m *MPDClient) WaitIdle(subsystems ...string) ([]string, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	w, err := mpd.NewWatcher("tcp",
		fmt.Sprintf("%s:%s", m.host, m.port),
		m.password, subsystems...)
	if err != nil {
		return nil, err
	}
	defer w.Close()
	var changed []string
	select {
	case sub := <-w.Event:
		changed = append(changed, sub)
	case err := <-w.Error:
		return nil, err
	}
	return changed, nil
}

func (m *MPDClient) Status() (mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Status()
}

func (m *MPDClient) CurrentSong() (mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.CurrentSong()
}

func (m *MPDClient) PlaylistInfo() ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.PlaylistInfo(-1, -1)
}

// FindInQueue searches the current queue for tracks matching pattern.
// Pattern formats:
//   /regex/   — compiled regex matched against Title, Artist, Album, file
//   *glob*    — filepath.Match glob matched against same fields
//   text      — case-insensitive substring match
//
// Returns matched songs with their "Pos" field set (0-based queue position).
func (m *MPDClient) FindInQueue(pattern string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	playlist, err := m.client.PlaylistInfo(-1, -1)
	if err != nil {
		return nil, err
	}

	var matcher func(song mpd.Attrs) bool

	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) >= 3 {
		inner := pattern[1 : len(pattern)-1]
		re, err := regexp.Compile("(?i)" + inner)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %v", inner, err)
		}
		matcher = func(song mpd.Attrs) bool {
			for _, f := range []string{"Title", "Artist", "Album", "file"} {
				if re.MatchString(song[f]) {
					return true
				}
			}
			return false
		}
	} else if strings.ContainsAny(pattern, "*?[") {
		pat := strings.ToLower(pattern)
		matcher = func(song mpd.Attrs) bool {
			for _, f := range []string{"Title", "Artist", "Album", "file"} {
				if ok, _ := filepath.Match(pat, strings.ToLower(song[f])); ok {
					return true
				}
				if ok, _ := filepath.Match(pat, strings.ToLower(filepath.Base(song[f]))); ok {
					return true
				}
			}
			return false
		}
	} else {
		sub := strings.ToLower(pattern)
		matcher = func(song mpd.Attrs) bool {
			for _, f := range []string{"Title", "Artist", "Album", "file"} {
				if strings.Contains(strings.ToLower(song[f]), sub) {
					return true
				}
			}
			return false
		}
	}

	var results []mpd.Attrs
	for _, song := range playlist {
		if matcher(song) {
			results = append(results, song)
		}
	}
	return results, nil
}

func (m *MPDClient) Search(typ, query string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Search(typ, query)
}

func (m *MPDClient) Find(typ, query string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Find(typ, query)
}

// FindAdd adds search results to the queue.
func (m *MPDClient) FindAdd(typ, query string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	results, err := m.client.Find(typ, query)
	if err != nil {
		return err
	}
	for _, song := range results {
		if f, ok := song["file"]; ok {
			_ = m.client.Add(f)
		}
	}
	return nil
}

func (m *MPDClient) ListAllSongs(path string) ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.ListAllInfo(path)
}

func (m *MPDClient) ListOutputs() ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.ListOutputs()
}

func (m *MPDClient) EnableOutput(id int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.EnableOutput(id)
}

func (m *MPDClient) DisableOutput(id int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.DisableOutput(id)
}

func (m *MPDClient) GetStats() (mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Stats()
}

// SavePlaylist / LoadPlaylist / ListPlaylists / RemovePlaylist

func (m *MPDClient) SavePlaylist(name string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.PlaylistSave(name)
}

func (m *MPDClient) LoadPlaylist(name string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.PlaylistLoad(name, -1, -1)
}

func (m *MPDClient) ListPlaylists() ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.ListPlaylists()
}

func (m *MPDClient) RemovePlaylist(name string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.PlaylistRemove(name)
}

// ──────────────────────────────────────────────
// MPD config file editing (local MPD only)
// ──────────────────────────────────────────────

func (m *MPDClient) GetConfig(key string) (string, error) {
	if m.config.MPD.ConfigPath == "" {
		return "", fmt.Errorf("MPD config path not set")
	}
	data, err := os.ReadFile(m.config.MPD.ConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read MPD config: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[0] == key {
			return strings.Trim(strings.Join(parts[1:], " "), "\""), nil
		}
	}
	return "", fmt.Errorf("config key %q not found", key)
}

func (m *MPDClient) SetConfig(key, value string) error {
	if m.config.MPD.ConfigPath == "" {
		return fmt.Errorf("MPD config path not set")
	}
	data, err := os.ReadFile(m.config.MPD.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read MPD config: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			continue
		}
		parts := strings.Fields(t)
		if len(parts) >= 1 && parts[0] == key {
			lines[i] = fmt.Sprintf("%s \"%s\"", key, value)
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, fmt.Sprintf("%s \"%s\"", key, value))
	}
	return os.WriteFile(m.config.MPD.ConfigPath, []byte(strings.Join(lines, "\n")), 0644)
}

// ──────────────────────────────────────────────
// Display helpers
// ──────────────────────────────────────────────

func getTerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

func printSeparator() {
	fmt.Println(strings.Repeat("─", getTerminalWidth()))
}

func formatDuration(secs string) string {
	sec, err := strconv.ParseFloat(secs, 64)
	if err != nil || secs == "" {
		return "0:00"
	}
	m := int(sec) / 60
	s := int(sec) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatBitrate(attrs mpd.Attrs) string {
	if audio, ok := attrs["audio"]; ok {
		parts := strings.Split(audio, ":")
		if len(parts) >= 1 {
			if sr, err := strconv.Atoi(parts[0]); err == nil {
				return fmt.Sprintf("%d kHz", sr/1000)
			}
		}
	}
	if bitrate, ok := attrs["bitrate"]; ok && bitrate != "" {
		return fmt.Sprintf("%s kbps", bitrate)
	}
	return "N/A"
}

func getOrDefault(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

func boolState(s string) string {
	if s == "1" {
		return "on"
	}
	return "off"
}

// ──────────────────────────────────────────────
// printStatus – MPC-style rich output
// ──────────────────────────────────────────────

// printStatus prints a rich, MPC-compatible status block.
func printStatus(client *MPDClient) error {
	status, err := client.Status()
	if err != nil {
		return err
	}
	song, err := client.CurrentSong()
	if err != nil {
		return err
	}

	state := status["state"]
	width := getTerminalWidth()
	sep := strings.Repeat("─", width)

	fmt.Println(sep)

	// ── Current song ────────────────────────────────────────────────
	if state != "stop" {
		title := getOrDefault(song, "Title", song["file"])
		artist := getOrDefault(song, "Artist", "Unknown Artist")
		album := getOrDefault(song, "Album", "Unknown Album")
		date := song["Date"]
		track := song["Track"]
		genre := song["Genre"]
		file := song["file"]

		fmt.Printf("%s%s%s\n", Bold+ColorWhite, title, Reset)
		fmt.Printf("%s%s%s\n", ColorYellow, artist, Reset)
		fmt.Printf("%s%s%s", ColorOrange, album, Reset)
		if date != "" {
			fmt.Printf("  %s(%s)%s", ColorGray, date, Reset)
		}
		fmt.Println()
		if track != "" {
			fmt.Printf("  Track: %s%s%s", ColorCyan, track, Reset)
		}
		if genre != "" {
			fmt.Printf("   Genre: %s%s%s", ColorCyan, genre, Reset)
		}
		if track != "" || genre != "" {
			fmt.Println()
		}
		fmt.Printf("  %sFile: %s%s%s\n", ColorGray, Reset, file, Reset)
		fmt.Println()
	}

	// ── Progress bar ────────────────────────────────────────────────
	if state == "play" || state == "pause" {
		elapsed := status["elapsed"]
		duration := song["duration"]
		elapsedSec, _ := strconv.ParseFloat(elapsed, 64)
		durSec, _ := strconv.ParseFloat(duration, 64)

		elapsedFmt := formatDuration(elapsed)
		durationFmt := formatDuration(duration)

		barWidth := width - 20
		if barWidth < 10 {
			barWidth = 10
		}
		filled := 0
		if durSec > 0 {
			filled = int(elapsedSec / durSec * float64(barWidth))
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		stateIcon := "▶"
		stateColor := ColorGreen
		if state == "pause" {
			stateIcon = "⏸"
			stateColor = ColorYellow
		}
		fmt.Printf("%s%s%s [%s%s%s] %s / %s\n",
			stateColor, stateIcon, Reset,
			ColorCyan, bar, Reset,
			elapsedFmt, durationFmt)
	} else {
		fmt.Printf("%s⏹ Stopped%s\n", ColorRed, Reset)
	}

	// ── Status line ─────────────────────────────────────────────────
	vol := status["volume"]
	repeat := boolState(status["repeat"])
	random := boolState(status["random"])
	single := boolState(status["single"])
	consume := boolState(status["consume"])
	xfade := status["xfade"]
	bitrate := formatBitrate(status)
	qlen := status["playlistlength"]
	qpos := status["song"]

	fmt.Println()
	fmt.Printf("  %sVolume:%s %-4s  %sRepeat:%s %-3s  %sRandom:%s %-3s  %sSingle:%s %-3s  %sConsume:%s %-3s\n",
		ColorGray, Reset, vol+"%",
		ColorGray, Reset, repeat,
		ColorGray, Reset, random,
		ColorGray, Reset, single,
		ColorGray, Reset, consume)
	fmt.Printf("  %sQueue:%s %s/%s  %sBitrate:%s %s  %sCrossfade:%s %ss\n",
		ColorGray, Reset, qpos, qlen,
		ColorGray, Reset, bitrate,
		ColorGray, Reset, xfade)

	fmt.Println(sep)
	return nil
}

// ──────────────────────────────────────────────
// Playlist renderer
// ──────────────────────────────────────────────

func renderPlayingBanner(index string, song mpd.Attrs) string {
	title := getOrDefault(song, "Title", "Unknown")
	artist := getOrDefault(song, "Artist", "Unknown")
	album := getOrDefault(song, "Album", "Unknown")
	result := fmt.Sprintf("%s %sPLAYING:%s %s %s%s%s. %s%s%s · %s%s%s · %s%s%s",
		PlayingBg, PlayingLabel, Reset,
		Icon, FgIndex, index, Reset,
		FgTitle, title, Reset,
		FgArtist, artist, Reset,
		FgAlbum, album, Reset)
	if d, ok := song["Date"]; ok && d != "" {
		result += fmt.Sprintf(" %s(%s)%s", FgDate, d, Reset)
	}
	return result
}

func renderPlaylist(client *MPDClient, messages []string) error {
	playlist, err := client.PlaylistInfo()
	if err != nil {
		return err
	}
	status, err := client.Status()
	if err != nil {
		return err
	}

	currentID := status["songid"]
	pad := len(strconv.Itoa(len(playlist)))

	var lines []string
	var nowIdx string
	var nowSong mpd.Attrs

	fmt.Print("\033c") // clear screen

	for idx, song := range playlist {
		idxStr := fmt.Sprintf("%0*d", pad, idx+1)
		title := getOrDefault(song, "Title", "Unknown")
		artist := getOrDefault(song, "Artist", "Unknown")
		album := getOrDefault(song, "Album", "Unknown")

		line := fmt.Sprintf("%s %s%s%s. %s%s%s · %s%s%s · %s%s%s",
			Icon, FgIndex, idxStr, Reset,
			FgTitle, title, Reset,
			FgArtist, artist, Reset,
			FgAlbum, album, Reset)

		if d, ok := song["Date"]; ok && d != "" {
			line += fmt.Sprintf(" %s(%s)%s", FgDate, d, Reset)
		}

		if song["Id"] == currentID {
			line = fmt.Sprintf("%s%s%s", BgBlue, line, Reset)
			nowIdx = idxStr
			nowSong = song
		}
		lines = append(lines, line)
	}

	fmt.Println(strings.Join(lines, "\n"))

	if nowIdx != "" {
		fmt.Println("\n" + renderPlayingBanner(nowIdx, nowSong))
	}

	if len(messages) > 0 {
		fmt.Println()
		for _, msg := range messages {
			fmt.Println("  " + msg)
		}
	}

	return nil
}

// formatConsoleMessage formats a single-song console display block.
// When showProgress is true, a terminal-width ASCII progress bar is rendered
// between the time line and the metadata lines.
func formatConsoleMessage(song mpd.Attrs, status mpd.Attrs, showProgress bool) string {
	pos := status["song"]
	total := status["playlistlength"]
	elapsed := formatDuration(status["elapsed"])
	duration := formatDuration(song["duration"])
	track := getOrDefault(song, "Track", "?")
	title := getOrDefault(song, "Title", song["file"])
	artist := getOrDefault(song, "Artist", "")
	album := getOrDefault(song, "Album", "")
	bitrate := formatBitrate(status)
	file := song["file"]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s▶ %s/%s/%s. %s%s\n", ColorCyan, pos, total, track, title, Reset))

	if showProgress {
		// Build progress bar sized to terminal width.
		// Layout: "  [████░░░░░░░░] 1:23 / 4:56"
		elapsedSec, _ := strconv.ParseFloat(status["elapsed"], 64)
		durSec, _ := strconv.ParseFloat(song["duration"], 64)

		width := getTerminalWidth()
		// Reserve space for "  [" + "] " + "0:00 / 0:00" (≈ 20 chars)
		barWidth := width - 22
		if barWidth < 8 {
			barWidth = 8
		}
		filled := 0
		if durSec > 0 {
			filled = int(elapsedSec / durSec * float64(barWidth))
			if filled > barWidth {
				filled = barWidth
			}
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		sb.WriteString(fmt.Sprintf("%s  [%s] %s / %s%s\n",
			ColorCyan, bar, elapsed, duration, Reset))
	} else {
		sb.WriteString(fmt.Sprintf("%s  🕓 %s / %s%s\n", ColorCyan, elapsed, duration, Reset))
	}

	if artist != "" {
		sb.WriteString(fmt.Sprintf("%s  🎤 %s%s\n", ColorYellow, artist, Reset))
	}
	if album != "" {
		sb.WriteString(fmt.Sprintf("%s  💿 %s%s\n", ColorOrange, album, Reset))
	}
	sb.WriteString(fmt.Sprintf("%s  🎵 %s%s\n", ColorBlue, bitrate, Reset))
	sb.WriteString(fmt.Sprintf("%s  📁 %s%s", ColorGreen, file, Reset))
	return sb.String()
}

// ──────────────────────────────────────────────
// GNTP / Notification
// ──────────────────────────────────────────────

// AppState holds runtime state for the monitor loop.
type AppState struct {
	client       *MPDClient
	gntp         *gntp.Client
	config       *Config
	debug        bool
	gntpEnabled  bool
	showProgress bool // whether to render progress bar in monitor output
	lastSongFile string
	lastState    string
}

func setupGNTP(cfg *Config, debug bool) (*gntp.Client, bool) {
	if !cfg.GNTP.Enabled {
		return nil, false
	}
	c := gntp.NewClient("MPD Monitor").
		WithHost(cfg.GNTP.Host).
		WithPort(cfg.GNTP.Port).
		WithTimeout(10 * time.Second)

	switch strings.ToLower(cfg.GNTP.IconMode) {
	case "dataurl":
		c.WithIconMode(gntp.IconModeDataURL)
	case "fileurl":
		c.WithIconMode(gntp.IconModeFileURL)
	case "httpurl":
		c.WithIconMode(gntp.IconModeHttpURL)
	default:
		c.WithIconMode(gntp.IconModeBinary)
	}

	sc := gntp.NewNotificationType("song_change").WithDisplayName("Song Changed")
	ps := gntp.NewNotificationType("player_state").WithDisplayName("Player State")
	if err := c.Register([]*gntp.NotificationType{sc, ps}); err != nil {
		if debug {
			log.Printf("⚠️  GNTP register failed: %v", err)
		}
		return nil, false
	}
	return c, true
}

func getAlbumArt(client *mpd.Client, uri string) *gntp.Resource {
	for _, fn := range []func(string) ([]byte, error){
		client.ReadPicture,
		client.AlbumArt,
	} {
		data, err := fn(uri)
		if err == nil && len(data) > 0 {
			ct := "image/jpeg"
			if len(data) > 4 && data[0] == 0x89 && data[1] == 0x50 {
				ct = "image/png"
			}
			return gntp.LoadResourceFromBytes(data, ct)
		}
	}
	return nil
}

func formatNotificationMessage(song, status mpd.Attrs) string {
	var sb strings.Builder
	pos := status["song"]
	total := status["playlistlength"]
	track := getOrDefault(song, "Track", "?")
	title := getOrDefault(song, "Title", song["file"])
	artist := getOrDefault(song, "Artist", "")
	album := getOrDefault(song, "Album", "")
	sb.WriteString(fmt.Sprintf("%s/%s/%s. %s\n", pos, total, track, title))
	sb.WriteString(fmt.Sprintf("%s / %s\n", formatDuration(status["elapsed"]), formatDuration(song["duration"])))
	if artist != "" {
		sb.WriteString(fmt.Sprintf("🎤 %s\n", artist))
	}
	if album != "" {
		sb.WriteString(fmt.Sprintf("💿 %s\n", album))
	}
	sb.WriteString(fmt.Sprintf("🎵 %s\n", formatBitrate(status)))
	sb.WriteString(fmt.Sprintf("📁 %s", song["file"]))
	return sb.String()
}

func sendNotification(state *AppState, event, title, message string, icon *gntp.Resource) error {
	if !state.gntpEnabled || state.gntp == nil {
		return nil
	}
	opts := gntp.NewNotifyOptions()
	if icon != nil {
		opts.WithIcon(icon)
	}
	return state.gntp.NotifyWithOptions(event, title, message, opts)
}

// ──────────────────────────────────────────────
// Monitor loop
// ──────────────────────────────────────────────

func checkStatus(state *AppState) error {
	if err := state.client.client.Ping(); err != nil {
		return fmt.Errorf("connection lost: %v", err)
	}
	status, err := state.client.Status()
	if err != nil {
		return fmt.Errorf("get status: %v", err)
	}
	song, err := state.client.CurrentSong()
	if err != nil {
		return fmt.Errorf("get current song: %v", err)
	}

	currentState := status["state"]
	currentFile := song["file"]
	songChanged := currentFile != state.lastSongFile && currentFile != ""
	stateChanged := currentState != state.lastState && state.lastState != ""

	if currentState == "play" && currentFile != "" {
		fmt.Println()
		fmt.Println(formatConsoleMessage(song, status, state.showProgress))
		printSeparator()
	} else if stateChanged {
		fmt.Printf("⏸  State: %s\n", currentState)
		printSeparator()
	}

	if songChanged && currentState == "play" {
		art := getAlbumArt(state.client.client, currentFile)
		title := getOrDefault(song, "Title", currentFile)
		if err := sendNotification(state, "song_change", title, formatNotificationMessage(song, status), art); err != nil {
			if state.debug {
				log.Printf("⚠️  Notification: %v", err)
			}
		}
		state.lastSongFile = currentFile
	}

	if stateChanged {
		stateMsg := map[string]string{
			"play":  "▶ Playing",
			"pause": "⏸ Paused",
			"stop":  "⏹ Stopped",
		}[currentState]
		if stateMsg == "" {
			stateMsg = currentState
		}
		var art *gntp.Resource
		if currentFile != "" {
			art = getAlbumArt(state.client.client, currentFile)
		}
		msg := stateMsg
		if currentState == "play" && currentFile != "" {
			msg = formatNotificationMessage(song, status)
		}
		_ = sendNotification(state, "player_state", stateMsg, msg, art)
	}

	state.lastState = currentState
	return nil
}

func monitorOnce(state *AppState) error {
	w, err := mpd.NewWatcher("tcp",
		fmt.Sprintf("%s:%s", state.config.MPD.Host, state.config.MPD.Port),
		state.config.MPD.Password, "player", "mixer")
	if err != nil {
		return fmt.Errorf("watcher: %v", err)
	}
	done := make(chan struct{})
	defer close(done)

	go func() {
		defer func() { recover() }()
		for {
			select {
			case err, ok := <-w.Error:
				if !ok {
					return
				}
				if state.debug {
					log.Printf("⚠️  Watcher error: %v", err)
				}
			case <-done:
				return
			}
		}
	}()

	for {
		select {
		case subsystem, ok := <-w.Event:
			if !ok {
				w.Close()
				return fmt.Errorf("event channel closed")
			}
			if subsystem == "database" || subsystem == "update" {
				continue
			}
			if err := checkStatus(state); err != nil {
				if state.debug {
					log.Printf("⚠️  checkStatus: %v", err)
				}
				if isConnectionErr(err) {
					w.Close()
					return err
				}
			}
		case <-done:
			w.Close()
			return nil
		case <-time.After(30 * time.Second):
			if err := state.client.client.Ping(); err != nil {
				w.Close()
				return fmt.Errorf("ping: %v", err)
			}
		}
	}
}

func isConnectionErr(err error) bool {
	s := err.Error()
	return strings.Contains(s, "EOF") ||
		strings.Contains(s, "connection") ||
		strings.Contains(s, "broken pipe")
}

func runMonitor(state *AppState) error {
	log.Printf("🎵 MPD Monitor  %s:%s", state.config.MPD.Host, state.config.MPD.Port)
	if state.gntpEnabled {
		log.Printf("📢 GNTP  %s:%d  (icon: %s)", state.config.GNTP.Host, state.config.GNTP.Port, state.config.GNTP.IconMode)
	} else {
		log.Println("📢 Notifications: disabled")
	}
	if state.debug {
		log.Println("🐛 Debug: on")
	}
	fmt.Println(strings.Repeat("=", getTerminalWidth()))
	_ = checkStatus(state)

	for {
		err := monitorOnce(state)
		if err != nil && (isConnectionErr(err) || strings.Contains(err.Error(), "watcher")) {
			if state.debug {
				log.Printf("🔄 Reconnecting: %v", err)
			}
			time.Sleep(2 * time.Second)
			if err := state.client.reconnect(); err != nil {
				time.Sleep(5 * time.Second)
			}
			continue
		}
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
}

// ──────────────────────────────────────────────
// Config file discovery
// ──────────────────────────────────────────────

func getConfigFile(configName string) string {
	var candidates []string

	if runtime.GOOS == "windows" {
		userProfile := os.Getenv("USERPROFILE")
		appData := os.Getenv("APPDATA")
		for _, base := range []string{appData, userProfile} {
			for _, ext := range []string{"", ".toml", ".ini", ".json", ".yaml", ".yml", ".env"} {
				name := configName
				if ext != "" {
					name = configName + ext
				} else {
					name = "." + configName
				}
				candidates = append(candidates, filepath.Join(base, name))
				candidates = append(candidates, filepath.Join(base, configName, name))
			}
		}
	} else {
		home, _ := os.UserHomeDir()
		for _, dir := range []string{home, filepath.Join(home, ".config", configName), filepath.Join(home, ".config")} {
			for _, ext := range []string{"", ".toml", ".ini", ".json", ".yaml", ".yml", ".env"} {
				name := "." + configName
				if ext != "" {
					name = configName + ext
				}
				candidates = append(candidates, filepath.Join(dir, name))
			}
		}
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}

	// fallback: next to executable
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "."+configName)
}

// ──────────────────────────────────────────────
// Help text
// ──────────────────────────────────────────────

func printHelp() {
	w := getTerminalWidth()
	sep := strings.Repeat("═", w)
	fmt.Printf(`
%s
%smpdl%s — Advanced MPD CLI Client  v%s
%s

%sUSAGE%s
  mpdl [OPTIONS] <command> [args...]

%sOPTIONS%s
  -c, --config PATH      Config file (auto-detected if omitted)
  --mpd-host HOST        MPD host     (default: localhost)
  --mpd-port PORT        MPD port     (default: 6600)
  --mpd-password PASS    MPD password
  --music-root PATH      Music root directory
  --mpd-config PATH      MPD config file path (get-config/set-config)
  --debug                Enable debug mode
  -h, --help             This help
  -v, --version          Show version

%sPLAYBACK%s
  play                   Resume if paused, else start from current position
  play <N>               Play queue position N (1-based)
  play <pattern>         Search queue and play matching track(s).
                         Pattern formats:
                           *girl*   – wildcard/glob (case-insensitive)
                           /girl/   – regular expression
                           girl     – substring match (title/artist/album/file)
                         1 match  → plays immediately
                         N matches → numbered prompt; pick number, 'a'=play all, Enter=cancel
  pause / toggle         Toggle play/pause
  stop                   Stop
  next, n                Next track
  prev, previous, p      Previous track
  seek [+/-]SECS         Seek (absolute or relative, decimal OK)

%sPLAYLIST%s
  add PATH               Add file/dir to queue
  addplay PATH           Clear queue, add PATH, play from track 1
  insert PATH            Insert after currently playing track
  del ARG                Delete from queue.  ARG can be:
                           N        – 1-based position (0 = current)
                           N-M      – position range  (e.g. 3-7)
                           /regex/  – regex against title/artist/file
                           glob*    – wildcard/glob pattern
                           text     – substring/path match
  clear                  Clear queue
  crop                   Remove all except current track
  shuffle                Shuffle queue
  move <from> <to>       Move track (1-based)
  list, ls, playlist     Show queue

%sSAVED PLAYLISTS%s
  save <name>            Save queue as playlist
  load <name>            Load saved playlist into queue
  lsplaylists            List saved playlists
  rm <name>              Delete saved playlist

%sSEARCH%s
  search <type> <query>  Search (partial)  types: artist album title any …
  find   <type> <query>  Find   (exact)
  findadd <type> <query> Find and add results to queue
  listall [PATH]         List all files in music dir

%sINFORMATION%s
  status, st             Full status (MPC-style with progress bar)
  current                Currently playing song
  stats                  MPD statistics
  version                MPD server version

%sOPTIONS/MODES%s
  volume [+/-][VOL]      Set/adjust volume (0-100)
  repeat  [on|off]       Repeat mode
  random  [on|off]       Random/shuffle mode
  single  [on|off]       Single mode
  consume [on|off]       Consume mode
  crossfade [SECS]       Crossfade duration

%sOUTPUTS%s
  outputs                List audio outputs
  enable  <id>           Enable output
  disable <id>           Disable output

%sDATABASE%s
  update [PATH]          Update music database
  rescan [PATH]          Force full rescan

%sCONFIG (local MPD only)%s
  get-config KEY
  set-config KEY VALUE

%sMONITOR%s
  monitor, mon, m        Monitor + GNTP notifications
    p  play/pause   n  next   b  prev   s  stop   q  quit

%sMEDIA KEYS%s
  mediakeys              Show media-key / Bluetooth headset setup guide

%sENVIRONMENT%s
  MPD_HOST  MPD_PORT  MPD_PASSWORD  MPD_TIMEOUT  MPD_MUSIC_ROOT  DEBUG

%s
`, sep,
		Bold+ColorCyan, Reset, Version,
		sep,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		Bold+ColorYellow, Reset,
		sep)
}

// ──────────────────────────────────────────────
// main
// ──────────────────────────────────────────────

func main() {
	var (
		configFile    string
		mpdHost       string
		mpdPort       string
		mpdPassword   string
		musicRoot     string
		mpdConfigPath string
		showVersion   bool
		showHelp      bool
		debugMode     bool
	)

	flag.StringVar(&configFile, "config", "", "Config file path")
	flag.StringVar(&configFile, "c", "", "Config file path (short)")
	flag.StringVar(&mpdHost, "mpd-host", "", "MPD host")
	flag.StringVar(&mpdPort, "mpd-port", "", "MPD port")
	flag.StringVar(&mpdPassword, "mpd-password", "", "MPD password")
	flag.StringVar(&musicRoot, "music-root", "", "Music root")
	flag.StringVar(&mpdConfigPath, "mpd-config", "", "MPD config file")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&debugMode, "debug", false, "Debug mode")
	flag.Parse()

	if showVersion {
		fmt.Printf("mpdl %s  (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}
	if showHelp || flag.NArg() == 0 {
		printHelp()
		os.Exit(0)
	}
	if os.Getenv("DEBUG") == "1" {
		debugMode = true
	}

	// Resolve config file
	if configFile == "" {
		configFile = getConfigFile("mpdl")
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("❌ Config: %v", err)
	}

	// CLI flag overrides
	if mpdHost != "" {
		config.MPD.Host = mpdHost
	}
	if mpdPort != "" {
		config.MPD.Port = mpdPort
	}
	if mpdPassword != "" {
		config.MPD.Password = mpdPassword
	}
	if musicRoot != "" {
		config.MPD.MusicRoot = musicRoot
	}
	if mpdConfigPath != "" {
		config.MPD.ConfigPath = mpdConfigPath
	}

	client, err := NewMPDClient(config.MPD.Host, config.MPD.Port, config.MPD.Password, config)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	defer client.Close()

	args := flag.Args()
	command := strings.ToLower(args[0])
	cargs := args[1:]

	var messages []string

	switch command {

	// ── Playback ────────────────────────────────────────────────────

	case "play":
		if len(cargs) == 0 {
			// No args: resume if paused, else start from current/beginning
			status, err := client.Status()
			if err != nil {
				log.Fatalf("❌ status: %v", err)
			}
			if status["state"] == "pause" {
				if err := client.Pause(); err != nil {
					log.Fatalf("❌ resume: %v", err)
				}
			} else {
				if err := client.Play(-1); err != nil {
					log.Fatalf("❌ play: %v", err)
				}
			}
			fmt.Printf("%s▶ Playing%s\n", ColorGreen, Reset)

		} else {
			arg := strings.Join(cargs, " ")

			// ── Numeric position: play 5  ──────────────────────────────
			if pos, err := strconv.Atoi(arg); err == nil {
				if pos < 1 {
					log.Fatal("❌ Position must be ≥ 1")
				}
				if err := client.Play(pos - 1); err != nil {
					log.Fatalf("❌ play: %v", err)
				}
				fmt.Printf("%s▶ Playing position %d%s\n", ColorGreen, pos, Reset)
				break
			}

			// ── Pattern: play *girl* or /regex/ or substring ───────────
			matches, err := client.FindInQueue(arg)
			if err != nil {
				log.Fatalf("❌ queue search: %v", err)
			}

			switch len(matches) {
			case 0:
				fmt.Printf("%s⚠ No tracks in queue match %q%s\n", ColorYellow, arg, Reset)
				fmt.Println("  Tip: use 'mpdl search' to find tracks, then 'mpdl add' to queue them.")
				os.Exit(1)

			case 1:
				// Exactly one match — play it directly
				pos, _ := strconv.Atoi(matches[0]["Pos"])
				if err := client.Play(pos); err != nil {
					log.Fatalf("❌ play: %v", err)
				}
				title := getOrDefault(matches[0], "Title", matches[0]["file"])
				fmt.Printf("%s▶ Playing: %s%s\n", ColorGreen, title, Reset)

			default:
				// Multiple matches — show numbered list, ask user to choose
				fmt.Printf("%s%d tracks match %q — pick one:%s\n",
					ColorYellow, len(matches), arg, Reset)
				fmt.Println()
				pad := len(strconv.Itoa(len(matches)))
				for i, song := range matches {
					qpos, _ := strconv.Atoi(song["Pos"])
					title := getOrDefault(song, "Title", song["file"])
					artist := getOrDefault(song, "Artist", "")
					album := getOrDefault(song, "Album", "")
					line := fmt.Sprintf("  %s%0*d%s. %s%s%s",
						FgIndex, pad, i+1, Reset,
						FgTitle, title, Reset)
					if artist != "" {
						line += fmt.Sprintf(" · %s%s%s", FgArtist, artist, Reset)
					}
					if album != "" {
						line += fmt.Sprintf(" · %s%s%s", FgAlbum, album, Reset)
					}
					line += fmt.Sprintf("  %s[queue #%d]%s", ColorGray, qpos+1, Reset)
					fmt.Println(line)
				}
				fmt.Println()
				fmt.Printf("  Enter number (1-%d), or 'a' to play all from first match, or Enter to cancel: ", len(matches))

				var input string
				fmt.Scanln(&input)
				input = strings.TrimSpace(input)

				switch {
				case input == "" || input == "q":
					fmt.Println("  Cancelled.")

				case input == "a" || input == "all":
					// Play from the first match
					pos, _ := strconv.Atoi(matches[0]["Pos"])
					if err := client.Play(pos); err != nil {
						log.Fatalf("❌ play: %v", err)
					}
					title := getOrDefault(matches[0], "Title", matches[0]["file"])
					fmt.Printf("%s▶ Playing from: %s%s\n", ColorGreen, title, Reset)

				default:
					choice, err := strconv.Atoi(input)
					if err != nil || choice < 1 || choice > len(matches) {
						fmt.Printf("%s❌ Invalid choice: %q%s\n", ColorRed, input, Reset)
						os.Exit(1)
					}
					song := matches[choice-1]
					pos, _ := strconv.Atoi(song["Pos"])
					if err := client.Play(pos); err != nil {
						log.Fatalf("❌ play: %v", err)
					}
					title := getOrDefault(song, "Title", song["file"])
					fmt.Printf("%s▶ Playing: %s%s\n", ColorGreen, title, Reset)
				}
			}
		}

	case "pause", "toggle":
		status, err := client.Status()
		if err != nil {
			log.Fatalf("❌ status: %v", err)
		}
		if status["state"] == "play" {
			if err := client.Pause(); err != nil {
				log.Fatalf("❌ pause: %v", err)
			}
			fmt.Printf("%s⏸ Paused%s\n", ColorYellow, Reset)
		} else {
			if err := client.Play(-1); err != nil {
				log.Fatalf("❌ play: %v", err)
			}
			fmt.Printf("%s▶ Playing%s\n", ColorGreen, Reset)
		}

	case "stop":
		if err := client.Stop(); err != nil {
			log.Fatalf("❌ stop: %v", err)
		}
		fmt.Printf("%s⏹ Stopped%s\n", ColorRed, Reset)

	case "next", "n":
		if err := client.Next(); err != nil {
			log.Fatalf("❌ next: %v", err)
		}
		fmt.Printf("%s⏭ Next%s\n", ColorCyan, Reset)

	case "prev", "previous", "p":
		if err := client.Previous(); err != nil {
			log.Fatalf("❌ prev: %v", err)
		}
		fmt.Printf("%s⏮ Previous%s\n", ColorCyan, Reset)

	case "seek":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl seek [+/-]SECONDS")
		}
		seekStr := cargs[0]
		relative := strings.HasPrefix(seekStr, "+") || strings.HasPrefix(seekStr, "-")
		trimmed := strings.TrimPrefix(strings.TrimPrefix(seekStr, "+"), "-")
		val, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			log.Fatalf("❌ Invalid seek value: %s", seekStr)
		}
		if strings.HasPrefix(seekStr, "-") {
			val = -val
		}
		if err := client.Seek(val, relative); err != nil {
			log.Fatalf("❌ seek: %v", err)
		}
		if relative {
			if val > 0 {
				fmt.Printf("⏩ +%.1fs\n", val)
			} else {
				fmt.Printf("⏪ %.1fs\n", val)
			}
		} else {
			fmt.Printf("⏩ → %.1fs\n", val)
		}

	// ── Playlist mutation ───────────────────────────────────────────

	case "add":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl add PATH")
		}
		path := resolveLocalPath(strings.Join(cargs, " "))

		if err := client.Add(path); err != nil {
			messages = append(messages, fmt.Sprintf("%s❌ Add failed: %s%s, %s%s%s", ColorRed, err, Reset, ColorYellow, path, Reset))
		} else {
			messages = append(messages, fmt.Sprintf("%s✅ Added: %s%s", ColorGreen, path, Reset))
		}
		if err := renderPlaylist(client, messages); err != nil {
			log.Fatalf("❌ %v", err)
		}

	case "addplay", "ap":
		// Clear queue, add path, start from track 1
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl addplay PATH")
		}
		path := resolveLocalPath(strings.Join(cargs, " "))
		if err := client.AddAndPlay(path); err != nil {
			log.Fatalf("❌ addplay: %v", err)
		}
		fmt.Printf("%s▶ Queue replaced — Playing: %s%s\n", ColorGreen, path, Reset)

	case "insert":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl insert PATH")
		}
		path := resolveLocalPath(strings.Join(cargs, " "))
		if err := client.Insert(path); err != nil {
			log.Fatalf("❌ insert: %v", err)
		}
		fmt.Printf("%s✅ Inserted next: %s%s\n", ColorGreen, path, Reset)

	case "del", "delete", "rm-track":
	    if len(cargs) == 0 {
	        log.Fatal("❌ Usage: mpdl del <pos|range|/regex/|glob|text|path>")
	    }
	    for _, c := range cargs {
	        // First, check if it's a file or directory path that exists
	        info, err := os.Stat(c)
	        if err == nil {
	            // Path exists
	            if info.IsDir() {
	                path := strings.Trim(c, "\"'")
	                path = resolveLocalPath(path)
	                deleted, err := client.Delete(path)
	                if err != nil || deleted == 0 {
	                    messages = append(messages, fmt.Sprintf("\033[37;41mFailed to delete: '%s'%s", path, Reset))
	                }
	                if err != nil {
	                    messages = append(messages, fmt.Sprintf("\033[37;41mError: %v%s", err, Reset))
	                }
	            } else {
	                // It's a file, treat like other patterns
	                arg := strings.Trim(c, "\"'")
	                deleted, err := client.Delete(arg)
	                if err != nil {
	                    messages = append(messages, fmt.Sprintf("%s❌ Delete failed: %v%s", ColorRed, err, Reset))
	                } else if deleted == 0 {
	                    messages = append(messages, fmt.Sprintf("%s⚠ No tracks matched: %q%s", ColorYellow, arg, Reset))
	                } else {
	                    messages = append(messages, fmt.Sprintf("%s✅ Deleted %d track(s) matching %q%s", ColorGreen, deleted, arg, Reset))
	                }
	            }
	        } else {
	            // Path doesn't exist, treat as position/range/regex/glob/text
	            arg := strings.Trim(c, "\"'")
	            deleted, err := client.Delete(arg)
	            if err != nil {
	                messages = append(messages, fmt.Sprintf("%s❌ Delete failed: %v%s", ColorRed, err, Reset))
	            } else if deleted == 0 {
	                messages = append(messages, fmt.Sprintf("%s⚠ No tracks matched: %q%s", ColorYellow, arg, Reset))
	            } else {
	                messages = append(messages, fmt.Sprintf("%s✅ Deleted %d track(s) matching %q%s", ColorGreen, deleted, arg, Reset))
	            }
	        }
	    }
	    
	    if err := renderPlaylist(client, messages); err != nil {
	        log.Fatalf("❌ %v", err)
	    }

	case "clear":
		if err := client.Clear(); err != nil {
			log.Fatalf("❌ clear: %v", err)
		}
		fmt.Println("🗑  Queue cleared")

	case "crop":
		if err := client.Crop(); err != nil {
			log.Fatalf("❌ crop: %v", err)
		}
		fmt.Println("✂️  Cropped to current track")

	case "shuffle":
		if err := client.Shuffle(); err != nil {
			log.Fatalf("❌ shuffle: %v", err)
		}
		fmt.Println("🔀 Queue shuffled")

	case "move", "mv":
		if len(cargs) < 2 {
			log.Fatal("❌ Usage: mpdl move <from> <to>  (1-based)")
		}
		from, e1 := strconv.Atoi(cargs[0])
		to, e2 := strconv.Atoi(cargs[1])
		if e1 != nil || e2 != nil || from < 1 || to < 1 {
			log.Fatal("❌ Positions must be positive integers")
		}
		if err := client.Move(from-1, to-1); err != nil {
			log.Fatalf("❌ move: %v", err)
		}
		fmt.Printf("📦 Moved %d → %d\n", from, to)

	case "list", "ls", "playlist":
		if err := renderPlaylist(client, messages); err != nil {
			log.Fatalf("❌ %v", err)
		}

	// ── Saved playlists ─────────────────────────────────────────────

	case "save":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl save <name>")
		}
		name := strings.Trim(strings.Join(cargs, " "), "\"'")
		if err := client.SavePlaylist(name); err != nil {
			log.Fatalf("❌ save: %v", err)
		}
		fmt.Printf("💾 Saved playlist: %q\n", name)

	case "load":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl load <name>")
		}
		name := strings.Trim(strings.Join(cargs, " "), "\"'")
		if err := client.LoadPlaylist(name); err != nil {
			log.Fatalf("❌ load: %v", err)
		}
		fmt.Printf("📂 Loaded playlist: %q\n", name)

	case "lsplaylists":
		pls, err := client.ListPlaylists()
		if err != nil {
			log.Fatalf("❌ lsplaylists: %v", err)
		}
		if len(pls) == 0 {
			fmt.Println("(no saved playlists)")
		}
		for _, pl := range pls {
			fmt.Printf("  📝 %s  %s(%s)%s\n", pl["playlist"], ColorGray, pl["Last-Modified"], Reset)
		}

	case "rm", "rmplaylist":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl rm <playlist-name>")
		}
		name := strings.Trim(strings.Join(cargs, " "), "\"'")
		if err := client.RemovePlaylist(name); err != nil {
			log.Fatalf("❌ rm: %v", err)
		}
		fmt.Printf("🗑  Removed playlist: %q\n", name)

	// ── Search ──────────────────────────────────────────────────────

	case "search":
		if len(cargs) < 2 {
			log.Fatal("❌ Usage: mpdl search <type> <query>")
		}
		results, err := client.Search(cargs[0], strings.Join(cargs[1:], " "))
		if err != nil {
			log.Fatalf("❌ search: %v", err)
		}
		printSongList(results, "Search")

	case "find":
		if len(cargs) < 2 {
			log.Fatal("❌ Usage: mpdl find <type> <query>")
		}
		results, err := client.Find(cargs[0], strings.Join(cargs[1:], " "))
		if err != nil {
			log.Fatalf("❌ find: %v", err)
		}
		printSongList(results, "Find")

	case "findadd":
		if len(cargs) < 2 {
			log.Fatal("❌ Usage: mpdl findadd <type> <query>")
		}
		if err := client.FindAdd(cargs[0], strings.Join(cargs[1:], " ")); err != nil {
			log.Fatalf("❌ findadd: %v", err)
		}
		fmt.Println("✅ Results added to queue")

	case "listall":
		path := ""
		if len(cargs) > 0 {
			path = strings.Trim(strings.Join(cargs, " "), "\"'")
		}
		songs, err := client.ListAllSongs(path)
		if err != nil {
			log.Fatalf("❌ listall: %v", err)
		}
		printSongList(songs, "ListAll")

	// ── Information ─────────────────────────────────────────────────

	case "status", "st", "":
		if err := printStatus(client); err != nil {
			log.Fatalf("❌ status: %v", err)
		}

	case "current":
		song, err := client.CurrentSong()
		if err != nil {
			log.Fatalf("❌ current: %v", err)
		}
		status, err := client.Status()
		if err != nil {
			log.Fatalf("❌ status: %v", err)
		}
		fmt.Println(formatConsoleMessage(song, status, config.Display.ShowProgress))

	case "stats":
		s, err := client.GetStats()
		if err != nil {
			log.Fatalf("❌ stats: %v", err)
		}
		fmt.Printf("%s📊 MPD Statistics%s\n", Bold, Reset)
		fmt.Printf("  Artists:      %s\n", s["artists"])
		fmt.Printf("  Albums:       %s\n", s["albums"])
		fmt.Printf("  Songs:        %s\n", s["songs"])
		fmt.Printf("  Uptime:       %s\n", formatDuration(s["uptime"]))
		fmt.Printf("  Play time:    %s\n", formatDuration(s["playtime"]))
		fmt.Printf("  DB play time: %s\n", formatDuration(s["db_playtime"]))

	case "version":
		status, err := client.Status()
		if err != nil {
			log.Fatalf("❌ version: %v", err)
		}
		_ = status
		fmt.Printf("mpdl %s\n", Version)

	// ── Volume / modes ──────────────────────────────────────────────

	case "volume", "vol":
		if len(cargs) == 0 {
			s, err := client.Status()
			if err != nil {
				log.Fatalf("❌ status: %v", err)
			}
			fmt.Printf("🔊 Volume: %s%%\n", s["volume"])
		} else {
			arg := cargs[0]
			if strings.HasPrefix(arg, "+") || strings.HasPrefix(arg, "-") {
				delta, err := strconv.Atoi(arg)
				if err != nil {
					log.Fatalf("❌ Invalid volume delta: %s", arg)
				}
				if err := client.VolumeRelative(delta); err != nil {
					log.Fatalf("❌ volume: %v", err)
				}
				fmt.Printf("🔊 Volume adjusted %+d\n", delta)
			} else {
				v, err := strconv.Atoi(arg)
				if err != nil || v < 0 || v > 100 {
					log.Fatal("❌ Volume must be 0–100 (or +N/-N for relative)")
				}
				if err := client.Volume(v); err != nil {
					log.Fatalf("❌ volume: %v", err)
				}
				fmt.Printf("🔊 Volume: %d%%\n", v)
			}
		}

	case "repeat":
		toggleBool(client.Repeat, cargs, "🔁 Repeat", "repeat", client)

	case "random":
		toggleBool(client.Random, cargs, "🔀 Random", "random", client)

	case "single":
		toggleBool(client.Single, cargs, "🔂 Single", "single", client)

	case "consume":
		toggleBool(client.Consume, cargs, "🔥 Consume", "consume", client)

	case "crossfade":
		if len(cargs) == 0 {
			s, err := client.Status()
			if err != nil {
				log.Fatalf("❌ status: %v", err)
			}
			fmt.Printf("Crossfade: %ss\n", s["xfade"])
		} else {
			secs, err := strconv.Atoi(cargs[0])
			if err != nil || secs < 0 {
				log.Fatal("❌ Invalid crossfade duration")
			}
			if err := client.Crossfade(secs); err != nil {
				log.Fatalf("❌ crossfade: %v", err)
			}
			fmt.Printf("🔀 Crossfade: %ds\n", secs)
		}

	// ── Outputs ─────────────────────────────────────────────────────

	case "outputs":
		outs, err := client.ListOutputs()
		if err != nil {
			log.Fatalf("❌ outputs: %v", err)
		}
		fmt.Printf("%s🔊 Audio Outputs%s\n", Bold, Reset)
		for _, o := range outs {
			status := ColorRed + "✗ off" + Reset
			if o["outputenabled"] == "1" {
				status = ColorGreen + "✓ on" + Reset
			}
			fmt.Printf("  [%s] %-30s  %s\n", o["outputid"], o["outputname"], status)
		}

	case "enable":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl enable <id>")
		}
		id, err := strconv.Atoi(cargs[0])
		if err != nil {
			log.Fatal("❌ Invalid output ID")
		}
		if err := client.EnableOutput(id); err != nil {
			log.Fatalf("❌ enable: %v", err)
		}
		fmt.Printf("%s✓ Enabled output %d%s\n", ColorGreen, id, Reset)

	case "disable":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl disable <id>")
		}
		id, err := strconv.Atoi(cargs[0])
		if err != nil {
			log.Fatal("❌ Invalid output ID")
		}
		if err := client.DisableOutput(id); err != nil {
			log.Fatalf("❌ disable: %v", err)
		}
		fmt.Printf("%s✗ Disabled output %d%s\n", ColorRed, id, Reset)

	// ── Database ─────────────────────────────────────────────────────

	case "update":
		path := ""
		if len(cargs) > 0 {
			path = strings.Trim(strings.Join(cargs, " "), "\"'")
		}
		jobID, err := client.Update(path)
		if err != nil {
			log.Fatalf("❌ update: %v", err)
		}
		fmt.Printf("🔄 DB update started (job %d)\n", jobID)

	case "rescan":
		path := ""
		if len(cargs) > 0 {
			path = strings.Trim(strings.Join(cargs, " "), "\"'")
		}
		jobID, err := client.Rescan(path)
		if err != nil {
			log.Fatalf("❌ rescan: %v", err)
		}
		fmt.Printf("🔄 Rescan started (job %d)\n", jobID)

	// ── MPD config ──────────────────────────────────────────────────

	case "get-config":
		if len(cargs) == 0 {
			log.Fatal("❌ Usage: mpdl get-config KEY")
		}
		val, err := client.GetConfig(cargs[0])
		if err != nil {
			log.Fatalf("❌ %v", err)
		}
		fmt.Printf("%s = %s\n", cargs[0], val)

	case "set-config":
		if len(cargs) < 2 {
			log.Fatal("❌ Usage: mpdl set-config KEY VALUE")
		}
		if err := client.SetConfig(cargs[0], strings.Join(cargs[1:], " ")); err != nil {
			log.Fatalf("❌ %v", err)
		}
		fmt.Printf("✅ %s = %s\n", cargs[0], strings.Join(cargs[1:], " "))
		fmt.Println("⚠  Restart MPD for changes to take effect")

	// ── Monitor ──────────────────────────────────────────────────────

	case "monitor", "mon", "m":
		gntpClient, gntpEnabled := setupGNTP(config, debugMode)

		// Determine showProgress: config default, overridden by CLI flags.
		// Accepted flags (anywhere in cargs after the subcommand):
		//   --progress    / -p   → force on
		//   --no-progress / -np  → force off
		showProgress := config.Display.ShowProgress
		useMediaKeys := false
		for _, a := range cargs {
			switch a {
			case "--progress", "-p":
				showProgress = true
			case "--no-progress", "-np":
				showProgress = false
			case "--media-keys", "-mk":
				useMediaKeys = true
			}
		}

		state := &AppState{
			client:       client,
			gntp:         gntpClient,
			config:       config,
			debug:        debugMode,
			gntpEnabled:  gntpEnabled,
			showProgress: showProgress,
		}

		var monErr error
		if useMediaKeys {
			monErr = RunWithMediaKeys(state)
		} else {
			monErr = runMonitor(state)
		}
		if monErr != nil {
			log.Fatalf("❌ Monitor: %v", monErr)
		}

	case "mediakeys":
		SetupSystemMediaKeys()

	default:
		fmt.Printf("%s❌ Unknown command: %q%s\n", ColorRed, command, Reset)
		fmt.Println("Run 'mpdl --help' for usage information")
		os.Exit(1)
	}
}

// ──────────────────────────────────────────────
// Utility helpers
// ──────────────────────────────────────────────

// resolveLocalPath expands relative / absolute local paths.
// Paths that look like MPD-relative paths (no leading . / \\ or drive letter)
// are passed through unchanged.
func resolveLocalPath(path string) string {
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	path = strings.Trim(path, "\"'")
	path = strings.TrimSpace(path)
	hasLeading := strings.HasPrefix(path, ".") ||
		strings.HasPrefix(path, "/") ||
		strings.HasPrefix(path, "\\") ||
		(len(path) >= 2 && path[1] == ':')
	if hasLeading {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
	}

	return path
}

// toggleBool toggles or sets a boolean MPD option.
// statusKey is the key to read from MPD status (e.g. "repeat", "random").
func toggleBool(fn func(bool) error, args []string, label, statusKey string, client *MPDClient) {
	var enable bool
	if len(args) == 0 {
		s, err := client.Status()
		if err != nil {
			log.Fatalf("❌ status: %v", err)
		}
		enable = s[statusKey] != "1"
	} else {
		v := strings.ToLower(args[0])
		enable = v == "on" || v == "1" || v == "true" || v == "yes"
	}
	if err := fn(enable); err != nil {
		log.Fatalf("❌ %s: %v", label, err)
	}
	state := "off"
	if enable {
		state = "on"
	}
	fmt.Printf("%s: %s\n", label, state)
}

// printSongList formats a list of songs for search/listall output.
func printSongList(songs []mpd.Attrs, source string) {
	fmt.Printf("%s%s: %d result(s)%s\n", Bold, source, len(songs), Reset)
	for _, song := range songs {
		file := song["file"]
		title := getOrDefault(song, "Title", file)
		artist := getOrDefault(song, "Artist", "")
		album := getOrDefault(song, "Album", "")

		if artist != "" && album != "" {
			fmt.Printf("  %s🎵%s %s%s%s · %s%s%s · %s%s%s\n",
				ColorBlue, Reset,
				FgTitle, title, Reset,
				FgArtist, artist, Reset,
				FgAlbum, album, Reset)
		} else if artist != "" {
			fmt.Printf("  %s🎵%s %s%s%s · %s%s%s\n",
				ColorBlue, Reset,
				FgTitle, title, Reset,
				FgArtist, artist, Reset)
		} else {
			fmt.Printf("  %s🎵%s %s%s%s\n", ColorBlue, Reset, FgTitle, title, Reset)
		}
	}
}
