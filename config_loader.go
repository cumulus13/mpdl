// File: config_loader.go
// Dynamic multi-format configuration loader
// Supports: .env, .toml, .json, .yaml, .yml, .ini

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

// ConfigFormat represents the configuration file format
type ConfigFormat int

const (
	FormatUnknown ConfigFormat = iota
	FormatTOML
	FormatJSON
	FormatYAML
	FormatINI
	FormatENV
)

// detectFormat detects the configuration file format based on extension
func detectFormat(filename string) ConfigFormat {
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
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
	case "": // Files without extension (like .mpdl)
		// Try to detect by content
		return detectFormatByContent(filename)
	default:
		return FormatUnknown
	}
}

// detectFormatByContent tries to detect format by reading file content
func detectFormatByContent(filename string) ConfigFormat {
	data, err := os.ReadFile(filename)
	if err != nil {
		return FormatUnknown
	}
	
	content := strings.TrimSpace(string(data))
	if len(content) == 0 {
		return FormatUnknown
	}
	
	// Check for JSON
	if (strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}")) ||
	   (strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]")) {
		return FormatJSON
	}
	
	// Check for YAML markers
	if strings.Contains(content, "---") || 
	   (strings.Contains(content, ":") && !strings.Contains(content, "=")) {
		return FormatYAML
	}
	
	// Check for ENV format (KEY=VALUE or KEY="VALUE")
	lines := strings.Split(content, "\n")
	envLikeLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "=") && !strings.Contains(line, "[") {
			envLikeLines++
		}
	}
	if envLikeLines > 0 && float64(envLikeLines)/float64(len(lines)) > 0.3 {
		return FormatENV
	}
	
	// Check for INI format (has [sections])
	if strings.Contains(content, "[") && strings.Contains(content, "]") {
		return FormatINI
	}
	
	// Check for TOML
	if strings.Contains(content, "[") || strings.Contains(content, "=") {
		return FormatTOML
	}
	
	return FormatUnknown
}

// LoadConfigFromFile loads configuration from file with automatic format detection
func LoadConfigFromFile(filename string, cfg *Config) error {
	if filename == "" {
		return fmt.Errorf("empty filename")
	}
	
	// Check if file exists
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
		// Try each format until one works
		formats := []struct {
			name string
			fn   func(string, *Config) error
		}{
			{"ENV", loadENV},    // Try ENV first as it's most forgiving
			{"INI", loadINI},
			{"TOML", loadTOML},
			{"YAML", loadYAML},
			{"JSON", loadJSON},
		}
		
		var lastErr error
		for _, f := range formats {
			if err := f.fn(filename, cfg); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		
		return fmt.Errorf("could not parse config file (tried all formats): %v", lastErr)
	}
}

// loadTOML loads TOML configuration
func loadTOML(filename string, cfg *Config) error {
	_, err := toml.DecodeFile(filename, cfg)
	return err
}

// loadJSON loads JSON configuration
func loadJSON(filename string, cfg *Config) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

// loadYAML loads YAML configuration
func loadYAML(filename string, cfg *Config) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

// loadINI loads INI configuration
func loadINI(filename string, cfg *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	currentSection := ""
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		
		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}
		
		// Key-value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		
		applyConfigValue(cfg, currentSection, key, value)
	}
	
	return scanner.Err()
}

// loadENV loads .env style configuration
func loadENV(filename string, cfg *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Key-value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		
		// Parse ENV-style keys (e.g., MPD_HOST, MPD_PORT)
		applyEnvStyleKey(cfg, key, value)
	}
	
	return scanner.Err()
}

// applyEnvStyleKey applies environment-style keys (MPD_HOST, GNTP_PORT, etc.)
func applyEnvStyleKey(cfg *Config, key, value string) {
	key = strings.ToUpper(key)
	
	switch {
	// MPD settings
	case key == "MPD_HOST":
		cfg.MPD.Host = value
	case key == "MPD_PORT":
		cfg.MPD.Port = value
	case key == "MPD_PASSWORD":
		cfg.MPD.Password = value
	case key == "MPD_TIMEOUT":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.MPD.Timeout = v
		}
	case key == "MPD_MUSIC_ROOT":
		cfg.MPD.MusicRoot = value
	case key == "MPD_CONFIG_PATH":
		cfg.MPD.ConfigPath = value
		
	// GNTP settings
	case key == "GNTP_HOST":
		cfg.GNTP.Host = value
	case key == "GNTP_PORT":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.GNTP.Port = v
		}
	case key == "GNTP_PASSWORD":
		cfg.GNTP.Password = value
	case key == "GNTP_ICON_MODE":
		cfg.GNTP.IconMode = value
	case key == "GNTP_ENABLED":
		cfg.GNTP.Enabled = value == "true" || value == "1" || value == "yes"
		
	// Display settings
	case key == "DISPLAY_SHOW_ALBUM_ART":
		cfg.Display.ShowAlbumArt = value == "true" || value == "1" || value == "yes"
	case key == "DISPLAY_USE_COLOR":
		cfg.Display.UseColor = value == "true" || value == "1" || value == "yes"
	}
}

