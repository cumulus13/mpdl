// File: main.go
// Project: mpdl - Advanced MPD CLI client
// Author: Hadi Cahyadi <cumulus13@gmail.com>
// Date: 2026-02-04
// Description: Production-ready MPD CLI with monitoring, playlist management, and MPC-like features
// License: MIT

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	// "github.com/BurntSushi/toml"
	"github.com/cumulus13/go-gntp"
	"github.com/fhs/gompd/v2/mpd"
	"golang.org/x/term"
	// "gopkg.in/yaml.v3"

)

var (
	configFile   string
	mpdHost      string
	mpdPort      string
	mpdPassword  string
	musicRoot    string
	mpdConfigPath string
	showVersion  bool
	showHelp     bool
	debug        bool
)

const (
	Version = "1.0.0"

	// ANSI color codes
	Reset       = "\033[0m"
	BgBlue      = "\033[44m"
	FgIndex     = "\033[33m"
	FgTitle     = "\033[93m"
	FgArtist    = "\033[96m"
	FgAlbum     = "\033[105m"
	FgDate      = "\033[92m"
	PlayingBg   = "\033[48;5;17m"
	PlayingLabel = "\033[97;41m"
	ColorCyan   = "\033[96m"
	ColorYellow = "\033[93m"
	ColorOrange = "\033[38;5;216m"
	ColorBlue   = "\033[94m"
	ColorGreen  = "\033[92m"
	Icon        = "🎵"
)

// Config represents the application configuration
type Config struct {
	MPD struct {
		Host      string `toml:"host"`
		Port      string `toml:"port"`
		Password  string `toml:"password"`
		Timeout   int    `toml:"timeout"`
		MusicRoot string `toml:"music_root"`
		ConfigPath string `toml:"config_path"`
	} `toml:"mpd"`

	GNTP struct {
		Host     string `toml:"host"`
		Port     int    `toml:"port"`
		Password string `toml:"password"`
		IconMode string `toml:"icon_mode"`
		Enabled  bool   `toml:"enabled"`
	} `toml:"gntp"`

	Display struct {
		ShowAlbumArt bool `toml:"show_album_art"`
		UseColor     bool `toml:"use_color"`
	} `toml:"display"`
}

// MPDClient wraps the MPD connection with automatic reconnection
type MPDClient struct {
	host     string
	port     string
	password string
	client   *mpd.Client
	config   *Config
}

// AppState holds the application runtime state
type AppState struct {
	client       *MPDClient
	gntp         *gntp.Client
	config       *Config
	debug        bool
	gntpEnabled  bool
	lastSongFile string
	lastState    string
}

// NewConfig creates a default configuration
func NewConfig() *Config {
	cfg := &Config{}
	
	// MPD defaults
	cfg.MPD.Host = "localhost"
	cfg.MPD.Port = "6600"
	cfg.MPD.Timeout = 10
	cfg.MPD.MusicRoot = getMusicRootDefault()
	cfg.MPD.ConfigPath = getMPDConfigDefault()
	
	// GNTP defaults
	cfg.GNTP.Host = "localhost"
	cfg.GNTP.Port = 23053
	cfg.GNTP.IconMode = "binary"
	cfg.GNTP.Enabled = true
	
	// Display defaults
	cfg.Display.ShowAlbumArt = true
	cfg.Display.UseColor = true
	
	return cfg
}

func getConfigFile(configName string, configDir ...string) string {
	var configDirPath string
	if len(configDir) > 0 {
		configDirPath = configDir[0]
	}

	var configFileList []string

	if runtime.GOOS == "windows" {
		userProfile := os.Getenv("USERPROFILE")
		appData := os.Getenv("APPDATA")
		
		configFileList = []string{
			filepath.Join(userProfile, configDirPath, "."+configName),
			
			filepath.Join(appData, configDirPath, configName+".ini"),
			filepath.Join(userProfile, configDirPath, configName+".ini"),
			
			filepath.Join(appData, configDirPath, configName+".toml"),
			filepath.Join(userProfile, configDirPath, configName+".toml"),
			
			filepath.Join(appData, configDirPath, configName+".json"),
			filepath.Join(userProfile, configDirPath, configName+".json"),
			
			filepath.Join(appData, configDirPath, configName+".yml"),
			filepath.Join(userProfile, configDirPath, configName+".yml"),
			
			filepath.Join(appData, configDirPath, filepath.Base(os.Args[0])+".ini"),
			filepath.Join(userProfile, configDirPath, filepath.Base(os.Args[0])+".ini"),
			
			filepath.Join(appData, configDirPath, filepath.Base(os.Args[0])+".toml"),
			filepath.Join(userProfile, configDirPath, filepath.Base(os.Args[0])+".toml"),
			
			filepath.Join(appData, configDirPath, filepath.Base(os.Args[0])+".json"),
			filepath.Join(userProfile, configDirPath, filepath.Base(os.Args[0])+".json"),
			
			filepath.Join(appData, configDirPath, filepath.Base(os.Args[0])+".yml"),
			filepath.Join(userProfile, configDirPath, filepath.Base(os.Args[0])+".yml"),
		}
	} else {
		home, _ := os.UserHomeDir()
		
		configFileList = []string{
			filepath.Join(home, configDirPath, "."+configName),
			filepath.Join(home, ".config", configDirPath, "."+configName),
			filepath.Join(home, ".config", "."+configName),
			
			filepath.Join(home, configDirPath, configName+".ini"),
			filepath.Join(home, ".config", configDirPath, configName+".ini"),
			filepath.Join(home, ".config", configName+".ini"),
			
			filepath.Join(home, configDirPath, configName+".toml"),
			filepath.Join(home, ".config", configDirPath, configName+".toml"),
			filepath.Join(home, ".config", configName+".toml"),
			
			filepath.Join(home, configDirPath, configName+".json"),
			filepath.Join(home, ".config", configDirPath, configName+".json"),
			filepath.Join(home, ".config", configName+".json"),
			
			filepath.Join(home, configDirPath, configName+".yml"),
			filepath.Join(home, ".config", configDirPath, configName+".yml"),
			filepath.Join(home, ".config", configName+".yml"),
			
			filepath.Join(home, configDirPath, filepath.Base(os.Args[0])+".ini"),
			filepath.Join(home, ".config", configDirPath, filepath.Base(os.Args[0])+".ini"),
			filepath.Join(home, ".config", filepath.Base(os.Args[0])+".ini"),
			
			filepath.Join(home, configDirPath, filepath.Base(os.Args[0])+".toml"),
			filepath.Join(home, ".config", configDirPath, filepath.Base(os.Args[0])+".toml"),
			filepath.Join(home, ".config", filepath.Base(os.Args[0])+".toml"),
			
			filepath.Join(home, configDirPath, filepath.Base(os.Args[0])+".json"),
			filepath.Join(home, ".config", configDirPath, filepath.Base(os.Args[0])+".json"),
			filepath.Join(home, ".config", filepath.Base(os.Args[0])+".json"),
			
			filepath.Join(home, configDirPath, filepath.Base(os.Args[0])+".yml"),
			filepath.Join(home, ".config", configDirPath, filepath.Base(os.Args[0])+".yml"),
			filepath.Join(home, ".config", filepath.Base(os.Args[0])+".yml"),
		}
	}

	var configFile string
	for _, cf := range configFileList {
		if fileInfo, err := os.Stat(cf); err == nil && !fileInfo.IsDir() {
			configFile = cf
			break
		}
	}

	if configFile != "" {
		dir := filepath.Dir(configFile)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}
	} else {
		execPath, _ := os.Executable()
		configFile = filepath.Join(filepath.Dir(execPath), "."+configName)
	}

	return configFile
}

