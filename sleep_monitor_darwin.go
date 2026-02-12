//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// darwinSleepMonitor implements SleepMonitor for macOS
// It monitors kern.sleeptime and kern.waketime to detect sleep/wake events
type darwinSleepMonitor struct {
	events chan SleepEvent
	done   chan struct{}
	wg     sync.WaitGroup

	lastSleepTime int64
	lastWakeTime  int64
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

	// Get initial sleep/wake times
	d.lastSleepTime, _ = getSleepTime()
	d.lastWakeTime, _ = getWakeTime()
	slog.Info("Initial sleep/wake times", "sleep", d.lastSleepTime, "wake", d.lastWakeTime)

	d.wg.Add(1)
	go d.monitor()
	return nil
}

// monitor runs the sleep monitoring loop
func (d *darwinSleepMonitor) monitor() {
	defer d.wg.Done()

	slog.Info("macOS: Monitoring kern.sleeptime and kern.waketime for sleep/wake events")

	for {
		select {
		case <-d.done:
			return

		case <-time.After(5 * time.Second):
			currentSleepTime, err := getSleepTime()
			if err != nil {
				slog.Warn("failed to get sleep time", "err", err)
				continue
			}

			currentWakeTime, err := getWakeTime()
			if err != nil {
				slog.Warn("failed to get wake time", "err", err)
				continue
			}

			slog.Debug("sleep/wake time check", "sleep", currentSleepTime, "wake", currentWakeTime,
				"lastSleep", d.lastSleepTime, "lastWake", d.lastWakeTime)

			// Detect sleep event: kern.sleeptime has changed
			if currentSleepTime > d.lastSleepTime {
				slog.Info("sleep detected", "old", d.lastSleepTime, "new", currentSleepTime)
				d.lastSleepTime = currentSleepTime

				select {
				case d.events <- SleepEvent{IsWake: false}:
					slog.Info("sent sleep event")
				default:
					slog.Warn("event channel full, dropping sleep event")
				}
			}

			// Detect wake event: kern.waketime has changed
			if currentWakeTime > d.lastWakeTime {
				slog.Info("wake detected", "old", d.lastWakeTime, "new", currentWakeTime)
				d.lastWakeTime = currentWakeTime

				select {
				case d.events <- SleepEvent{IsWake: true}:
					slog.Info("sent wake event")
				default:
					slog.Warn("event channel full, dropping wake event")
				}
			}
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

// parseSysctlTimestamp parses a sysctl timestamp output like "{ sec = 1234567, usec = 472008 }"
func parseSysctlTimestamp(output string) (int64, error) {
	output = strings.TrimSpace(output)

	// Find sec value - handles both "sec = X" and "usec = X, sec = Y" formats
	fields := strings.Fields(output)
	for i, field := range fields {
		if field == "sec" && i+2 < len(fields) {
			// Check if next field is "="
			if fields[i+1] != "=" {
				continue
			}
			// Next field after "=" should be the number
			trimmedNum := strings.Trim(fields[i+2], ",")
			if num, err := strconv.ParseInt(trimmedNum, 10, 64); err == nil {
				return num, nil
			}
		}
	}

	return 0, fmt.Errorf("could not parse sec value from: %s", output)
}

// getSleepTime gets the last sleep time in Unix timestamp seconds
func getSleepTime() (int64, error) {
	cmd := exec.Command("sysctl", "-n", "kern.sleeptime")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return parseSysctlTimestamp(string(output))
}

// getWakeTime gets the last wake time in Unix timestamp seconds
func getWakeTime() (int64, error) {
	cmd := exec.Command("sysctl", "-n", "kern.waketime")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return parseSysctlTimestamp(string(output))
}
