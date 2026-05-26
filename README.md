# mpdl — Advanced MPD CLI Client

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](https://github.com/cumulus13/mpdl)

Command-line client for [Music Player Daemon (MPD)](https://www.musicpd.org/) with MPC-compatible commands, smart pattern matching, GNTP/Growl notifications, album-aware playback, and media key support.

---

## Features

**Playback**
- Play, pause/toggle, stop, next, previous, seek (absolute or relative, decimal seconds)
- `play *pattern*` — search the queue by wildcard, regex, or substring; prompts when multiple tracks match
- `addplay PATH` — replace queue with a folder/album, auto-sort by Disc→Track tag, play from track #1

**Queue management**
- Add, insert (after current), delete, clear, crop, shuffle, move
- `del` accepts: position number, position range (`3-7`), `/regex/`, `glob*`, or substring
- `insert PATH` — queue a track to play next without interrupting the current song

**Search**
- `search`, `find`, `findadd` against artist/album/title/any/…
- `listall [PATH]` — browse the music library

**Rich status display** (MPC-style)
- Progress bar with elapsed/total, bitrate, volume, repeat/random/single/consume/crossfade
- Full song metadata: title, artist, album, date, track, genre, file path

**Saved playlists**
- Save, load, list, delete named playlists

**Notifications**
- GNTP/Growl desktop notifications with album artwork
- Song-change and state-change events

**Media keys / Bluetooth headset**
- `mpdl monitor --media-keys` enables platform media key integration
- `mpdl mediakeys` prints a step-by-step setup guide for your OS

**Outputs & database**
- List, enable, disable audio outputs
- `update` (incremental) and `rescan` (full tag re-read)

**Configuration**
- Multi-format config: TOML, JSON, YAML, INI, `.env` — auto-detected
- Auto-discovery of config file from standard locations
- Environment variable overrides

---

## Installation

### From source

```bash
git clone https://github.com/cumulus13/mpdl.git
cd mpdl
go build -o mpdl .
```

### Using go install

```bash
go install github.com/cumulus13/mpdl@latest
```

### Pre-built binaries

Download from the [releases page](https://github.com/cumulus13/mpdl/releases).
Available for: Windows amd64/arm64, Linux amd64/arm64/arm/386, macOS amd64/arm64, FreeBSD amd64.

---

## Quick Start

```bash
mpdl status          # Show current status (MPC-style with progress bar)
mpdl play            # Resume / start playing
mpdl next            # Skip to next track
mpdl volume +10      # Raise volume by 10%
mpdl list            # Show current queue
```

---

## Commands Reference

### Playback

```bash
mpdl play                  # Resume if paused, otherwise start
mpdl play 5                # Jump to queue position 5 (1-based)
mpdl play "*girl*"         # Find tracks matching pattern, prompt if multiple
mpdl play "/pink floyd/"   # Regex match (case-insensitive)
mpdl play "soul"           # Substring match against title/artist/album/file
mpdl pause                 # Toggle pause
mpdl toggle                # Same as pause
mpdl stop                  # Stop
mpdl next, n               # Next track
mpdl prev, previous, p     # Previous track
mpdl seek 90               # Seek to 1:30
mpdl seek +15              # Seek forward 15 seconds
mpdl seek -5.5             # Seek back 5.5 seconds
```

**Pattern matching for `play`**

| Pattern | Example | Behaviour |
|---|---|---|
| Bare text | `mpdl play soul` | Case-insensitive substring match |
| Glob/wildcard | `mpdl play "*girl*"` | `*` matches any chars, `?` matches one |
| Regex | `mpdl play "/floyd\|bowie/"` | Full regex, always case-insensitive |
| Number | `mpdl play 3` | Jump to queue position 3 |

When **multiple tracks** match, mpdl shows a numbered list and waits for your input:
```
3 tracks match "*girl*" — pick one:

  1. Girl from Ipanema · Stan Getz  [queue #4]
  2. California Girls · Beach Boys  [queue #12]
  3. Girls Just Wanna Have Fun · Cyndi Lauper  [queue #17]

Enter number (1-3), or 'a' to play all from first match, or Enter to cancel:
```

---

### Queue

```bash
mpdl add PATH              # Add file or folder to end of queue
mpdl addplay PATH          # Clear queue, add PATH, sort by track #, play
mpdl insert PATH           # Add after currently playing track (play next)
mpdl list / ls / playlist  # Show queue
mpdl clear                 # Clear queue
mpdl crop                  # Remove all except current track
mpdl shuffle               # Shuffle queue
mpdl move 5 2              # Move track 5 to position 2 (1-based)
```

**`addplay` — Album/folder playback**

```bash
mpdl addplay "Pink Floyd/The Wall"
mpdl addplay "C:/Musics/Jazz/Miles Davis/Kind of Blue"
```

This clears the current queue, adds all tracks from the folder, **sorts them by Disc→Track metadata tag** (not filename), and starts playing from track #1. Works correctly even if files are named `02 - Title.flac` or have inconsistent filenames.

---

### Deleting from the queue

```bash
mpdl del 4                 # Delete position 4 (1-based)
mpdl del 0                 # Delete currently playing track (mpc-style)
mpdl del 3-7               # Delete positions 3 through 7 (inclusive)
mpdl del "*floyd*"         # Delete all matching glob (title/artist/file)
mpdl del "/bowie/i"        # Delete all matching regex
mpdl del "soul"            # Delete all with "soul" in title/artist/file
```

---

### Saved Playlists

```bash
mpdl save "Evening Chill"  # Save current queue as named playlist
mpdl load "Evening Chill"  # Load playlist into queue
mpdl lsplaylists           # List all saved playlists
mpdl rm "Evening Chill"    # Delete saved playlist
```

---

### Search & Browse

```bash
mpdl search artist "Pink Floyd"       # Partial match
mpdl search any "wall"                # Search all fields
mpdl find album "The Wall"            # Exact match
mpdl findadd artist "Bowie"           # Find and add results to queue
mpdl listall "Rock/Led Zeppelin"      # Browse library path
```

Search types: `artist`, `album`, `title`, `track`, `name`, `genre`, `date`, `composer`, `performer`, `filename`, `any`

---

### Status & Information

```bash
mpdl status / st           # Full MPC-style status
mpdl current               # Currently playing song only
mpdl stats                 # MPD database statistics
mpdl version               # mpdl and MPD server version
```

**Status output example:**
```
────────────────────────────────────────────────────────────────
Comfortably Numb
Pink Floyd
The Wall  (1979)
  Track: 13   Genre: Progressive Rock
  File: Pink Floyd/The Wall/13 - Comfortably Numb.flac

▶ [██████████████░░░░░░░░░░░░░░░░░] 4:12 / 6:23

  Volume: 80%   Repeat: off   Random: off   Single: off   Consume: off
  Queue: 13/26   Bitrate: 44 kHz   Crossfade: 0s
────────────────────────────────────────────────────────────────
```

---

### Volume & Playback Modes

```bash
mpdl volume              # Show current volume
mpdl volume 75           # Set to 75%
mpdl volume +10          # Increase by 10%
mpdl volume -5           # Decrease by 5%
mpdl repeat [on|off]     # Toggle or set repeat
mpdl random [on|off]     # Toggle or set random/shuffle
mpdl single [on|off]     # Toggle or set single (stop after one track)
mpdl consume [on|off]    # Toggle or set consume (remove track after play)
mpdl crossfade [N]       # Set crossfade seconds (0 = off)
```

---

### Audio Outputs

```bash
mpdl outputs             # List all audio outputs with status
mpdl enable 0            # Enable output #0
mpdl disable 1           # Disable output #1
```

---

### Database

```bash
mpdl update [PATH]       # Incremental database update
mpdl rescan [PATH]       # Full rescan (re-reads all tags, slower)
```

---

### MPD Config Editing (local MPD only)

```bash
mpdl get-config music_directory
mpdl set-config volume_normalization yes
```

Reads/writes the MPD config file directly. Restart MPD after changes.

---

### Monitor Mode

```bash
mpdl monitor             # Monitor MPD events, send GNTP notifications
mpdl monitor --media-keys  # Same + enable media key integration
mpdl mon                 # Short alias
```

**Keyboard shortcuts in monitor mode:**

| Key | Action |
|---|---|
| `p` | Play / Pause |
| `n` | Next track |
| `b` | Previous track |
| `s` | Stop |
| `q` | Quit monitor |

---

### Media Keys / Bluetooth Headset

```bash
mpdl mediakeys           # Print OS-specific setup guide
```

**Linux** — install `mpDris2` or `mpd-mpris`:
```bash
sudo apt install mpdris2
systemctl --user enable --now mpd-mpris
```
After this, hardware media keys and Bluetooth headset buttons control MPD natively via MPRIS.

**Windows** — use AutoHotkey:
```ahk
Media_Play_Pause::Run, mpdl pause
Media_Next::Run, mpdl next
Media_Prev::Run, mpdl prev
Media_Stop::Run, mpdl stop
```

**macOS** — use Hammerspoon (free):
```lua
hs.hotkey.bind({}, "F7", function() os.execute("mpdl prev")  end)
hs.hotkey.bind({}, "F8", function() os.execute("mpdl pause") end)
hs.hotkey.bind({}, "F9", function() os.execute("mpdl next")  end)
```

---

## Configuration

### Auto-discovery locations

mpdl searches for a config file automatically (in order):

**Linux/macOS:**
- `~/.mpdl`
- `~/.config/mpdl/config.toml`
- `~/.config/mpdl.toml`
- Also `.ini`, `.json`, `.yaml`, `.env` variants

**Windows:**
- `%USERPROFILE%\.mpdl`
- `%APPDATA%\mpdl\config.toml`
- Also `.ini`, `.json`, `.yaml`, `.env` variants

Override with: `mpdl --config /path/to/config.toml <command>`

---

### TOML (recommended)

```toml
[mpd]
host        = "localhost"
port        = "6600"
password    = ""
timeout     = 10
music_root  = "/home/user/Music"   # Windows: "C:/Musics"
config_path = "~/.config/mpd/mpd.conf"

[gntp]
host      = "localhost"
port      = 23053
password  = ""
icon_mode = "binary"   # binary | dataurl | fileurl | httpurl
enabled   = true

[display]
show_album_art = true
use_color      = true
```

### ENV / .env file

```bash
MPD_HOST=localhost
MPD_PORT=6600
MPD_PASSWORD=
MPD_TIMEOUT=10
MPD_MUSIC_ROOT=/home/user/Music
GNTP_HOST=localhost
GNTP_PORT=23053
GNTP_ICON_MODE=binary
GNTP_ENABLED=true
DISPLAY_SHOW_ALBUM_ART=true
DISPLAY_USE_COLOR=true
```

Also supported: `.json`, `.yaml`, `.ini` — format is auto-detected from extension.

### Environment variables (always override config file)

| Variable | Description |
|---|---|
| `MPD_HOST` | MPD server host |
| `MPD_PORT` | MPD server port |
| `MPD_PASSWORD` | MPD password |
| `MPD_TIMEOUT` | Connection timeout (seconds) |
| `MPD_MUSIC_ROOT` | Music root directory |
| `DEBUG` | Set to `1` for debug output |

---

## GNTP / Growl Notifications

Notifications fire on song change and player state change, with album artwork when available.

| Platform | Recommended app |
|---|---|
| Windows | [Growl for Windows](http://www.growlforwindows.com/) |
| macOS | [Growl](https://growl.github.io/growl/) |
| Linux | [go-gntp](https://github.com/cumulus13/go-gntp) |

```toml
[gntp]
enabled   = true
host      = "localhost"
port      = 23053
icon_mode = "binary"   # use "dataurl" if binary doesn't work on your system
```

---

## Connecting to a Remote MPD

```bash
mpdl --mpd-host 192.168.1.100 --mpd-port 6600 status
```

Or set environment variables:
```bash
export MPD_HOST=192.168.1.100
export MPD_PORT=6600
mpdl play
```

---

## Building for Multiple Platforms

```bash
# Linux
GOOS=linux  GOARCH=amd64 go build -o mpdl-linux-amd64 .
GOOS=linux  GOARCH=arm64 go build -o mpdl-linux-arm64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o mpdl-windows-amd64.exe .

# macOS
GOOS=darwin GOARCH=amd64  go build -o mpdl-darwin-amd64 .
GOOS=darwin GOARCH=arm64  go build -o mpdl-darwin-arm64 .   # Apple Silicon

# FreeBSD
GOOS=freebsd GOARCH=amd64 go build -o mpdl-freebsd-amd64 .
```

---

## Troubleshooting

**Cannot connect to MPD**
```bash
mpdl --debug status
# Check MPD is running:
systemctl status mpd   # Linux
net start mpd          # Windows
# Test port directly:
telnet localhost 6600
```

**Wrong track plays first with addplay**
Make sure your music files have correct `Track` and `Disc` tags. mpdl sorts by these tags — if they are missing or wrong, files fall back to filename order.

**GNTP notifications not appearing**
1. Verify Growl/GNTP server is running on the configured host/port
2. Try `icon_mode = "dataurl"` instead of `"binary"` in your config
3. Check firewall — default GNTP port is 23053

**Media keys not working on Linux**
Install `mpd-mpris` or `mpdris2` (see [Media Keys](#media-keys--bluetooth-headset) section above).

---

## Dependencies

| Package | Purpose |
|---|---|
| [fhs/gompd](https://github.com/fhs/gompd) | MPD protocol client |
| [cumulus13/go-gntp](https://github.com/cumulus13/go-gntp) | GNTP notification library |
| [BurntSushi/toml](https://github.com/BurntSushi/toml) | TOML config parsing |
| [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | YAML config parsing |
| [golang.org/x/term](https://golang.org/x/term) | Terminal size detection |

---

## Related Projects

- [MPD](https://www.musicpd.org/) — Music Player Daemon
- [MPC](https://www.musicpd.org/clients/mpc/) — Official MPD CLI client
- [ncmpcpp](https://github.com/ncmpcpp/ncmpcpp) — NCurses MPD client

---

## Author

[Hadi Cahyadi](mailto:cumulus13@gmail.com)

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)

[Patreon](https://www.patreon.com/cumulus13)

---

## License

MIT — see [LICENSE](LICENSE)