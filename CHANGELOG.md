# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
