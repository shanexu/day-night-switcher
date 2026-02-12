//go:build darwin

package main

import (
	"github.com/prashantgupta24/mac-sleep-notifier/notifier"
	"golang.org/x/exp/slog"
)

// macosNotifierSleepMonitor implements SleepMonitor for macOS
// It uses the mac-sleep-notifier library which leverages IOKit for system power notifications
type macosNotifierSleepMonitor struct {
	events chan SleepEvent
	done   chan struct{}
	notifier *notifier.Notifier
}

func newPlatformSleepMonitor() SleepMonitor {
	return &macosNotifierSleepMonitor{
		events: make(chan SleepEvent, 10),
		done:   make(chan struct{}),
	}
}

// Start implements SleepMonitor
func (m *macosNotifierSleepMonitor) Start() error {
	slog.Info("Starting macOS sleep monitor using mac-sleep-notifier")

	// Get the notifier instance
	m.notifier = notifier.GetInstance()

	// Start the notifier
	notifierCh := m.notifier.Start()

	// Start a goroutine to receive events and convert them
	go func() {
		for {
			select {
			case <-m.done:
				slog.Info("stopping mac-sleep-notifier receiver")
				return
			case activity := <-notifierCh:
				slog.Debug("received activity from mac-sleep-notifier", "type", activity.Type)

				// Convert mac-sleep-notifier activity to our SleepEvent
				var isWake bool
				switch activity.Type {
				case notifier.Awake:
					isWake = true
					slog.Info("wake detected via mac-sleep-notifier")
				case notifier.Sleep:
					isWake = false
					slog.Info("sleep detected via mac-sleep-notifier")
				default:
					slog.Warn("unknown activity type", "type", activity.Type)
					continue
				}

				select {
				case m.events <- SleepEvent{IsWake: isWake}:
					slog.Info("sent sleep/wake event")
				default:
					slog.Warn("event channel full, dropping sleep/wake event")
				}
			}
		}
	}()

	return nil
}

// Stop implements SleepMonitor
func (m *macosNotifierSleepMonitor) Stop() error {
	close(m.done)
	if m.notifier != nil {
		m.notifier.Quit()
	}
	return nil
}

// Events implements SleepMonitor
func (m *macosNotifierSleepMonitor) Events() <-chan SleepEvent {
	return m.events
}
