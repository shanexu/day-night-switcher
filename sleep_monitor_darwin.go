//go:build darwin

package main

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// darwinSleepMonitor implements SleepMonitor for macOS
// It uses a simpler approach: periodically checks system sleep status
// and detects wake events by monitoring for significant time jumps
type darwinSleepMonitor struct {
	events chan SleepEvent
	done   chan struct{}
	wg     sync.WaitGroup
}

func newPlatformSleepMonitor() SleepMonitor {
	return &darwinSleepMonitor{
		events: make(chan SleepEvent, 10),
		done:   make(chan struct{}),
	}
}

// Start implements SleepMonitor
func (d *darwinSleepMonitor) Start() error {
	slog.Info("Starting macOS sleep monitor")
	d.wg.Add(1)
	go d.monitor()
	return nil
}

// monitor runs the sleep monitoring loop
func (d *darwinSleepMonitor) monitor() {
	defer d.wg.Done()

	slog.Info("macOS: Using uptime-based sleep detection")

	// Get initial uptime
	lastUptime := getUptimeSecs()
	if lastUptime == 0 {
		slog.Warn("Could not get initial system uptime")
		lastUptime = 1 // Start with a default value to avoid false positives
	}

	// Track sleep state
	wasSleeping := false
	lastSleepTime := time.Time{}

	for {
		select {
		case <-d.done:
			return

		case <-time.After(5 * time.Second):
			currentUptime := getUptimeSecs()
			slog.Debug("uptime check", "last", lastUptime, "current", currentUptime)

			if currentUptime == 0 {
				slog.Warn("Could not get system uptime")
				continue
			}

			// Detect sleep: uptime goes down significantly
			if currentUptime < lastUptime-2 {
				slog.Info("sleep detected via uptime drop", "old", lastUptime, "new", currentUptime)
				wasSleeping = true
				lastSleepTime = time.Now()

				select {
				case d.events <- SleepEvent{IsWake: false}:
					slog.Info("sent sleep event")
				default:
					slog.Warn("event channel full, dropping sleep event")
				}
			}

			// Detect wake: time has advanced but uptime stayed same or increased
			if wasSleeping && time.Since(lastSleepTime) > time.Duration(2)*time.Second {
				// We're after a sleep event and time has passed
				// This indicates the system woke up
				slog.Info("wakeup detected", "elapsed", time.Since(lastSleepTime))
				wasSleeping = false

				select {
				case d.events <- SleepEvent{IsWake: true}:
					slog.Info("sent wake event")
				default:
					slog.Warn("event channel full, dropping wake event")
				}
			}

			lastUptime = currentUptime
		}
	}
}

// Stop implements SleepMonitor
func (d *darwinSleepMonitor) Stop() error {
	close(d.done)
	d.wg.Wait()
	return nil
}

// Events implements SleepMonitor
func (d *darwinSleepMonitor) Events() <-chan SleepEvent {
	return d.events
}

// getUptimeSecs gets the system uptime in seconds using sysctl
func getUptimeSecs() int64 {
	// Use sysctl to get system uptime
	// kern.boottime gives the boot time, we can calculate uptime from it
	cmd := exec.Command("sysctl", "-n", "kern.boottime")
	output, err := cmd.Output()
	if err != nil {
		slog.Warn("failed to get boot time via sysctl", "err", err)
		return 0
	}

	// Parse the output: { sec = 1234567, usec = 0 }
	// We need to extract the seconds and calculate uptime
	outputStr := strings.TrimSpace(string(output))

	// Try to extract the timestamp
	var bootSec int64
	fields := strings.Fields(outputStr)
	for i, field := range fields {
		if field == "sec" && i+2 < len(fields) {
			// Next field should be "=", next should be the number
			trimmedNum := strings.Trim(fields[i+2], ",")
			if num, err := strconv.ParseInt(trimmedNum, 10, 64); err == nil {
				bootSec = num
				break
			}
		}
	}

	if bootSec == 0 {
		slog.Warn("could not parse boot time")
		return 0
	}

	// Calculate uptime
	currentTime := time.Now().Unix()
	uptime := currentTime - bootSec

	return uptime
}
