<h1 align="center">Lantern</h1>

<p align="center">
  <b>A lightweight LAN music streaming server written in Go.</b>
</p>

<p align="center">
Stream your personal music library from one machine to any device on your network.
</p>

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/sawthorn/lantern.git
cd lantern
````

### 2. Build

```bash
go build -o lantern cmd/main.go
```

*You can specify GOOS and GOARCH to build it for different devices.*

### 3. Run

```bash
./lantern
```

## Configuration

Lantern uses a simple config file. You will be prompted to specify some values on first launch. After that it will be accessible in:

```
~/.config/lantern/config.json
```

Example:

```json
{
  "music_dir": "/home/user/Music",
  "port": 8080
}
```

## Usage

Once running, Lantern will generate a connection QR-code as well as URL.  
You can access it on host device via:

```
http://localhost:8080
```

Or from another device on the same network:

```
http://<your-local-ip>:8080
```

Use the built-in QR code to open it instantly on mobile.

## Roadmap

### Storage & Performance

- [x] SQLite-based persistent library index
- [ ] File system watcher (fsnotify) for live updates

### Features

- [ ] Playlists
- [ ] Playback caching
- [ ] Better search & filtering

### Misc

- [ ] Vanilla CSS only

### Platform

- [ ] Responsive UI for comfortable mobile experience

## Features

- Local music streaming over HTTP (LAN)
- Minimal server-rendered web UI
- Recursive folder scanning
- QR code sharing for instant access on mobile devices
- LAN-open access (no authentication by design)
- Supports common audio formats:
  - MP3
  - FLAC
  - WAV
  - OGG
- Simple JSON configuration
  
## Tech stack

- Go
- go-taglib (metadata extraction)
- modernc/sqlite (SQLite driver for local persistence)
- Tailwind CSS *(will be replaced with vanilla css in future)*
- Vanilla server-rendered HTML

## License

Project is MIT licensed.
