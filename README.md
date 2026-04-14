# clash-cli

A keyboard-driven TUI client for [mihomo](https://github.com/MetaCubeX/mihomo) (Clash Meta), inspired by [Clash Verge Rev](https://github.com/clash-verge-rev/clash-verge-rev) but built for the terminal.

The command is `clashc`.

```
┌─ [Home] [Profiles] [Proxies] [Connections] [Rules] [Logs] [Settings] ─┐
│                                                                        │
│   Traffic     ↑ 1.23 MB/s   ↓ 5.67 MB/s                                │
│   Memory      45 MB     Connections  42                                │
│   Mode: rule  Core: running  SysProxy: ON                              │
│                                                                        │
│   Quick: [m] mode  [s] sysproxy  [t] tun  [r] restart                  │
└────────────────────────────────────────────────────────────────────────┘
 Profile: MySub │ Mode: rule │ ↑1.2MB/s ↓3.4MB/s │ Mem: 45M │ SysProxy: ON
```

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/boomyao/clash-cli/main/install.sh | sh
```

That installs:

- `clashc` itself
- `mihomo` core (only if not already on your PATH)

By default it goes to `~/.local/bin`. Run as root and it goes to `/usr/local/bin`.

If `~/.local/bin` isn't on your `PATH` yet, the installer will tell you which line to add to your shell rc.

## Usage

```bash
# First run: import a subscription URL
clashc https://example.com/your-sub.yaml

# After that: just `clashc` — it remembers the last subscription
clashc
```

`clashc` will:
1. Ensure mihomo's config exists (downloads your subscription, applies safety patches like rewriting DNS port 53 → 1053)
2. Auto-launch mihomo as a child process if it isn't already running
3. Start the TUI
4. On exit, stop mihomo if `clashc` was the one that started it

### Flags

```
clashc [flags] [subscription-url]

  --api-url string      mihomo external controller URL (default 127.0.0.1:9090)
  --secret string       mihomo API secret
  --mihomo-bin string   path to mihomo binary
  --config-dir string   mihomo config directory (default ~/.config/mihomo)
  --no-autostart        don't auto-launch mihomo if it's not running
  --version
```

## Features

- 📊 **Live dashboard** — real-time traffic, memory, connection count
- 📋 **Profile management** — import subscription URLs, switch profiles, auto-update
- 🌐 **Proxy switching** — browse groups, select nodes, batch latency tests
- 🔗 **Connection monitor** — live table of active connections, filter, close one or all
- 📜 **Rules viewer** — searchable list of routing rules
- 📝 **Log streaming** — real-time logs with level filtering and pause/resume
- ⚙️ **Settings** — toggle mode, TUN, allow-LAN, restart core, flush caches
- 🔌 **System proxy** — one-key toggle (macOS / Linux GNOME)
- 🎨 **Polished TUI** — vim-style navigation, tab switching, toast notifications

## Keybindings

### Global
| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Switch tabs |
| `1`–`7` | Jump to specific tab |
| `s` | Toggle system proxy |
| `q` / `ctrl+c` | Quit |

### Page-specific
- **Proxies**: `j/k` navigate · `tab` switch pane · `enter` select · `t` test latency
- **Profiles**: `a` add · `enter` activate · `u` update · `d` delete
- **Connections**: `j/k` navigate · `x` close · `X` close all · `/` filter
- **Rules**: `j/k` navigate · `/` filter · `g/G` top/bottom
- **Logs**: `l` cycle level · `/` filter · `p` pause · `c` clear · `g/G` top/bottom
- **Settings**: `j/k` navigate · `enter` toggle/run

## Files

| Path | Purpose |
|------|---------|
| `~/.config/clash-cli/config.yaml` | clashc app settings (api URL, profiles list, etc.) |
| `~/.local/share/clash-cli/profiles/` | Downloaded subscription YAMLs |
| `~/.local/share/clash-cli/mihomo.log` | Stdout/stderr from auto-launched mihomo |
| `~/.config/mihomo/config.yaml` | Active mihomo runtime config |

## Build from source

```bash
git clone https://github.com/boomyao/clash-cli
cd clash-cli
make build
./clashc
```

Requires Go 1.23+.

## Architecture

- **Go** + [bubbletea](https://github.com/charmbracelet/bubbletea) (Elm Architecture)
- **lipgloss** for styling, **bubbles** for components
- **gorilla/websocket** for real-time streams
- API client and process manager are decoupled from the TUI

```
internal/
├── api/         # mihomo REST API client
├── ws/          # WebSocket streams (traffic/memory/logs/connections)
├── core/        # mihomo subprocess lifecycle
├── sysproxy/    # platform-specific system proxy (darwin/linux)
├── profile/     # subscription management
├── bootstrap/   # pre-TUI setup: download URL, ensure mihomo running
├── config/      # clashc app config (XDG paths)
├── tui/         # bubbletea UI
│   ├── components/  # tabs, statusbar, toast
│   └── pages/       # 7 tabs
└── util/        # formatting helpers
```

## License

MIT — see [LICENSE](./LICENSE).
