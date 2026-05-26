// File: config_loader.go
// Dynamic multi-format configuration loader
// Supports: .toml, .json, .yaml/.yml, .ini, .env (and auto-detection)
// License: MIT

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ConfigFormat identifies the file format.
type ConfigFormat int

const (
	FormatUnknown ConfigFormat = iota
	FormatTOML
	FormatJSON
	FormatYAML
	FormatINI
	FormatENV
)

// detectFormat returns the format based on the file extension, falling back to
// content sniffing for extension-less files (e.g. ~/.mpdl).
func detectFormat(filename string) ConfigFormat {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".toml":
		return FormatTOML
	case ".json":
		return FormatJSON
	case ".yaml", ".yml":
		return FormatYAML
	case ".ini":
		return FormatINI
	case ".env":
		return FormatENV
	case "":
		return detectFormatByContent(filename)
	default:
		return FormatUnknown
	}
}

func detectFormatByContent(filename string) ConfigFormat {
	data, err := os.ReadFile(filename)
	if err != nil {
		return FormatUnknown
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return FormatUnknown
	}

	if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[{") {
		return FormatJSON
	}

	lines := strings.Split(content, "\n")
	envLines := 0
	sectionLines := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, ";") {
			continue
		}
		if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
			sectionLines++
		}
		if strings.Contains(l, "=") && !strings.HasPrefix(l, "[") {
			envLines++
		}
	}

	if sectionLines > 0 {
		// Could be INI or TOML
		if strings.Contains(content, "---") {
			return FormatYAML
		}
		return FormatTOML
	}
	if envLines > 0 {
		return FormatENV
	}
	if strings.Contains(content, ": ") && !strings.Contains(content, "=") {
		return FormatYAML
	}

	return FormatUnknown
}

// LoadConfigFromFile loads and merges a config file into cfg.
func LoadConfigFromFile(filename string, cfg *Config) error {
	if filename == "" {
		return fmt.Errorf("empty filename")
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", filename)
	}

	format := detectFormat(filename)

	switch format {
	case FormatTOML:
		return loadTOML(filename, cfg)
	case FormatJSON:
		return loadJSON(filename, cfg)
	case FormatYAML:
		return loadYAML(filename, cfg)
	case FormatINI:
		return loadINI(filename, cfg)
	case FormatENV:
		return loadENV(filename, cfg)
	default:
		// Try each format in order of likelihood
		for _, fn := range []func(string, *Config) error{loadENV, loadINI, loadTOML, loadYAML, loadJSON} {
			if err := fn(filename, cfg); err == nil {
				return nil
			}
		}
		return fmt.Errorf("could not parse config file %q (tried all formats)", filename)
	}
}

// ── Format loaders ──────────────────────────────────────────────────────────

func loadTOML(filename string, cfg *Config) error {
	_, err := toml.DecodeFile(filename, cfg)
	return err
}

func loadJSON(filename string, cfg *Config) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

func loadYAML(filename string, cfg *Config) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

func loadINI(filename string, cfg *Config) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	section := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		applyConfigValue(cfg, section, key, val)
	}
	return scanner.Err()
}

func loadENV(filename string, cfg *Config) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		applyEnvStyleKey(cfg, key, val)
	}
	return scanner.Err()
}

// ── Key appliers ────────────────────────────────────────────────────────────

func applyEnvStyleKey(cfg *Config, key, value string) {
	switch strings.ToUpper(key) {
	case "MPD_HOST":
		cfg.MPD.Host = value
	case "MPD_PORT":
		cfg.MPD.Port = value
	case "MPD_PASSWORD":
		cfg.MPD.Password = value
	case "MPD_TIMEOUT":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.MPD.Timeout = v
		}
	case "MPD_MUSIC_ROOT":
		cfg.MPD.MusicRoot = value
	case "MPD_CONFIG_PATH":
		cfg.MPD.ConfigPath = value
	case "GNTP_HOST":
		cfg.GNTP.Host = value
	case "GNTP_PORT":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.GNTP.Port = v
		}
	case "GNTP_PASSWORD":
		cfg.GNTP.Password = value
	case "GNTP_ICON_MODE":
		cfg.GNTP.IconMode = value
	case "GNTP_ENABLED":
		cfg.GNTP.Enabled = parseBool(value)
	case "DISPLAY_SHOW_ALBUM_ART":
		cfg.Display.ShowAlbumArt = parseBool(value)
	case "DISPLAY_USE_COLOR":
		cfg.Display.UseColor = parseBool(value)
	}
}

