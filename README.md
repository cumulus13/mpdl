# mpdl - Advanced MPD CLI Client

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](https://github.com/cumulus13/mpdl)

Command-line client for [Music Player Daemon (MPD)](https://www.musicpd.org/) with monitoring capabilities, GNTP/Growl notifications, and MPC-like functionality.

## Features

✨ **Core Features:**
- 🎵 Complete MPD control (play, pause, stop, next, previous)
- 📝 Playlist management (add, delete, clear, list)
- 🔊 Volume and playback mode control
- 📊 Beautiful colored terminal output
- 🔄 Automatic reconnection on connection loss
- 🖥️ Cross-platform support (Windows, Linux, macOS)

🎬 **Monitor Mode:**
- 👀 Real-time monitoring of MPD status
- 📢 Desktop notifications via GNTP/Growl
- 🎨 Album artwork support
- ⌨️ Keyboard shortcuts for quick control
- 🔔 Song change and state notifications

⚙️ **Advanced:**
- 📄 TOML configuration file support
- 🔧 MPD config file editing (local MPD)
- 🌍 Environment variable support
- 🔍 Debug mode for troubleshooting
- 🎯 MPC-compatible commands

## Installation

### From Source

```bash
git clone https://github.com/cumulus13/mpdl.git
cd mpdl
go build -o mpdl
```

### Using Go Install

```bash
go install github.com/cumulus13/mpdl@latest
```

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/cumulus13/mpdl/releases).

## Quick Start

```bash
# Show help
mpdl --help

# Start playback
mpdl play

# Add a song to playlist
mpdl add ~/Music/song.mp3

# Show current playlist
mpdl list

# Monitor MPD with notifications
mpdl monitor

# Show current status
mpdl status
```

## Configuration

### Configuration File

Create a configuration file at `~/.config/mpdl/config.toml` (Linux/macOS) or `%APPDATA%\mpdl\config.toml` (Windows):

```toml
[mpd]
host = "localhost"
port = "6600"
password = ""
timeout = 10
music_root = "/home/user/Music"  # or "C:/Musics" on Windows
config_path = "/home/user/.config/mpd/mpd.conf"

[gntp]
host = "localhost"
port = 23053
password = ""
icon_mode = "binary"  # binary, dataurl, fileurl, httpurl
enabled = true

[display]
show_album_art = true
use_color = true
```

### Environment Variables

```bash
export MPD_HOST="localhost"
export MPD_PORT="6600"
export MPD_PASSWORD=""
export MPD_TIMEOUT="10"
export MPD_MUSIC_ROOT="/home/user/Music"
export DEBUG="1"  # Enable debug mode
```

## Usage

### Playback Commands

```bash
mpdl play [POS]        # Start playback (optionally at position)
mpdl pause             # Toggle pause
mpdl stop              # Stop playback
mpdl next              # Next song (alias: n)
mpdl prev              # Previous song (aliases: previous, p)
mpdl seek [+/-]TIME    # Seek to position (e.g., 30, +10, -5)
```

### Playlist Commands

```bash
mpdl add PATH          # Add song/directory to playlist
mpdl del PATH          # Delete songs matching PATH (aliases: delete)
mpdl clear             # Clear playlist
mpdl list              # Show current playlist (aliases: ls, playlist)
```

### Information Commands

```bash
mpdl current           # Show current song
mpdl status            # Show player status (alias: st)
```

### Options Commands

```bash
mpdl volume [VOL]      # Set volume (0-100) or show current
mpdl repeat [on|off]   # Toggle or set repeat mode
mpdl random [on|off]   # Toggle or set random mode
mpdl single [on|off]   # Toggle or set single mode
```

### Database Commands

```bash
mpdl update [PATH]     # Update music database
```

### Config Commands (Local MPD Only)

```bash
mpdl get-config KEY          # Get MPD config value
mpdl set-config KEY VALUE    # Set MPD config value
```

Examples:
```bash
mpdl get-config music_directory
mpdl set-config volume_normalization yes
```

### Monitor Mode

```bash
mpdl monitor           # Start monitoring mode (aliases: mon, m)
```

**Keyboard Shortcuts in Monitor Mode:**
- `p` - Play/Pause
- `s` - Stop
- `n` - Next song
- `b` - Previous song (back)
- `q` - Quit monitor

## Examples

### Basic Usage

```bash
# Connect to MPD and show status
mpdl status

# Add all MP3 files from a directory
mpdl add ~/Music/Artist/Album

# Delete all songs from an artist
mpdl del "Artist Name"

# Set volume to 75%
mpdl volume 75

# Enable shuffle mode
mpdl random on
```

### Monitor Mode with Notifications

```bash
# Start monitoring with default settings
mpdl monitor

# Start monitoring with custom config
mpdl --config ~/.config/mpdl/config.toml monitor

# Monitor with debug output
mpdl --debug monitor
```

### Using with Remote MPD

```bash
# Connect to remote MPD server
mpdl --mpd-host 192.168.1.100 --mpd-port 6600 status

# Or set environment variables
export MPD_HOST="192.168.1.100"
export MPD_PORT="6600"
mpdl play
```

### Configuration Management

```bash
# View MPD music directory
mpdl get-config music_directory

# Enable audio normalization
mpdl set-config volume_normalization yes

# Set audio output
mpdl set-config audio_output_format "44100:16:2"
```

## GNTP/Growl Notifications

mpdl supports desktop notifications via GNTP (Growl Notification Transport Protocol). To use this feature:

1. Install a GNTP-compatible notification system:
   - **Windows**: [Growl for Windows](http://www.growlforwindows.com/)
   - **macOS**: [Growl](https://growl.github.io/growl/)
   - **Linux**: [go-gntp](https://github.com/cumulus13/go-gntp) or similar

2. Enable GNTP in your config file:

```toml
[gntp]
host = "localhost"
port = 23053
enabled = true
icon_mode = "binary"  # Recommended for Windows
```

3. Start monitor mode:

```bash
mpdl monitor
```

Notifications will show:
- Album artwork (if available)
- Song title, artist, and album
- Playback time and bitrate
- Player state changes

## Platform-Specific Notes

### Windows

- Default music root: `C:\Musics`
- Default MPD config: `%APPDATA%\mpd\mpd.conf`
- Use binary icon mode for GNTP notifications

### Linux

- Default music root: `~/Music`
- Default MPD config: `~/.config/mpd/mpd.conf`
- May require `libmpd` package

### macOS

- Default music root: `~/Music`
- Default MPD config: `~/.config/mpd/mpd.conf`
- Growl notifications supported

## Troubleshooting

### Connection Issues

```bash
# Enable debug mode
mpdl --debug status

# Check MPD is running
systemctl status mpd  # Linux
# or
net start mpd  # Windows

# Test connection manually
telnet localhost 6600
```

### Permission Issues (Config Editing)

The `get-config` and `set-config` commands only work with local MPD installations where you have read/write access to the MPD configuration file.

### GNTP Notifications Not Working

1. Verify GNTP server is running
2. Check firewall settings
3. Try different icon modes:
   ```bash
   mpdl --config config.toml monitor
   # Edit config.toml and change icon_mode
   ```

## Development

### Building from Source

```bash
git clone https://github.com/cumulus13/mpdl.git
cd mpdl
go mod download
go build -o mpdl
```

### Running Tests

```bash
go test ./...
```

### Building for Multiple Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o mpdl-linux-amd64

# Windows
GOOS=windows GOARCH=amd64 go build -o mpdl-windows-amd64.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o mpdl-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o mpdl-darwin-arm64
```

## Dependencies

- [gompd](https://github.com/fhs/gompd) - MPD client library
- [go-gntp](https://github.com/cumulus13/go-gntp) - GNTP notification library
- [toml](https://github.com/BurntSushi/toml) - TOML parser
- [term](https://golang.org/x/term) - Terminal utilities

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author
        
[Hadi Cahyadi](mailto:cumulus13@gmail.com)
    

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)

## Acknowledgments

- MPD developers for the excellent music player daemon
- gompd library maintainers
- All contributors to this project

## Related Projects

- [MPD](https://www.musicpd.org/) - Music Player Daemon
- [MPC](https://www.musicpd.org/clients/mpc/) - Official MPD command-line client
- [ncmpcpp](https://github.com/ncmpcpp/ncmpcpp) - NCurses MPD client

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for release history.

## Support

- 🐛 [Report bugs](https://github.com/cumulus13/mpdl/issues)
- 💡 [Request features](https://github.com/cumulus13/mpdl/issues)
- 📖 [Documentation](https://github.com/cumulus13/mpdl/wiki)
- 💬 [Discussions](https://github.com/cumulus13/mpdl/discussions)