// getMusicRootDefault returns platform-specific default music directory
func getMusicRootDefault() string {
	switch runtime.GOOS {
	case "windows":
		return "C:/Musics"
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Music")
	default: // linux and others
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Music")
	}
}

// getMPDConfigDefault returns platform-specific default MPD config path
func getMPDConfigDefault() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "mpd", "mpd.conf")
		}
		return "C:/mpd/mpd.conf"
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "mpd", "mpd.conf")
	default: // linux
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "mpd", "mpd.conf")
	}
}

// LoadConfig loads configuration from file with environment and CLI overrides
func LoadConfig(configPath string) (*Config, error) {
	cfg := NewConfig()
	
	// Try to load config file if provided
	fmt.Printf("Load config: '%s'\n", configPath)
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			// Use the new LoadConfigFromFile function that supports multiple formats
			if err := LoadConfigFromFile(configPath, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config: %v", err)
			}
		}
	}
	
	// Override with environment variables
	if val := os.Getenv("MPD_HOST"); val != "" {
		cfg.MPD.Host = val
	}
	if val := os.Getenv("MPD_PORT"); val != "" {
		cfg.MPD.Port = val
	}
	if val := os.Getenv("MPD_PASSWORD"); val != "" {
		cfg.MPD.Password = val
	}
	if val := os.Getenv("MPD_TIMEOUT"); val != "" {
		if timeout, err := strconv.Atoi(val); err == nil {
			cfg.MPD.Timeout = timeout
		}
	}
	if val := os.Getenv("MPD_MUSIC_ROOT"); val != "" {
		cfg.MPD.MusicRoot = val
	}
	
	return cfg, nil
}


// NewMPDClient creates a new MPD client with connection
func NewMPDClient(host, port, password string, cfg *Config) (*MPDClient, error) {
	client := &MPDClient{
		host:     host,
		port:     port,
		password: password,
		config:   cfg,
	}
	
	if err := client.connect(); err != nil {
		return nil, err
	}
	
	return client, nil
}

// connect establishes connection to MPD
func (m *MPDClient) connect() error {
	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	
	client, err := mpd.DialAuthenticated("tcp", addr, m.password)
	if err != nil {
		// fmt.Printf("Load config from [connect]: '%s'\n", configFile)
		return fmt.Errorf("failed to connect to MPD at %s: %v", addr, err)
	}
	
	m.client = client
	return nil
}

// reconnect attempts to reconnect with exponential backoff
func (m *MPDClient) reconnect() error {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
	
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		if err := m.connect(); err != nil {
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
			}
			continue
		}
		
		// Test connection
		if err := m.client.Ping(); err != nil {
			m.client.Close()
			m.client = nil
			continue
		}
		
		return nil
	}
	
	return fmt.Errorf("failed to reconnect after %d attempts", maxRetries)
}

// ensureConnected checks connection and reconnects if needed
func (m *MPDClient) ensureConnected() error {
	if m.client == nil {
		return m.connect()
	}
	
	if err := m.client.Ping(); err != nil {
		return m.reconnect()
	}
	
	return nil
}

// Close closes the MPD connection
func (m *MPDClient) Close() {
	if m.client != nil {
		m.client.Close()
	}
}

// normalizePath normalizes file paths for MPD
func (m *MPDClient) normalizePath(path string) string {
	if path == "" {
		return ""
	}
	
	// Convert to forward slashes
	path = filepath.ToSlash(path)
	
	// Get music root as forward slashes
	musicRoot := filepath.ToSlash(m.config.MPD.MusicRoot)
	
	// Trim trailing slashes from music root for consistent comparison
	musicRoot = strings.TrimSuffix(musicRoot, "/")
	
	// Case-insensitive comparison for Windows paths
	pathLower := strings.ToLower(path)
	musicRootLower := strings.ToLower(musicRoot)
	
	// If path starts with music root, make it relative
	if strings.HasPrefix(pathLower, musicRootLower+"/") {
		// Remove music root from path (use original case for the result)
		path = path[len(musicRoot)+1:]
	} else if pathLower == musicRootLower {
		path = ""
	} else if strings.Contains(pathLower, musicRootLower) {
		// Handle case where music root is embedded in path
		// Find the position in lowercase, extract from original
		idx := strings.Index(pathLower, musicRootLower)
		if idx >= 0 {
			path = path[idx+len(musicRoot):]
			path = strings.TrimPrefix(path, "/")
		}
	}
	
	// Remove Windows drive letters (C:, D:, etc.)
	if len(path) >= 2 && path[1] == ':' {
		path = path[2:]
	}
	
	// Clean up leading slashes
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "\\")
	
	// Replace any remaining backslashes with forward slashes
	path = strings.ReplaceAll(path, "\\", "/")
	
	return path
}

