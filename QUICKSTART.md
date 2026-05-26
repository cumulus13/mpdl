# mpdl Quick Start Guide

Get up and running with mpdl in 5 minutes!

## Installation

### Linux / macOS

```bash
# Quick install
curl -sSL https://raw.githubusercontent.com/cumulus13/mpdl/main/install.sh | bash

# Or download and run
wget https://raw.githubusercontent.com/cumulus13/mpdl/main/install.sh
chmod +x install.sh
./install.sh
```

### Windows

```powershell
# Run in PowerShell as Administrator
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/cumulus13/mpdl/main/install.ps1" -OutFile "install.ps1"
powershell -ExecutionPolicy Bypass -File install.ps1
```

### From Source

```bash
git clone https://github.com/cumulus13/mpdl.git
cd mpdl
make build
sudo make install
```

## Basic Usage

### First Time Setup

#### Local MPD

1. **Make sure MPD is running:**
   ```bash
   # Linux
   systemctl status mpd
   
   # macOS
   brew services list
   
   # Windows
   # Check if mpd.exe is running in Task Manager
   ```

2. **Test connection:**
   ```bash
   mpdl status
   ```

3. **Configure (optional):**
   ```bash
   # Edit config file
   # Linux/macOS: ~/.config/mpdl/config.toml
   # Windows: %APPDATA%\mpdl\config.toml
   ```

### Essential Commands

```bash
# Playback control
mpdl play              # Start playback
mpdl pause             # Toggle pause
mpdl stop              # Stop playback
mpdl next              # Next song
mpdl prev              # Previous song

# Playlist management
mpdl add ~/Music/song.mp3       # Add a song
mpdl add ~/Music/Artist         # Add a directory
mpdl list                       # Show playlist
mpdl clear                      # Clear playlist
mpdl del "Artist/Album"         # Delete songs

# Volume & modes
mpdl volume 75         # Set volume to 75%
mpdl volume            # Show current volume
mpdl random on         # Enable shuffle
mpdl repeat on         # Enable repeat
mpdl single on         # Enable single mode

# Information
mpdl status            # Show player status
mpdl current           # Show current song

# Database
mpdl update            # Update music database
```

### Monitor Mode

Start real-time monitoring with notifications:

```bash
mpdl monitor
```

**Keyboard shortcuts in monitor mode:**
- `p` - Play/Pause
- `s` - Stop
- `n` - Next song
- `b` - Previous song (back)
- `q` - Quit

### Common Use Cases

#### 1. Add and Play Music

```bash
# Add a folder and start playing
mpdl clear
mpdl add ~/Music/My-Playlist
mpdl play
```

#### 2. Create a Shuffle Queue

```bash
# Clear, add music, shuffle, and play
mpdl clear
mpdl add ~/Music
mpdl random on
mpdl play
```

#### 3. Volume Control

```bash
# Set volume
mpdl volume 50

# Check current volume
mpdl volume
```

#### 4. Monitor with Notifications

```bash
# Start monitoring (shows notifications for song changes)
mpdl monitor
```

#### 5. Remote MPD Control

```bash
# Connect to remote MPD server
mpdl --mpd-host 192.168.1.100 --mpd-port 6600 status

# Or set environment variables
export MPD_HOST="192.168.1.100"
export MPD_PORT="6600"
mpdl play
```

## Configuration

### Quick Config

Edit `~/.config/mpdl/config.toml` (Linux/macOS) or `%APPDATA%\mpdl\config.toml` (Windows):

```toml
[mpd]
host = "localhost"
port = "6600"
music_root = "/home/user/Music"  # Your music directory

[gntp]
enabled = true  # Enable/disable notifications
```

### Environment Variables

```bash
# Set in ~/.bashrc or ~/.zshrc
export MPD_HOST="localhost"
export MPD_PORT="6600"
export MPD_MUSIC_ROOT="$HOME/Music"
```

## Troubleshooting

### Connection Issues

```bash
# Check if MPD is running
telnet localhost 6600

# Enable debug mode
mpdl --debug status

# Check MPD logs
# Linux: journalctl -u mpd
# macOS: brew services list
```

### "Permission Denied" on Linux

```bash
# Add your user to the audio group
sudo usermod -a -G audio $USER

# Or fix MPD socket permissions
sudo chmod 666 /run/mpd/socket
```

### Notifications Not Working

1. Install Growl/GNTP server:
   - **Windows**: [Growl for Windows](http://www.growlforwindows.com/)
   - **macOS**: [Growl](https://growl.github.io/growl/)
   - **Linux**: Install `gntp-send` or similar

2. Enable in config:
   ```toml
   [gntp]
   enabled = true
   ```

3. Test:
   ```bash
   mpdl monitor
   # Change song and check for notification
   ```

### Music Not Found

Make sure `music_root` in config matches your MPD's `music_directory`:

```bash
# Check MPD config
mpdl get-config music_directory

# Update mpdl config to match
```

## Next Steps

- Read the full [README](README.md)
- Check the [Configuration Guide](config.toml.example)
- Browse [Examples](#examples)
- Report issues on [GitHub](https://github.com/cumulus13/mpdl/issues)

## Examples

### Example 1: Quick DJ Mode

```bash
# Clear playlist, add library, shuffle, and play
mpdl clear && mpdl add ~/Music && mpdl random on && mpdl play
```

### Example 2: Album Listening

```bash
# Play a specific album in order
mpdl clear
mpdl add "Artist Name/Album Name"
mpdl random off
mpdl repeat on
mpdl play
```

### Example 3: Background Monitor

```bash
# Start monitoring in background (Linux/macOS)
nohup mpdl monitor > /dev/null 2>&1 &

# Or use screen/tmux
screen -dmS mpdl-monitor mpdl monitor
```

### Example 4: Party Mode Script

```bash
#!/bin/bash
# party.sh - Set up party mode

mpdl clear
mpdl add ~/Music/Party
mpdl random on
mpdl repeat on
mpdl volume 80
mpdl play

echo "🎉 Party mode activated!"
mpdl status
```

## Tips & Tricks

1. **Alias for Quick Access:**
   ```bash
   alias mp='mpdl'
   alias mpp='mpdl play'
   alias mps='mpdl stop'
   ```

2. **Quick Playlist from Find:**
   ```bash
   find ~/Music -name "*.mp3" -type f | while read f; do mpdl add "$f"; done
   ```

3. **Random Artist:**
   ```bash
   mpdl clear
   mpdl add "$(ls ~/Music | shuf -n 1)"
   mpdl play
   ```

4. **Volume Hotkeys (with xbindkeys on Linux):**
   ```
   "mpdl volume +5"
       XF86AudioRaiseVolume
   "mpdl volume -5"
       XF86AudioLowerVolume
   ```

## Get Help

```bash
# Show all commands
mpdl --help

# Show version
mpdl --version

# Enable debug output
mpdl --debug [command]
```

---

**Enjoy your music with mpdl! 🎵**
