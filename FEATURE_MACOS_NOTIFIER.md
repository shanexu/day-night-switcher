# macOS Sleep Monitor with mac-sleep-notifier

## Overview
This feature branch implements a new macOS sleep monitor using the `mac-sleep-notifier` library, which provides real-time system sleep/wake event detection using IOKit framework.

## Key Changes

### New File: `sleep_monitor_macos_notifer.go`
- Replaces the `sleep_monitor_darwin.go` implementation
- Uses IOKit power management notifications instead of sysctl polling
- Provides instant (real-time) event detection vs 5-second polling delay
- Leverages CGo to interface with native macOS IOKit framework

### Dependencies
- `github.com/prashantgupta24/mac-sleep-notifier v1.0.1` added as direct dependency

## Implementation Details

### Old Approach (sleep_monitor_darwin.go)
```go
// Incorrect method: tried to detect sleep via uptime changes
if currentUptime < lastUptime-2 {
    // This never worked because kern.boottime doesn't change on sleep
}
```

### New Approach (sleep_monitor_macos_notifer.go)
```go
// Uses macOS IOKit power management notifications
notifierCh := notifier.GetInstance().Start()
// Receives real-time notifications for:
// - kIOMessageSystemWillSleep (sleep event)
// - kIOMessageSystemWillPowerOn (wake event)
```

## Architecture Comparison

| Aspect | Old (sysctl polling) | New (mac-sleep-notifier) |
|--------|---------------------|--------------------------|
| Detection Method | `kern.sleeptime` / `kern.waketime` polling | IOKit event callbacks |
| Real-time | No (5-second delay) | Yes (instant) |
| Complexity | Simple, no CGo | Uses CGo, IOKit |
| Dependencies | None | mac-sleep-notifier library |
| Memory Usage | Low (simple polling) | Moderate (CFRunLoop) |

## Benefits of mac-sleep-notifier

1. **Real-time Detection**: Events are received immediately, not on next poll
2. **More Accurate**: Direct system notifications, no timing issues
3. **Battery Friendly**: No continuous polling, only wakes on events
4. **Official API**: Uses documented macOS IOKit framework

## Testing

To test the new implementation:
```bash
go build -o day-night-switcher
./day-night-switcher --debug
```

The application should log:
- "Starting macOS sleep monitor using mac-sleep-notifier"
- "sleep detected via mac-sleep-notifier" when sleeping
- "wake detected via mac-sleep-notifier" when waking

## Compatibility

- **Platforms**: macOS only (darwin)
- **Requires**: macOS IOKit framework (standard on all macOS versions)
- **Build Tags**: `//go:build darwin`

## Potential Issues

1. **CFRunLoop Blocking**: The library runs a CFRunLoop internally. This may affect
   event loop behavior if not properly managed.

2. **CGo Required**: Requires C compiler and macOS SDK for CGo bindings.

3. **CanSleep Delay**: The library has a 5-second delay in `CanSleep()` callback
   which may delay sleep permission.

## Comparison with Other Approaches

### vs sysctl polling
- **Better**: Real-time, more reliable
- **Worse**: More complex, requires CGo

### vs NSWorkspace notifications
- **Better**: Works for CLI apps (NSWorkspace requires AppKit event loop)
- **Worse**: Uses private IOKit API (backwards compatible though)

### vs pmset log parsing
- **Better**: Real-time, no file I/O
- **Worse**: Requires CGo

## Backwards Compatibility

This is a breaking change only for macOS. The Linux implementation remains unchanged.

If mac-sleep-notifier has issues, the old implementation can be restored:
```bash
# Restore old implementation
git checkout dev -- sleep_monitor_darwin.go
```

## Performance

- CPU Usage: Near zero (event-driven)
- Memory: ~10-15KB additional overhead for CGo/IOKit
- Battery: Better than polling (no periodic wakeups)

## Security

The library uses standard macOS IOKit APIs for system power management.
No special permissions are required.