// Add adds a song or directory to the playlist
func (m *MPDClient) Add(path string) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	
	normalizedPath := m.normalizePath(path)
	return m.client.Add(normalizedPath)
}

// Delete removes songs matching the path from playlist
func (m *MPDClient) Delete(pathArg string) (int, error) {
	if err := m.ensureConnected(); err != nil {
		return 0, err
	}
	
	normalizedPath := m.normalizePath(pathArg)
	
	playlist, err := m.client.PlaylistInfo(-1, -1)
	if err != nil {
		return 0, err
	}
	
	deleted := 0
	for _, song := range playlist {
		songPath := m.normalizePath(song["file"])
		songDir := filepath.Dir(songPath)
		
		if songPath == normalizedPath || songDir == normalizedPath {
			if id, err := strconv.Atoi(song["Id"]); err == nil {
				if err := m.client.DeleteID(id); err == nil {
					deleted++
				}
			}
		}
	}
	
	return deleted, nil
}

// Clear clears the current playlist
func (m *MPDClient) Clear() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Clear()
}

// Play starts playback
func (m *MPDClient) Play(pos int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	
	if pos >= 0 {
		// Play specific position
		return m.client.Play(pos)
	}
	
	// pos is -1: Let MPD decide what to play
	// MPD will resume at current position if paused/stopped
	// or start from position 0 if nothing was ever played
	return m.client.Play(-1)
}

// Pause toggles pause
func (m *MPDClient) Pause() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	
	status, err := m.client.Status()
	if err != nil {
		return err
	}
	
	isPaused := status["state"] == "pause"
	return m.client.Pause(!isPaused)
}

// Stop stops playback
func (m *MPDClient) Stop() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Stop()
}

// Next plays next song
func (m *MPDClient) Next() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Next()
}

// Previous plays previous song
func (m *MPDClient) Previous() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Previous()
}

// Seek seeks within current song
// seconds can be a float (e.g., 3.5 for 3.5 seconds)
func (m *MPDClient) Seek(seconds float64, relative bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	
	// Convert float64 seconds to time.Duration
	duration := time.Duration(seconds * float64(time.Second))
	return m.client.SeekCur(duration, relative)
}

// Volume sets or gets volume
func (m *MPDClient) Volume(vol int) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.SetVolume(vol)
}

// Random toggles random mode
func (m *MPDClient) Random(enable bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Random(enable)
}

// Repeat toggles repeat mode
func (m *MPDClient) Repeat(enable bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Repeat(enable)
}

// Single toggles single mode
func (m *MPDClient) Single(enable bool) error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Single(enable)
}

// Shuffle shuffles the current playlist
func (m *MPDClient) Shuffle() error {
	if err := m.ensureConnected(); err != nil {
		return err
	}
	return m.client.Shuffle(-1, -1)
}

// Update updates the music database
func (m *MPDClient) Update(path string) (int, error) {
	if err := m.ensureConnected(); err != nil {
		return 0, err
	}
	return m.client.Update(path)
}

// Status returns current player status
func (m *MPDClient) Status() (mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.Status()
}

// CurrentSong returns the currently playing song
func (m *MPDClient) CurrentSong() (mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.CurrentSong()
}

// PlaylistInfo returns playlist information
func (m *MPDClient) PlaylistInfo() ([]mpd.Attrs, error) {
	if err := m.ensureConnected(); err != nil {
		return nil, err
	}
	return m.client.PlaylistInfo(-1, -1)
}

// GetConfig reads MPD config value (local MPD only)
func (m *MPDClient) GetConfig(key string) (string, error) {
	if m.config.MPD.ConfigPath == "" {
		return "", fmt.Errorf("MPD config path not set")
	}
	
	data, err := os.ReadFile(m.config.MPD.ConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read MPD config: %v", err)
	}
	
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[0] == key {
			return strings.Trim(strings.Join(parts[1:], " "), "\""), nil
		}
	}
	
	return "", fmt.Errorf("config key '%s' not found", key)
}

// SetConfig writes MPD config value (local MPD only)
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
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		
		parts := strings.Fields(trimmed)
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