func applyConfigValue(cfg *Config, section, key, value string) {
	section = strings.ToLower(section)
	key = strings.ToLower(key)

	switch section {
	case "mpd":
		switch key {
		case "host":
			cfg.MPD.Host = value
		case "port":
			cfg.MPD.Port = value
		case "password":
			cfg.MPD.Password = value
		case "timeout":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.MPD.Timeout = v
			}
		case "music_root", "musicroot":
			cfg.MPD.MusicRoot = value
		case "config_path", "configpath":
			cfg.MPD.ConfigPath = value
		}
	case "gntp":
		switch key {
		case "host":
			cfg.GNTP.Host = value
		case "port":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.GNTP.Port = v
			}
		case "password":
			cfg.GNTP.Password = value
		case "icon_mode", "iconmode":
			cfg.GNTP.IconMode = value
		case "enabled":
			cfg.GNTP.Enabled = parseBool(value)
		}
	case "display":
		switch key {
		case "show_album_art", "showalbumart":
			cfg.Display.ShowAlbumArt = parseBool(value)
		case "use_color", "usecolor":
			cfg.Display.UseColor = parseBool(value)
		}
	}
}

func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// ── Saving ──────────────────────────────────────────────────────────────────

// SaveConfigToFile writes a Config struct to disk in the specified format.
func SaveConfigToFile(filename string, cfg *Config, format ConfigFormat) error {
	if format == FormatUnknown {
		format = detectFormat(filename)
		if format == FormatUnknown {
			format = FormatTOML
		}
	}

	var (
		data []byte
		err  error
	)

	switch format {
	case FormatTOML:
		var buf strings.Builder
		if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
			return err
		}
		data = []byte(buf.String())
	case FormatJSON:
		data, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
	case FormatYAML:
		data, err = yaml.Marshal(cfg)
		if err != nil {
			return err
		}
	case FormatENV:
		data = []byte(configToEnv(cfg))
	case FormatINI:
		data = []byte(configToINI(cfg))
	default:
		return fmt.Errorf("unsupported save format: %v", format)
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func configToEnv(cfg *Config) string {
	var sb strings.Builder
	sb.WriteString("# MPD Configuration\n")
	sb.WriteString(fmt.Sprintf("MPD_HOST=%s\n", cfg.MPD.Host))
	sb.WriteString(fmt.Sprintf("MPD_PORT=%s\n", cfg.MPD.Port))
	if cfg.MPD.Password != "" {
		sb.WriteString(fmt.Sprintf("MPD_PASSWORD=%q\n", cfg.MPD.Password))
	}
	sb.WriteString(fmt.Sprintf("MPD_TIMEOUT=%d\n", cfg.MPD.Timeout))
	sb.WriteString(fmt.Sprintf("MPD_MUSIC_ROOT=%q\n", cfg.MPD.MusicRoot))
	sb.WriteString(fmt.Sprintf("MPD_CONFIG_PATH=%q\n", cfg.MPD.ConfigPath))
	sb.WriteString("\n# GNTP Configuration\n")
	sb.WriteString(fmt.Sprintf("GNTP_HOST=%s\n", cfg.GNTP.Host))
	sb.WriteString(fmt.Sprintf("GNTP_PORT=%d\n", cfg.GNTP.Port))
	sb.WriteString(fmt.Sprintf("GNTP_ICON_MODE=%s\n", cfg.GNTP.IconMode))
	sb.WriteString(fmt.Sprintf("GNTP_ENABLED=%v\n", cfg.GNTP.Enabled))
	sb.WriteString("\n# Display Configuration\n")
	sb.WriteString(fmt.Sprintf("DISPLAY_SHOW_ALBUM_ART=%v\n", cfg.Display.ShowAlbumArt))
	sb.WriteString(fmt.Sprintf("DISPLAY_USE_COLOR=%v\n", cfg.Display.UseColor))
	return sb.String()
}

func configToINI(cfg *Config) string {
	var sb strings.Builder
	sb.WriteString("[mpd]\n")
	sb.WriteString(fmt.Sprintf("host = %s\n", cfg.MPD.Host))
	sb.WriteString(fmt.Sprintf("port = %s\n", cfg.MPD.Port))
	sb.WriteString(fmt.Sprintf("timeout = %d\n", cfg.MPD.Timeout))
	sb.WriteString(fmt.Sprintf("music_root = %q\n", cfg.MPD.MusicRoot))
	sb.WriteString(fmt.Sprintf("config_path = %q\n", cfg.MPD.ConfigPath))
	sb.WriteString("\n[gntp]\n")
	sb.WriteString(fmt.Sprintf("host = %s\n", cfg.GNTP.Host))
	sb.WriteString(fmt.Sprintf("port = %d\n", cfg.GNTP.Port))
	sb.WriteString(fmt.Sprintf("icon_mode = %s\n", cfg.GNTP.IconMode))
	sb.WriteString(fmt.Sprintf("enabled = %v\n", cfg.GNTP.Enabled))
	sb.WriteString("\n[display]\n")
	sb.WriteString(fmt.Sprintf("show_album_art = %v\n", cfg.Display.ShowAlbumArt))
	sb.WriteString(fmt.Sprintf("use_color = %v\n", cfg.Display.UseColor))
	return sb.String()
}
