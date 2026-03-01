# tweb

Lightweight web-based terminal multiplexer. Single binary, runs on Linux/macOS/Windows.

## Install

```bash
sudo curl -Lo /usr/local/bin/tweb https://github.com/roraja/tweb/releases/latest/download/tweb-linux-amd64 && sudo chmod +x /usr/local/bin/tweb
```

## Quick Start

```bash
make build
./tweb
```

Open `http://localhost:8080` in your browser.

## Config

Create `~/tweb.yml`:

```yaml
port: 8080
password: "your-secret"
shell: "/bin/bash"
```

All fields are optional. Defaults: port `8080`, no password, auto-detected shell.

## Flags

```
--port PORT    Override listening port
--config PATH  Custom config file path
```

## Features

- **Tabbed terminals** — create multiple sessions, switch with clicks or `Ctrl+Shift+T`
- **Password auth** — optional, via config file
- **Auto-resize** — terminals adapt to browser window size
- **256 colors** — full color support via xterm.js
- **Single binary** — HTML/CSS embedded, just copy and run
- **Cross-platform** — Linux, macOS, Windows

## Cross-Compile

```bash
make build-all
```

Produces binaries for linux/darwin/windows (amd64 + arm64).

## Dependencies (client-side)

The terminal UI loads [xterm.js](https://xtermjs.org/) from CDN. The client browser needs internet access on first load (cached thereafter).