// getTerminalWidth returns the terminal width
func getTerminalWidth() int {
	fd := int(os.Stdout.Fd())
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

// printSeparator prints a separator line
func printSeparator() {
	width := getTerminalWidth()
	fmt.Println(strings.Repeat("─", width))
}

// formatDuration formats duration in seconds to MM:SS
func formatDuration(seconds string) string {
	if seconds == "" {
		return "0:00"
	}
	
	sec, err := strconv.ParseFloat(seconds, 64)
	if err != nil {
		return "0:00"
	}
	
	mins := int(sec) / 60
	secs := int(sec) % 60
	
	return fmt.Sprintf("%d:%02d", mins, secs)
}

// formatBitrate formats bitrate information
func formatBitrate(attrs mpd.Attrs) string {
	if bitrate, ok := attrs["audio"]; ok {
		parts := strings.Split(bitrate, ":")
		if len(parts) >= 1 {
			if sr, err := strconv.Atoi(parts[0]); err == nil {
				kbps := sr / 1000
				return fmt.Sprintf("%d kHz", kbps)
			}
		}
	}
	
	if bitrate, ok := attrs["bitrate"]; ok {
		return fmt.Sprintf("%s kbps", bitrate)
	}
	
	return "N/A"
}

// renderPlayingBanner renders the currently playing banner
func renderPlayingBanner(index string, song mpd.Attrs) string {
	title := getOrDefault(song, "Title", "Unknown")
	artist := getOrDefault(song, "Artist", "Unknown")
	album := getOrDefault(song, "Album", "Unknown")
	
	result := fmt.Sprintf("%s %sPLAYING:%s %s %s%s%s. %s%s%s - %s%s%s - %s%s%s",
		PlayingBg, PlayingLabel, Reset,
		Icon, FgIndex, index, Reset,
		FgTitle, title, Reset,
		FgArtist, artist, Reset,
		FgAlbum, album, Reset)
	
	if date, ok := song["Date"]; ok && date != "" {
		result += fmt.Sprintf(" %s(%s)%s", FgDate, date, Reset)
	}
	
	return result
}

// renderPlaylist renders the playlist
func renderPlaylist(client *MPDClient, messages []string) error {
	// fmt.Printf("Load config from: '%s'", configFile)
	playlist, err := client.PlaylistInfo()
	if err != nil {
		return err
	}
	
	status, err := client.Status()
	if err != nil {
		return err
	}
	
	currentSongID := status["songid"]
	pad := len(strconv.Itoa(len(playlist)))
	
	var lines []string
	var currentPlaying *struct {
		index string
		song  mpd.Attrs
	}
	
	// Clear screen
	fmt.Print("\033c")
	
	for idx, song := range playlist {
		idxNum := idx + 1
		idxStr := fmt.Sprintf("%0*d", pad, idxNum)
		
		title := getOrDefault(song, "Title", "Unknown")
		artist := getOrDefault(song, "Artist", "Unknown")
		album := getOrDefault(song, "Album", "Unknown")
		
		line := fmt.Sprintf("%s %s%s%s. %s%s%s - %s%s%s - %s%s%s",
			Icon, FgIndex, idxStr, Reset,
			FgTitle, title, Reset,
			FgArtist, artist, Reset,
			FgAlbum, album, Reset)
		
		if date, ok := song["Date"]; ok && date != "" {
			line += fmt.Sprintf(" %s(%s)%s", FgDate, date, Reset)
		}
		
		if song["Id"] == currentSongID {
			line = fmt.Sprintf("%s%s%s", BgBlue, line, Reset)
			currentPlaying = &struct {
				index string
				song  mpd.Attrs
			}{idxStr, song}
		}
		
		lines = append(lines, line)
	}
	
	fmt.Println(strings.Join(lines, "\n"))
	
	if currentPlaying != nil {
		fmt.Println("\n" + renderPlayingBanner(currentPlaying.index, currentPlaying.song))
	}
	
	if len(messages) > 0 {
		fmt.Println("\n  " + strings.Join(messages, "\n  "))
	}
	
	return nil
}

// printStatus prints current status
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
	volume := status["volume"]
	repeat := status["repeat"] == "1"
	random := status["random"] == "1"
	single := status["single"] == "1"
	
	fmt.Printf("State: %s\n", state)
	fmt.Printf("Volume: %s%%\n", volume)
	fmt.Printf("Repeat: %v | Random: %v | Single: %v\n", repeat, random, single)
	
	if state != "stop" {
		title := getOrDefault(song, "Title", song["file"])
		artist := getOrDefault(song, "Artist", "Unknown")
		album := getOrDefault(song, "Album", "Unknown")
		elapsed := formatDuration(status["elapsed"])
		duration := formatDuration(song["duration"])
		
		fmt.Printf("\nPlaying: %s\n", title)
		fmt.Printf("Artist: %s\n", artist)
		fmt.Printf("Album: %s\n", album)
		fmt.Printf("Time: %s / %s\n", elapsed, duration)
	}
	
	return nil
}

// getOrDefault gets value from map with default
func getOrDefault(m map[string]string, key, def string) string {
	if val, ok := m[key]; ok && val != "" {
		return val
	}
	return def
}

// setupGNTP initializes GNTP client
func setupGNTP(cfg *Config, debug bool) (*gntp.Client, bool) {
	if !cfg.GNTP.Enabled {
		return nil, false
	}
	
	client := gntp.NewClient("MPD Monitor").
		WithHost(cfg.GNTP.Host).
		WithPort(cfg.GNTP.Port).
		WithTimeout(10 * time.Second)
	
	switch strings.ToLower(cfg.GNTP.IconMode) {
	case "dataurl":
		client.WithIconMode(gntp.IconModeDataURL)
	case "fileurl":
		client.WithIconMode(gntp.IconModeFileURL)
	case "httpurl":
		client.WithIconMode(gntp.IconModeHttpURL)
	default:
		client.WithIconMode(gntp.IconModeBinary)
	}
	
	songChange := gntp.NewNotificationType("song_change").
		WithDisplayName("Song Changed")
	
	playerState := gntp.NewNotificationType("player_state").
		WithDisplayName("Player State")
	
	if err := client.Register([]*gntp.NotificationType{songChange, playerState}); err != nil {
		if debug {
			log.Printf("⚠️  Failed to register with GNTP: %v", err)
		}
		return nil, false
	}
	
	return client, true
}

// getAlbumArt retrieves album artwork
func getAlbumArt(client *mpd.Client, uri string) *gntp.Resource {
	artwork, err := client.ReadPicture(uri)
	if err == nil && len(artwork) > 0 {
		contentType := "image/jpeg"
		if len(artwork) > 8 {
			if artwork[0] == 0x89 && artwork[1] == 0x50 && artwork[2] == 0x4E && artwork[3] == 0x47 {
				contentType = "image/png"
			}
		}
		return gntp.LoadResourceFromBytes(artwork, contentType)
	}
	
	artwork, err = client.AlbumArt(uri)
	if err == nil && len(artwork) > 0 {
		contentType := "image/jpeg"
		if len(artwork) > 8 {
			if artwork[0] == 0x89 && artwork[1] == 0x50 && artwork[2] == 0x4E && artwork[3] == 0x47 {
				contentType = "image/png"
			}
		}
		return gntp.LoadResourceFromBytes(artwork, contentType)
	}
	
	return nil
}

