# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-02-16

### Added - MPC Compatibility & Media Keys
- **MPC-compatible commands**: toggle, crop, move, save, load, lsplaylists, rm, listall
- **Search commands**: search, find with multiple types (artist, album, title, etc.)
- **Output management**: outputs, enable, disable for audio device control
- **Playback options**: consume mode, crossfade support
- **Statistics command**: stats to show MPD statistics
- **Media keys support**: Framework for Bluetooth headset and keyboard media keys
- **Media key setup guide**: `mediakeys` command with platform-specific instructions
- **Multi-format config support**: TOML, JSON, YAML, INI, and ENV files
- **Dynamic config loading**: Automatic format detection and flexible config paths
- **Improved path handling**: Better support for paths with spaces
- **Comprehensive documentation**: MPC_COMMANDS.md with all command references

### Changed
- Improved `add` and `del` commands to handle paths with spaces
- Enhanced `seek` command with better decimal support
- Better error messages with actual error details
- Improved help text with categorized commands
- More examples in help output

### Fixed
- Path normalization for remote MPD servers
- Handling of quoted arguments
- Case-sensitive path matching on Linux
- Config file loading from multiple locations

### Technical
- Added `config_loader.go` for multi-format config support
- Added `media_keys.go` for media key handling framework
- Improved command organization in main.go
- Better code documentation

## [1.0.0] - 2026-02-04

### Added
- Initial release of mpdl
- Complete MPD playback control (play, pause, stop, next, previous)
- Playlist management (add, delete, clear, list)
- Volume and playback mode control (repeat, random, single)
- Monitor mode with real-time status updates
- GNTP/Growl notification support with album artwork
- Keyboard shortcuts in monitor mode
- MPD config file editing (get-config, set-config)
- Cross-platform support (Windows, Linux, macOS, FreeBSD)
- TOML configuration file support
- Environment variable configuration
- Automatic reconnection on connection loss
- Debug mode for troubleshooting
- Colored terminal output
- Comprehensive help system
- MPC-compatible command syntax

### Features
- 🎵 Full MPD control via command line
- 📝 Easy playlist management
- 🔊 Volume and mode controls
- 📊 Beautiful colored output
- 🔄 Robust error handling and reconnection
- 🖥️ True cross-platform support
- 🎬 Monitor mode with notifications
- 🎨 Album artwork in notifications
- ⚙️ Flexible configuration options
- 🔧 Local MPD config editing

### Technical Details
- Written in Go 1.21+
- Uses gompd for MPD communication
- GNTP protocol for desktop notifications
- TOML for configuration
- Automatic platform detection
- Exponential backoff for reconnection
- Production-ready error handling

### Platforms Supported
- Linux (amd64, arm64, armv7, 386)
- Windows (amd64, arm64, 386)
- macOS (amd64, arm64/Apple Silicon)
- FreeBSD (amd64)

### Dependencies
- github.com/fhs/gompd/v2 v2.3.0
- github.com/BurntSushi/toml v1.3.2
- github.com/cumulus13/go-gntp (for notifications)
- golang.org/x/term v0.16.0

## [Unreleased]

### Planned Features
- Interactive playlist editor
- Lyrics display support
- Playlist search and filter
- Output device switching
- Sticker database support
- Multiple playlist management
- Stream URL support
- Music database search
- GUI mode (optional)
- Plugin system

### Known Issues
- Config editing only works for local MPD instances
- GNTP icon modes may vary by platform
- Monitor mode keyboard shortcuts require terminal focus

---

[1.0.0]: https://github.com/cumulus13/mpdl/releases/tag/v1.0.0