// applyConfigValue applies a configuration value based on section and key
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
			cfg.GNTP.Enabled = value == "true" || value == "1" || value == "yes"
		}
		
	case "display":
		switch key {
		case "show_album_art", "showalbumart":
			cfg.Display.ShowAlbumArt = value == "true" || value == "1" || value == "yes"
		case "use_color", "usecolor":
			cfg.Display.UseColor = value == "true" || value == "1" || value == "yes"
		}
	}
}

// SaveConfigToFile saves configuration to file in specified format
func SaveConfigToFile(filename string, cfg *Config, format ConfigFormat) error {
	// Auto-detect format from extension if format is unknown
	if format == FormatUnknown {
		format = detectFormat(filename)
		if format == FormatUnknown {
			format = FormatTOML // Default to TOML
		}
	}
	
	var data []byte
	var err error
	
	switch format {
	case FormatTOML:
		var buf strings.Builder
		encoder := toml.NewEncoder(&buf)
		if err := encoder.Encode(cfg); err != nil {
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
		return fmt.Errorf("unsupported format for saving: %v", format)
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

// configToEnv converts Config to .env format
func configToEnv(cfg *Config) string {
	var sb strings.Builder
	
	sb.WriteString("# MPD Configuration\n")
	sb.WriteString(fmt.Sprintf("MPD_HOST=%s\n", cfg.MPD.Host))
	sb.WriteString(fmt.Sprintf("MPD_PORT=%s\n", cfg.MPD.Port))
	if cfg.MPD.Password != "" {
		sb.WriteString(fmt.Sprintf("MPD_PASSWORD=\"%s\"\n", cfg.MPD.Password))
	}
	sb.WriteString(fmt.Sprintf("MPD_TIMEOUT=%d\n", cfg.MPD.Timeout))
	sb.WriteString(fmt.Sprintf("MPD_MUSIC_ROOT=\"%s\"\n", cfg.MPD.MusicRoot))
	sb.WriteString(fmt.Sprintf("MPD_CONFIG_PATH=\"%s\"\n", cfg.MPD.ConfigPath))
	
	sb.WriteString("\n# GNTP Configuration\n")
	sb.WriteString(fmt.Sprintf("GNTP_HOST=%s\n", cfg.GNTP.Host))
	sb.WriteString(fmt.Sprintf("GNTP_PORT=%d\n", cfg.GNTP.Port))
	if cfg.GNTP.Password != "" {
		sb.WriteString(fmt.Sprintf("GNTP_PASSWORD=\"%s\"\n", cfg.GNTP.Password))
	}
	sb.WriteString(fmt.Sprintf("GNTP_ICON_MODE=%s\n", cfg.GNTP.IconMode))
	sb.WriteString(fmt.Sprintf("GNTP_ENABLED=%v\n", cfg.GNTP.Enabled))
	
	sb.WriteString("\n# Display Configuration\n")
	sb.WriteString(fmt.Sprintf("DISPLAY_SHOW_ALBUM_ART=%v\n", cfg.Display.ShowAlbumArt))
	sb.WriteString(fmt.Sprintf("DISPLAY_USE_COLOR=%v\n", cfg.Display.UseColor))
	
	return sb.String()
}

// configToINI converts Config to INI format
func configToINI(cfg *Config) string {
	var sb strings.Builder
	
	sb.WriteString("[mpd]\n")
	sb.WriteString(fmt.Sprintf("host = %s\n", cfg.MPD.Host))
	sb.WriteString(fmt.Sprintf("port = %s\n", cfg.MPD.Port))
	if cfg.MPD.Password != "" {
		sb.WriteString(fmt.Sprintf("password = \"%s\"\n", cfg.MPD.Password))
	}
	sb.WriteString(fmt.Sprintf("timeout = %d\n", cfg.MPD.Timeout))
	sb.WriteString(fmt.Sprintf("music_root = \"%s\"\n", cfg.MPD.MusicRoot))
	sb.WriteString(fmt.Sprintf("config_path = \"%s\"\n", cfg.MPD.ConfigPath))
	
	sb.WriteString("\n[gntp]\n")
	sb.WriteString(fmt.Sprintf("host = %s\n", cfg.GNTP.Host))
	sb.WriteString(fmt.Sprintf("port = %d\n", cfg.GNTP.Port))
	if cfg.GNTP.Password != "" {
		sb.WriteString(fmt.Sprintf("password = \"%s\"\n", cfg.GNTP.Password))
	}
	sb.WriteString(fmt.Sprintf("icon_mode = %s\n", cfg.GNTP.IconMode))
	sb.WriteString(fmt.Sprintf("enabled = %v\n", cfg.GNTP.Enabled))
	
	sb.WriteString("\n[display]\n")
	sb.WriteString(fmt.Sprintf("show_album_art = %v\n", cfg.Display.ShowAlbumArt))
	sb.WriteString(fmt.Sprintf("use_color = %v\n", cfg.Display.UseColor))
	
	return sb.String()
}