// formatNotificationMessage formats notification message
func formatNotificationMessage(song mpd.Attrs, status mpd.Attrs) string {
	pos := status["song"]
	total := status["playlistlength"]
	elapsed := formatDuration(status["elapsed"])
	duration := formatDuration(song["duration"])
	track := getOrDefault(song, "Track", "?")
	title := getOrDefault(song, "Title", song["file"])
	artist := getOrDefault(song, "Artist", "")
	album := getOrDefault(song, "Album", "")
	bitrate := formatBitrate(status)
	filepath := song["file"]
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s/%s/%s. %s\n", pos, total, track, title))
	sb.WriteString(fmt.Sprintf("%s / %s\n", elapsed, duration))
	
	if artist != "" {
		sb.WriteString(fmt.Sprintf("🎤 %s\n", artist))
	}
	
	if album != "" {
		sb.WriteString(fmt.Sprintf("💿 %s\n", album))
	}
	
	sb.WriteString(fmt.Sprintf("🎵 %s\n", bitrate))
	sb.WriteString(fmt.Sprintf("📁 %s", filepath))
	
	return sb.String()
}

// formatConsoleMessage formats console message
func formatConsoleMessage(song mpd.Attrs, status mpd.Attrs) string {
	pos := status["song"]
	total := status["playlistlength"]
	elapsed := formatDuration(status["elapsed"])
	duration := formatDuration(song["duration"])
	track := getOrDefault(song, "Track", "?")
	title := getOrDefault(song, "Title", song["file"])
	artist := getOrDefault(song, "Artist", "")
	album := getOrDefault(song, "Album", "")
	bitrate := formatBitrate(status)
	filepath := song["file"]
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s▶ %s/%s/%s. %s%s\n", ColorCyan, pos, total, track, title, Reset))
	sb.WriteString(fmt.Sprintf("%s  🕓 %s / %s%s\n", ColorCyan, elapsed, duration, Reset))
	
	if artist != "" {
		sb.WriteString(fmt.Sprintf("%s  🎤 %s%s\n", ColorYellow, artist, Reset))
	}
	
	if album != "" {
		sb.WriteString(fmt.Sprintf("%s  💿 %s%s\n", ColorOrange, album, Reset))
	}
	
	sb.WriteString(fmt.Sprintf("%s  🎵 %s%s\n", ColorBlue, bitrate, Reset))
	sb.WriteString(fmt.Sprintf("%s  📁 %s%s", ColorGreen, filepath, Reset))
	
	return sb.String()
}

// sendNotification sends GNTP notification
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

// checkStatus checks MPD status and sends notifications
func checkStatus(state *AppState) error {
	if err := state.client.client.Ping(); err != nil {
		return fmt.Errorf("connection lost: %v", err)
	}
	
	status, err := state.client.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %v", err)
	}
	
	currentState := status["state"]
	
	song, err := state.client.CurrentSong()
	if err != nil {
		return fmt.Errorf("failed to get current song: %v", err)
	}
	
	currentFile := song["file"]
	
	songChanged := currentFile != state.lastSongFile && currentFile != ""
	stateChanged := currentState != state.lastState && state.lastState != ""
	
	if currentState == "play" && currentFile != "" {
		info := formatConsoleMessage(song, status)
		fmt.Println()
		fmt.Println(info)
		printSeparator()
	} else if stateChanged {
		fmt.Printf("⏸  State: %s\n", currentState)
		printSeparator()
	}
	
	if songChanged && currentState == "play" {
		artwork := getAlbumArt(state.client.client, currentFile)
		title := getOrDefault(song, "Title", currentFile)
		message := formatNotificationMessage(song, status)
		
		if err := sendNotification(state, "song_change", title, message, artwork); err != nil {
			if state.debug {
				log.Printf("⚠️  Failed to send notification: %v", err)
			}
		}
		
		state.lastSongFile = currentFile
	}
	
	if stateChanged {
		var stateMsg string
		switch currentState {
		case "play":
			stateMsg = "▶ Playing"
		case "pause":
			stateMsg = "⏸ Paused"
		case "stop":
			stateMsg = "⏹ Stopped"
		default:
			stateMsg = fmt.Sprintf("State: %s", currentState)
		}
		
		var artwork *gntp.Resource
		if currentFile != "" {
			artwork = getAlbumArt(state.client.client, currentFile)
		}
		
		message := stateMsg
		if currentState == "play" && currentFile != "" {
			message = formatNotificationMessage(song, status)
		}
		
		if err := sendNotification(state, "player_state", stateMsg, message, artwork); err != nil {
			if state.debug {
				log.Printf("⚠️  Failed to send notification: %v", err)
			}
		}
	}
	
	state.lastState = currentState
	return nil
}

// monitorOnce runs one monitoring cycle
func monitorOnce(state *AppState) error {
	w, err := mpd.NewWatcher("tcp",
		fmt.Sprintf("%s:%s", state.config.MPD.Host, state.config.MPD.Port),
		state.config.MPD.Password, "player", "mixer")
	if err != nil {
		return fmt.Errorf("failed to create watcher: %v", err)
	}
	
	done := make(chan struct{})
	defer close(done)
	
	go func() {
		defer func() {
			if r := recover(); r != nil && state.debug {
				log.Printf("🛡️  Recovered from panic in error monitor: %v", r)
			}
		}()
		
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
				return fmt.Errorf("watcher event channel closed")
			}
			
			if subsystem == "database" || subsystem == "update" {
				continue
			}
			
			if err := checkStatus(state); err != nil {
				if state.debug {
					log.Printf("⚠️  Status check failed: %v", err)
				}
				
				if strings.Contains(err.Error(), "EOF") ||
					strings.Contains(err.Error(), "connection") ||
					strings.Contains(err.Error(), "broken pipe") {
					w.Close()
					return err
				}
			}
			
		case <-done:
			w.Close()
			return nil
			
		case <-time.After(30 * time.Second):
			if err := state.client.client.Ping(); err != nil {
				if state.debug {
					log.Printf("⚠️  Ping failed: %v", err)
				}
				w.Close()
				return fmt.Errorf("ping failed: %v", err)
			}
		}
	}
}

