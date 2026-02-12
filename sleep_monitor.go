package main

// SleepMonitor is an interface for monitoring sleep/wake events
type SleepMonitor interface {
	Start() error
	Stop() error
	Events() <-chan SleepEvent
}

type SleepEvent struct {
	IsWake bool // true for wake event, false for sleep event
}

// NewSleepMonitor creates a platform-specific sleep monitor
// useIOKit parameter is only used on macOS, it selects between polling and IOKit-based implementation
func NewSleepMonitor(useIOKit bool) SleepMonitor {
	return newPlatformSleepMonitor(useIOKit)
}

// isIOKitSupported is defined in platform-specific files for macOS
// Returns false on other platforms by default
