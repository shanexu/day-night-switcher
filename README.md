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
| macOS | Uptime monitoring via `sysctl` |

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

# Build for current platform
go build -o day-night-switcher

# Or build for specific platform
GOOS=linux go build -o day-night-switcher-linux        # Linux
GOOS=darwin go build -o day-night-switcher-darwin      # macOS
```

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
# Day and night begin times (24-hour format)
day_begin = '06:00:00'
night_begin = '18:00:00'

# Commands to run when switching to day/night mode
day_action = ["$HOME/bin/night-theme-switch.sh", "light"]
night_action = ["$HOME/bin/night-theme-switch.sh", "dark"]

# Optional: Wallpaper update schedule (cron format)
# wallpaper_cron = "@hourly"
# wallpaper_action = ["set-wallpaper.sh"]
```

### Configuration Options

| Option | Description | Required |
|--------|-------------|----------|
| `day_begin` | Start time for day mode (HH:MM:SS) | Yes |
| `night_begin` | Start time for night mode (HH:MM:SS) | Yes |
| `day_action` | Command to run for day mode | Yes |
| `night_action` | Command to run for night mode | Yes |
| `wallpaper_cron` | Cron schedule for wallpaper updates | Optional |
| `wallpaper_action` | Command to run for wallpaper updates | Optional |

### Environment Variable Expansion

The configuration supports environment variable substitution using `${VAR}` or `$VAR` syntax:

```toml
day_action = ["${HOME}/bin/theme-switch.sh", "light"]
```

## Usage

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
- `sleep_monitor_darwin.go` - macOS-specific uptime-based implementation

## Troubleshooting

### No Events After Sleep

- **Linux**: Ensure systemd-logind is running (`systemctl status systemd-logind`)
- **macOS**: Check `sysctl kern.boottime` works properly

### Empty Config File

If you only specify `day_action` and `night_action`, the wallpaper features are optional and won't cause errors.

### Permission Issues

Ensure the script paths in `day_action` and `night_action` are executable:
```bash
chmod +x /path/to/your/script.sh
```

## License

MIT License - see LICENSE file for details.