// runMonitor runs the monitoring loop
func runMonitor(state *AppState) error {
	log.Println("🎵 MPD Monitor started")
	log.Printf("📡 Monitoring: %s:%s", state.config.MPD.Host, state.config.MPD.Port)
	
	if state.gntpEnabled {
		log.Printf("📢 GNTP Server: %s:%d", state.config.GNTP.Host, state.config.GNTP.Port)
		log.Printf("✅ GNTP registered (icon mode: %s)", state.config.GNTP.IconMode)
	} else {
		log.Println("📢 GNTP/Growl notifications: disabled")
	}
	
	if state.debug {
		log.Println("🐛 Debug mode: enabled")
	}
	
	fmt.Println(strings.Repeat("=", getTerminalWidth()))
	
	if err := checkStatus(state); err != nil {
		if state.debug {
			log.Printf("⚠️  Initial status check failed: %v", err)
		}
	}
	
	for {
		err := monitorOnce(state)
		if err != nil {
			if state.debug {
				log.Printf("❌ Monitor error: %v", err)
			}
			
			if strings.Contains(err.Error(), "EOF") ||
				strings.Contains(err.Error(), "connection") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "watcher") {
				
				if state.debug {
					log.Println("🔄 Attempting to reconnect to MPD...")
				}
				
				time.Sleep(2 * time.Second)
				
				if err := state.client.reconnect(); err != nil {
					if state.debug {
						log.Printf("❌ Reconnect failed: %v", err)
					}
					time.Sleep(5 * time.Second)
					continue
				}
				
				if state.debug {
					log.Println("✅ Reconnected to MPD")
				}
				
				continue
			}
			
			return err
		}
		
		if state.debug {
			log.Println("📡 Connection lost, attempting to reconnect...")
		}
		time.Sleep(2 * time.Second)
	}
}

// printHelp prints help message
func printHelp() {
	fmt.Printf(`mpdl - Advanced MPD CLI client v%s

USAGE:
    mpdl [OPTIONS] [COMMAND] [ARGS...]

OPTIONS:
    -c, --config PATH       Configuration file path (TOML format)
    --mpd-host HOST        MPD server host (default: localhost)
    --mpd-port PORT        MPD server port (default: 6600)
    --mpd-password PASS    MPD password
    --music-root PATH      Music root directory
    --mpd-config PATH      MPD config file path (for get-config/set-config)
    --debug                Enable debug mode
    -h, --help             Show this help message
    -v, --version          Show version information

PLAYBACK COMMANDS:
    play [POS]             Start playback (optionally at position)
    pause                  Toggle pause
    stop                   Stop playback
    next, n                Next song
    prev, previous, p      Previous song
    seek [+/-]SECONDS      Seek to or by seconds (supports decimals)
                           Examples: seek 30.5, seek +10, seek -5.2

PLAYLIST COMMANDS:
    add PATH               Add song/directory to playlist
    del, delete PATH       Delete songs matching PATH from playlist
    clear                  Clear playlist
    list, ls, playlist     Show current playlist
    shuffle                Shuffle playlist

QUEUE COMMANDS:
    current                Show current song
    status, st             Show player status

OPTIONS COMMANDS:
    volume [VOL]           Set volume (0-100) or show current
    repeat [on|off|1|0]    Toggle or set repeat mode
    random [on|off|1|0]    Toggle or set random mode
    single [on|off|1|0]    Toggle or set single mode

DATABASE COMMANDS:
    update [PATH]          Update music database

CONFIG COMMANDS (local MPD only):
    get-config KEY         Get MPD config value
    set-config KEY VALUE   Set MPD config value

MONITOR MODE:
    monitor, mon, m        Monitor MPD with notifications
      Keyboard shortcuts in monitor mode:
        p     - Play/Pause
        s     - Stop
        n     - Next song
        b     - Previous song
        q     - Quit monitor

ENVIRONMENT VARIABLES:
    MPD_HOST              MPD server host
    MPD_PORT              MPD server port
    MPD_PASSWORD          MPD password
    MPD_TIMEOUT           Connection timeout in seconds
    MPD_MUSIC_ROOT        Music root directory
    DEBUG                 Enable debug mode (set to 1)

CONFIG FILE EXAMPLE (TOML):
    [mpd]
    host = "localhost"
    port = "6600"
    password = ""
    timeout = 10
    music_root = "C:/Musics"  # or ~/Music on Linux/macOS
    config_path = "C:/mpd/mpd.conf"
    
    [gntp]
    host = "localhost"
    port = 23053
    password = ""
    icon_mode = "binary"  # binary, dataurl, fileurl, httpurl
    enabled = true
    
    [display]
    show_album_art = true
    use_color = true

EXAMPLES:
    mpdl play              # Start playback
    mpdl add ~/Music/song.mp3
    mpdl del "Artist/Album"
    mpdl volume 75
    mpdl status
    mpdl monitor           # Start monitoring mode
    mpdl get-config music_directory
    mpdl set-config volume_normalization yes

For more information, visit: https://github.com/cumulus13/mpdl
`, Version)
}

