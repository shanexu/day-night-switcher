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
func NewSleepMonitor() SleepMonitor {
	return newPlatformSleepMonitor()
}
