# Day-Night-Switcher

A simple tool that executes scripts alternately during daytime and nighttime.

This tool runs as a background daemon and automatically switches between day/light and night/dark themes based on a configurable schedule. It also supports wake-from-sleep detection to immediately update the theme after system resume.

## Features

- **Cross-platform**: Works on both Linux and macOS
- **Automatic theme switching**: Based on configurable time windows
- **Sleep/wake detection**: Automatically resumes and updates themes after system sleep
- **Wallpaper scheduling**: Optional cron-based wallpaper updates
- **Environment variable support**: Built-in `$HOME` and environment variable expansion

## Platform Support

| Platform | Sleep Detection Method |
|----------|----------------------|
| Linux | D-Bus signals from systemd-logind |
| macOS | Polling via `kern.sleeptime`/`kern.waketime` or IOKit (optional) |

## Installation

### Prerequisites

- **Linux**: D-Bus and systemd-logind
- **macOS**: Built-in `sysctl` command (always available)
- **Go 1.20+** (for building from source)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/shanexu/day-night-switcher.git
cd day-night-switcher

# Build for current platform (default - uses polling)
go build -o day-night-switcher

# Or build for specific platform
GOOS=linux go build -o day-night-switcher-linux        # Linux
GOOS=darwin go build -o day-night-switcher-darwin      # macOS
```

#### Advanced: Build with IOKit Support (macOS Only)

For real-time sleep/wake detection on macOS:

```bash
# 1. Install Xcode command line tools
xcode-select --install

# 2. Build with CGo and iokit tag
CGO_ENABLED=1 go build -tags iokit -o day-night-switcher

# 3. Configure in config.toml
sleep_monitor = "native"
```

See the [Sleep Monitor Configuration](SLEEP_MONITOR_CONFIG.md) document for complete details.

### Installation

```bash
# Install to /usr/local/bin (requires sudo)
sudo cp day-night-switcher /usr/local/bin/

# Or install to user's bin directory
cp day-night-switcher ~/bin/
```

## Configuration

### Creating the Configuration File

Create a configuration file at `~/.config/day-night-switcher/config.toml`:

```toml
# Day and night begin times (24-hour format) - optional, defaults to 06:00:00 and 18:00:00
day_begin = '06:00:00'
night_begin = '18:00:00'

# Commands to run when switching to day/night mode - optional
day_action = ["$HOME/bin/night-theme-switch.sh", "light"]
night_action = ["$HOME/bin/night-theme-switch.sh", "dark"]

# Optional: Wallpaper update schedule (cron format)
wallpaper_cron = "@hourly"
wallpaper_action = ["set-wallpaper.sh"]

# macOS only: Sleep monitor implementation (optional)
# Options: "polling" (default, uses sysctl), "native" (uses IOKit, requires special build)
sleep_monitor = "polling"
```

### Configuration Options

| Option | Description | Required |
|--------|-------------|----------|
| `day_begin` | Start time for day mode (HH:MM:SS) | Optional (default: 06:00:00) |
| `night_begin` | Start time for night mode (HH:MM:SS) | Optional (default: 18:00:00) |
| `day_action` | Command to run for day mode | Optional |
| `night_action` | Command to run for night mode | Optional |
| `wallpaper_cron` | Cron schedule for wallpaper updates | Optional |
| `wallpaper_action` | Command to run for wallpaper updates | Optional |
| `sleep_monitor` | macOS sleep monitor implementation: `polling` (default) or `native` | Optional (macOS only) |

### Environment Variable Expansion

The configuration supports environment variable substitution using `${VAR}` or `$VAR` syntax:

```toml
day_action = ["${HOME}/bin/theme-switch.sh", "light"]
```

## Usage

### Sleep Monitor Configuration (macOS)

The macOS version supports two different sleep monitor implementations:

#### Polling-based (Default - Recommended)
Uses `sysctl` to poll `kern.sleeptime` and `kern.waketime` every 5 seconds:
```toml
sleep_monitor = "polling"  # Default
```
**Pros**: Simple, no CGo required, works everywhere
**Cons**: Up to 5 seconds delay in detection

#### IOKit-based (Optional - Advanced)
Uses macOS IOKit framework for real-time notifications:
```toml
sleep_monitor = "native"
```

**Requirements**:
- Must build with special flags: `CGO_ENABLED=1 go build -tags iokit`
- Requires Xcode command line tools
- Available for users needing real-time event detection

**Pros**: Zero latency, reliable, no polling overhead
**Cons**: Complex build process, macOS-specific

To build with IOKit support:
```bash
# Install Xcode command line tools
xcode-select --install

