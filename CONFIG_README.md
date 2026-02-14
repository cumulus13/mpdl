# MPDL Dynamic Configuration System

This implementation supports multiple configuration file formats with automatic detection and parsing.

## Supported Formats

1. **ENV** (.env files) - Environment variable style
2. **INI** (.ini files) - Windows-style INI files
3. **TOML** (.toml files) - Tom's Obvious Minimal Language
4. **YAML** (.yaml, .yml files) - YAML Ain't Markup Language
5. **JSON** (.json files) - JavaScript Object Notation

## Configuration File Search Order

The application searches for configuration files in the following locations:

### Windows:
1. `%USERPROFILE%/.mpdl`
2. `%APPDATA%/mpdl.ini`
3. `%USERPROFILE%/mpdl.ini`
4. `%APPDATA%/mpdl.toml`
5. `%USERPROFILE%/mpdl.toml`
6. `%APPDATA%/mpdl.json`
7. `%USERPROFILE%/mpdl.json`
8. `%APPDATA%/mpdl.yml`
9. `%USERPROFILE%/mpdl.yml`
10. And more...

### Linux/macOS:
1. `~/.mpdl`
2. `~/.config/.mpdl`
3. `~/.config/mpdl.ini`
4. `~/.config/mpdl.toml`
5. `~/.config/mpdl.json`
6. `~/.config/mpdl.yml`
7. And more...

## Automatic Format Detection

The system automatically detects the configuration file format in three ways:

1. **By file extension** (.env, .ini, .toml, .yaml, .yml, .json)
2. **By content analysis** (for files without extension like `.mpdl`)
3. **By trying all parsers** (fallback mechanism)

## Configuration File Examples

### .env Format (Recommended for simplicity)
```env
# MPD Configuration
MPD_HOST=localhost
MPD_PORT=6600
MPD_PASSWORD=
MPD_TIMEOUT=10
MPD_MUSIC_ROOT="C:/Musics"
MPD_CONFIG_PATH="C:/mpd/mpd.conf"

# GNTP Configuration
GNTP_HOST=localhost
GNTP_PORT=23053
GNTP_ENABLED=true
GNTP_ICON_MODE=binary

# Display Configuration
DISPLAY_SHOW_ALBUM_ART=true
DISPLAY_USE_COLOR=true
```

### INI Format
```ini
[mpd]
host = localhost
port = 6600
timeout = 10
music_root = "C:/Musics"
config_path = "C:/mpd/mpd.conf"

[gntp]
host = localhost
port = 23053
enabled = true
icon_mode = binary

[display]
show_album_art = true
use_color = true
```

### TOML Format
```toml
[mpd]
host = "localhost"
port = "6600"
password = ""
timeout = 10
music_root = "C:/Musics"
config_path = "C:/mpd/mpd.conf"

[gntp]
host = "localhost"
port = 23053
password = ""
icon_mode = "binary"
enabled = true

[display]
show_album_art = true
use_color = true
```

### YAML Format
```yaml
mpd:
  host: localhost
  port: "6600"
  password: ""
  timeout: 10
  music_root: "C:/Musics"
  config_path: "C:/mpd/mpd.conf"

gntp:
  host: localhost
  port: 23053
  password: ""
  icon_mode: binary
  enabled: true

display:
  show_album_art: true
  use_color: true
```

### JSON Format
```json
{
  "mpd": {
    "host": "localhost",
    "port": "6600",
    "password": "",
    "timeout": 10,
    "music_root": "C:/Musics",
    "config_path": "C:/mpd/mpd.conf"
  },
  "gntp": {
    "host": "localhost",
    "port": 23053,
    "password": "",
    "icon_mode": "binary",
    "enabled": true
  },
  "display": {
    "show_album_art": true,
    "use_color": true
  }
}
```

## Configuration Priority

Configuration values are loaded and overridden in this order (later values override earlier ones):

1. **Default values** (hardcoded in the application)
2. **Configuration file** (any supported format)
3. **Environment variables** (MPD_HOST, MPD_PORT, etc.)
4. **Command-line flags** (--mpd-host, --mpd-port, etc.)

## Environment Variables

All configuration options can be set via environment variables:

- `MPD_HOST` - MPD server hostname
- `MPD_PORT` - MPD server port
- `MPD_PASSWORD` - MPD password
- `MPD_TIMEOUT` - Connection timeout
- `MPD_MUSIC_ROOT` - Music directory root
- `MPD_CONFIG_PATH` - Path to MPD config file
- `GNTP_HOST` - GNTP/Growl server hostname
- `GNTP_PORT` - GNTP/Growl server port
- `GNTP_PASSWORD` - GNTP/Growl password
- `GNTP_ICON_MODE` - Icon mode (binary, dataurl, fileurl, httpurl)
- `GNTP_ENABLED` - Enable GNTP notifications (true/false)
- `DISPLAY_SHOW_ALBUM_ART` - Show album art (true/false)
- `DISPLAY_USE_COLOR` - Use colored output (true/false)
- `DEBUG` - Enable debug mode (1/0)

## Usage Examples

### Using a specific config file:
```bash
# Explicitly specify config file
mpdl -c ~/.config/mpdl.toml play

# Or use environment variable
export MPD_HOST=192.168.1.100
mpdl play
```

### Command-line override:
```bash
# Override config file settings
mpdl --mpd-host=192.168.1.100 --mpd-port=6601 play
```

### Creating a new config file:
```bash
# Create ENV format (recommended)
cat > ~/.mpdl << 'EOF'
MPD_HOST=localhost
MPD_PORT=6600
MPD_MUSIC_ROOT="/home/user/Music"
GNTP_ENABLED=true
EOF

# Or create TOML format
cat > ~/.config/mpdl.toml << 'EOF'
[mpd]
host = "localhost"
port = "6600"
music_root = "/home/user/Music"

[gntp]
enabled = true
EOF
```

## Required Go Dependencies

Add these to your `go.mod`:

```bash
go get github.com/BurntSushi/toml
go get gopkg.in/yaml.v3
```

Your existing dependencies should work for JSON (built-in) and custom ENV/INI parsing.

## Troubleshooting

### Config file not found
The application will use default values if no config file is found. Check the search paths listed above.

### Parse errors
If you get a parse error:
1. Check your file format syntax
2. The system will try all formats automatically
3. Enable debug mode to see which parser is being used: `mpdl --debug`

### Values not being applied
Check the priority order - environment variables and CLI flags override config file values.

## Migration from Old Format

If you have an old config file that's not being parsed correctly:

1. The new system will automatically try to parse it
2. If it fails, create a new `.env` format config (simplest)
3. Or convert to any supported format using the examples above
