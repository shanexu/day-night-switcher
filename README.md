# Day-Night-Switcher
A simple tool that executes scripts alternately during daytime and nighttime.

## How to use
Creating the configuration file ```~/.config/day-night-switcher/config.toml```.
```toml
day_begin = '06:00:00'
night_begin = '18:00:00'
day_action = ["$HOME/bin/night-theme-switch.sh", "light"]
night_action = ["$HOME/bin/night-theme-switch.sh", "dark"]
```