# Build with CGo
CGO_ENABLED=1 go build -tags iokit -o day-night-switcher
```

> **Note**: If you use `sleep_monitor = "native"` without building with the `iokit` tag, the application will automatically fall back to polling and show a warning message.

### Running as Foreground Process

```bash
./day-night-switcher
```

### Running as Background Daemon (Linux Systemd Example)

Create a systemd service file `/etc/systemd/system/day-night-switcher.service`:

```ini
[Unit]
Description=Day Night Switcher
After=graphical-session.target

[Service]
Type=simple
ExecStart=/usr/local/bin/day-night-switcher
Restart=always
RestartSec=10
Environment=DISPLAY=:0

[Install]
WantedBy=default.target
```

Enable and start the service:

```bash
sudo systemctl enable day-night-switcher.service
sudo systemctl start day-night-switcher.service
sudo systemctl status day-night-switcher.service
```

### Running as Background Daemon (macOS Launchd Example)

Create a plist file `~/Library/LaunchAgents/com.user.daynightswitcher.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.daynightswitcher</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/day-night-switcher</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/daynightswitcher.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/daynightswitcher.err</string>
</dict>
</plist>
```

Load and start the service:

```bash
launchctl load ~/Library/LaunchAgents/com.user.daynightswitcher.plist
launchctl start com.user.daynightswitcher
```

## Example Scripts

### Simple Theme Switcher Script

Create `~/bin/night-theme-switch.sh`:

```bash
#!/bin/bash

if [ "$1" = "light" ]; then
    # Light theme commands
    gsettings set org.gnome.desktop.interface gtk-theme "Adwaita" 2>/dev/null || true
    gsettings set org.gnome.desktop.interface color-scheme "prefer-light" 2>/dev/null || true
elif [ "$1" = "dark" ]; then
    # Dark theme commands
    gsettings set org.gnome.desktop.interface gtk-theme "Adwaita-dark" 2>/dev/null || true
    gsettings set org.gnome.desktop.interface color-scheme "prefer-dark" 2>/dev/null || true
fi
```

Make it executable:

```bash
chmod +x ~/bin/night-theme-switch.sh
```

## Default Behavior

If no configuration file is found, the tool uses these defaults:
- **day_begin**: `06:00:00`
- **night_begin**: `18:00:00`

## Logs

The application logs to stdout/stderr. You can capture logs to a file:

```bash
./day-night-switcher > /tmp/day-night-switcher.log 2>&1
```

## Development

### Build Targets

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o day-night-switcher-linux
GOOS=linux GOARCH=arm64 go build -o day-night-switcher-linux-arm64

# macOS
GOOS=darwin GOARCH=amd64 go build -o day-night-switcher-darwin
GOOS=darwin GOARCH=arm64 go build -o day-night-switcher-darwin-arm64 (Apple Silicon)
```

### Code Structure

- `main.go` - Main application logic
- `sleep_monitor.go` - Sleep/wake monitoring interface
- `sleep_monitor_linux.go` - Linux-specific D-Bus implementation
- `sleep_monitor_darwin.go` - macOS polling-based implementation (default)
- `config.example.toml` - Configuration file example
- `SLEEP_MONITOR_CONFIG.md` - Detailed sleep monitor configuration guide
- `IMPLEMENTATION_SUMMARY.md` - Technical implementation details

## Troubleshooting

### No Events After Sleep

- **Linux**: Ensure systemd-logind is running (`systemctl status systemd-logind`)
- **macOS**:
  - Check `sysctl kern.sleeptime` works properly
  - If using `sleep_monitor = "native"`, ensure you built with `CGO_ENABLED=1 go build -tags iokit`
  - If you see warning about IOKit not available, use `sleep_monitor = "polling"` instead

### Sleep Monitor Performance Comparison

| Feature | Polling (Default) | IOKit (Native) |
|---------|-------------------|----------------|
| Detection Delay | 0-5 seconds | <100ms |
| CPU Usage | ~0.1% (polling overhead) | ~0.0% (event-driven) |
| Memory Usage | ~100KB | ~150KB |
| Binary Size | ~7.7MB | ~8.5MB |
| Build Complexity | Simple | Requires CGo + iokit tag |
| Reliability | Good | Excellent |

**Recommendation**: Use the default polling implementation unless you need real-time event detection.

### Empty Config File

All configuration options are optional. If no actions are specified, only time-based switching will occur without executing any commands.

### Permission Issues

Ensure the script paths in `day_action` and `night_action` are executable:
```bash
chmod +x /path/to/your/script.sh
```

## License

MIT License - see LICENSE file for details.