func main() {
	// Command-line flags
	
	flag.StringVar(&configFile, "config", "", "Configuration file path")
	flag.StringVar(&configFile, "c", "", "Configuration file path (short)")
	flag.StringVar(&mpdHost, "mpd-host", "", "MPD host")
	flag.StringVar(&mpdPort, "mpd-port", "", "MPD port")
	flag.StringVar(&mpdPassword, "mpd-password", "", "MPD password")
	flag.StringVar(&musicRoot, "music-root", "", "Music root directory")
	flag.StringVar(&mpdConfigPath, "mpd-config", "", "MPD config file path")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (short)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help (short)")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	
	flag.Parse()
	
	if showVersion {
		fmt.Printf("mpdl version %s\n", Version)
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}
	
	if showHelp || (flag.NArg() == 0 && !debug) {
		printHelp()
		os.Exit(0)
	}
	
	// Check DEBUG environment
	if os.Getenv("DEBUG") == "1" {
		debug = true
	}
	
	fmt.Printf("configFile: %s\n", configFile)

	if configFile == "" {
		// fmt.Println("Try to load config ...")
		configFile = getConfigFile("mpdl")
		// fmt.Printf("configFile: %s\n", configFile)
	} //else {
	// 	fmt.Printf("Load config from: '%s'", configFile)
	// }
	
	// Load configuration
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}
	
	// Override with CLI flags
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
	
	// Create MPD client
	client, err := NewMPDClient(config.MPD.Host, config.MPD.Port, config.MPD.Password, config)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	defer client.Close()
	
	// Parse command
	args := flag.Args()
	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}
	
	command := strings.ToLower(args[0])
	commandArgs := args[1:]
	
	var messages []string
	
	// Execute command
	switch command {
	case "play":
		if len(commandArgs) > 0 {
			// Play specific position
			pos, _ := strconv.Atoi(commandArgs[0])
			if err := client.Play(pos - 1); err != nil {
				log.Fatalf("❌ Failed to play: %v", err)
			}
		} else {
			// No position specified - resume/unpause
			status, err := client.Status()
			if err != nil {
				log.Fatalf("❌ Failed to get status: %v", err)
			}
			
			if status["state"] == "pause" {
				// If paused, unpause
				if err := client.Pause(); err != nil {
					log.Fatalf("❌ Failed to resume: %v", err)
				}
			} else {
				// If stopped, start playing
				if err := client.Play(-1); err != nil {
					log.Fatalf("❌ Failed to play: %v", err)
				}
			}
		}
		fmt.Println("▶ Playing")
		
	case "pause":
		if err := client.Pause(); err != nil {
			log.Fatalf("❌ Failed to pause: %v", err)
		}
		fmt.Println("⏸ Paused")
		
	case "stop":
		if err := client.Stop(); err != nil {
			log.Fatalf("❌ Failed to stop: %v", err)
		}
		fmt.Println("⏹ Stopped")
		
	case "next", "n":
		if err := client.Next(); err != nil {
			log.Fatalf("❌ Failed to skip: %v", err)
		}
		fmt.Println("⏭ Next")
		
	case "prev", "previous", "p":
		if err := client.Previous(); err != nil {
			log.Fatalf("❌ Failed to go back: %v", err)
		}
		fmt.Println("⏮ Previous")
		
	case "seek":
		if len(commandArgs) == 0 {
			log.Fatal("❌ Usage: mpdl seek [+/-]SECONDS (e.g., 30, +10, -5)")
		}
		
		seekStr := commandArgs[0]
		relative := false
		
		// Check if it's relative (starts with + or -)
		if strings.HasPrefix(seekStr, "+") || strings.HasPrefix(seekStr, "-") {
			relative = true
		}
		
		// Parse the seek value
		seekValue, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimPrefix(seekStr, "+"), "-"), 64)
		if err != nil {
			log.Fatalf("❌ Invalid seek value: %s", seekStr)
		}
		
		// Apply negative sign if present
		if strings.HasPrefix(seekStr, "-") {
			seekValue = -seekValue
		}
		
		if err := client.Seek(seekValue, relative); err != nil {
			log.Fatalf("❌ Failed to seek: %v", err)
		}
		
		if relative {
			if seekValue > 0 {
				fmt.Printf("⏩ Seek forward %.1f seconds\n", seekValue)
			} else {
				fmt.Printf("⏪ Seek backward %.1f seconds\n", -seekValue)
			}
		} else {
			fmt.Printf("⏩ Seek to %.1f seconds\n", seekValue)
		}
		
	case "add":
		if len(commandArgs) == 0 {
			log.Fatal("❌ Usage: mpdl add PATH")
		}
		
		// FIXED: Join all args to handle paths with spaces
		path := strings.Join(commandArgs, " ")
		// Remove quotes if present
		path = strings.Trim(path, "\"'")
		
		// Only expand to absolute path if it looks like a local file reference
		// (starts with . or / or drive letter)
		if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") || 
		   strings.HasPrefix(path, "\\") || (len(path) >= 2 && path[1] == ':') {
			// This looks like a local absolute/relative path
			if filepath.Dir(path) == "." || strings.HasPrefix(path, ".") {
				cwd, _ := os.Getwd()
				path = filepath.Join(cwd, path)
			}
			path, _ = filepath.Abs(path)
		}
		// Otherwise, assume it's already a path relative to MPD's music_directory

		
		if err := client.Add(path); err != nil {
			messages = append(messages, fmt.Sprintf("\033[37;41mFailed to add: '%s'%s", path, Reset))
			messages = append(messages, fmt.Sprintf("\033[37;41mError: %v%s", err, Reset))
		} else {
			messages = append(messages, fmt.Sprintf("\033[1;30;93mSuccessfully added: '%s'%s", path, Reset))
		}
		
		if err := renderPlaylist(client, messages); err != nil {
			log.Fatalf("❌ %v", err)
		}
		
	case "del", "delete":
		if len(commandArgs) == 0 {
			log.Fatal("❌ Usage: mpdl delete PATH")
		}
		
		// FIXED: Join all args to handle paths with spaces
		path := strings.Join(commandArgs, " ")
		// Remove quotes if present
		path = strings.Trim(path, "\"'")
		
		deleted, err := client.Delete(path)
		if err != nil || deleted == 0 {
			messages = append(messages, fmt.Sprintf("\033[37;41mFailed to delete: '%s'%s", path, Reset))
			if err != nil {
				messages = append(messages, fmt.Sprintf("\033[37;41mError: %v%s", err, Reset))
			}
		} else {
			messages = append(messages, fmt.Sprintf("\033[1;30;105mSuccessfully deleted %d song(s): '%s'%s", deleted, path, Reset))
		}
		
		if err := renderPlaylist(client, messages); err != nil {
			log.Fatalf("❌ %v", err)
		}
		
	case "clear":
		if err := client.Clear(); err != nil {
			log.Fatalf("❌ Failed to clear playlist: %v", err)
		}
		fmt.Println("🗑 Playlist cleared")
		
	case "shuffle":
		if err := client.Shuffle(); err != nil {
			log.Fatalf("❌ Failed to shuffle playlist: %v", err)
		}
		fmt.Println("🔀 Playlist shuffled")
		
	case "list", "ls", "playlist":
		if err := renderPlaylist(client, messages); err != nil {
			log.Fatalf("❌ %v", err)
			// fmt.Printf("Load config from [ls]: '%s'", configFile)
		}
		
	case "current":
		song, err := client.CurrentSong()
		if err != nil {
			log.Fatalf("❌ Failed to get current song: %v", err)
		}
		
		status, err := client.Status()
		if err != nil {
			log.Fatalf("❌ Failed to get status: %v", err)
		}
		
		fmt.Println(formatConsoleMessage(song, status))
		
	case "status", "st":
		if err := printStatus(client); err != nil {
			log.Fatalf("❌ %v", err)
		}
		
	case "volume":
		if len(commandArgs) == 0 {
			status, err := client.Status()
			if err != nil {
				log.Fatalf("❌ Failed to get status: %v", err)
			}
			fmt.Printf("Volume: %s%%\n", status["volume"])
		} else {
			vol, err := strconv.Atoi(commandArgs[0])
			if err != nil || vol < 0 || vol > 100 {
				log.Fatal("❌ Volume must be between 0 and 100")
			}
			if err := client.Volume(vol); err != nil {
				log.Fatalf("❌ Failed to set volume: %v", err)
			}
			fmt.Printf("🔊 Volume set to %d%%\n", vol)
		}
		
	case "repeat":
		if len(commandArgs) == 0 {
			status, err := client.Status()
			if err != nil {
				log.Fatalf("❌ Failed to get status: %v", err)
			}
			enable := status["repeat"] != "1"
			if err := client.Repeat(enable); err != nil {
				log.Fatalf("❌ Failed to set repeat: %v", err)
			}
			fmt.Printf("🔁 Repeat: %v\n", enable)
		} else {
			enable := commandArgs[0] == "on" || commandArgs[0] == "1"
			if err := client.Repeat(enable); err != nil {
				log.Fatalf("❌ Failed to set repeat: %v", err)
			}
			fmt.Printf("🔁 Repeat: %v\n", enable)
		}
		
	case "random":
		if len(commandArgs) == 0 {
			status, err := client.Status()
			if err != nil {
				log.Fatalf("❌ Failed to get status: %v", err)
			}
			enable := status["random"] != "1"
			if err := client.Random(enable); err != nil {
				log.Fatalf("❌ Failed to set random: %v", err)
			}
			fmt.Printf("🔀 Random: %v\n", enable)
		} else {
			enable := commandArgs[0] == "on" || commandArgs[0] == "1"
			if err := client.Random(enable); err != nil {
				log.Fatalf("❌ Failed to set random: %v", err)
			}
			fmt.Printf("🔀 Random: %v\n", enable)
		}
		
	case "single":
		if len(commandArgs) == 0 {
			status, err := client.Status()
			if err != nil {
				log.Fatalf("❌ Failed to get status: %v", err)
			}
			enable := status["single"] != "1"
			if err := client.Single(enable); err != nil {
				log.Fatalf("❌ Failed to set single: %v", err)
			}
			fmt.Printf("🔂 Single: %v\n", enable)
		} else {
			enable := commandArgs[0] == "on" || commandArgs[0] == "1"
			if err := client.Single(enable); err != nil {
				log.Fatalf("❌ Failed to set single: %v", err)
			}
			fmt.Printf("🔂 Single: %v\n", enable)
		}
		
	case "update":
		path := ""
		if len(commandArgs) > 0 {
			// FIXED: Join all args to handle paths with spaces
			path = strings.Join(commandArgs, " ")
			// Remove quotes if present
			path = strings.Trim(path, "\"'")
		}
		jobID, err := client.Update(path)
		if err != nil {
			log.Fatalf("❌ Failed to update database: %v", err)
		}
		fmt.Printf("🔄 Database update started (job %d)\n", jobID)
		
	case "get-config":
		if len(commandArgs) == 0 {
			log.Fatal("❌ Usage: mpdl get-config KEY")
		}
		value, err := client.GetConfig(commandArgs[0])
		if err != nil {
			log.Fatalf("❌ %v", err)
		}
		fmt.Printf("%s = %s\n", commandArgs[0], value)
		
	case "set-config":
		if len(commandArgs) < 2 {
			log.Fatal("❌ Usage: mpdl set-config KEY VALUE")
		}
		key := commandArgs[0]
		value := strings.Join(commandArgs[1:], " ")
		
		if err := client.SetConfig(key, value); err != nil {
			log.Fatalf("❌ %v", err)
		}
		fmt.Printf("✅ Set %s = %s\n", key, value)
		fmt.Println("⚠️  Note: You may need to restart MPD for changes to take effect")
		
	case "monitor", "mon", "m":
		gntpClient, gntpEnabled := setupGNTP(config, debug)
		
		state := &AppState{
			client:      client,
			gntp:        gntpClient,
			config:      config,
			debug:       debug,
			gntpEnabled: gntpEnabled,
		}
		
		if err := runMonitor(state); err != nil {
			log.Fatalf("❌ Monitor error: %v", err)
		}
		
	default:
		fmt.Printf("❌ Unknown command: %s\n", command)
		fmt.Println("Run 'mpdl --help' for usage information")
		os.Exit(1)
	}

	// fmt.Printf("Load config from [end]: '%s'", configFile)
}
