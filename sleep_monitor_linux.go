//go:build linux

package main

import (
	"github.com/godbus/dbus/v5"
	"golang.org/x/exp/slog"
)

type linuxSleepMonitor struct {
	conn   *dbus.Conn
	events chan SleepEvent
	done   chan struct{}
}

func newPlatformSleepMonitor() SleepMonitor {
	return &linuxSleepMonitor{
		events: make(chan SleepEvent, 10),
		done:   make(chan struct{}),
	}
}

func (l *linuxSleepMonitor) Start() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		slog.Error("failed to connect to session bus", "err", err)
		return err
	}
	l.conn = conn

	if err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
	); err != nil {
		slog.Error("failed to add match signal", "err", err)
		return err
	}

	dbusChan := make(chan *dbus.Signal, 10)
	conn.Signal(dbusChan)

	go l.monitor(dbusChan)
	return nil
}

func (l *linuxSleepMonitor) monitor(dbusChan <-chan *dbus.Signal) {
	for {
		select {
		case <-l.done:
			return
		case signal := <-dbusChan:
			if signal.Name == "org.freedesktop.login1.Manager.PrepareForSleep" {
				if len(signal.Body) == 1 {
					prepareForSleep, ok := signal.Body[0].(bool)
					if ok {
						if !prepareForSleep {
							slog.Info("wakeup detected")
							l.events <- SleepEvent{IsWake: true}
						} else {
							slog.Info("sleep detected")
							l.events <- SleepEvent{IsWake: false}
						}
					}
				}
			}
		}
	}
}

func (l *linuxSleepMonitor) Stop() error {
	close(l.done)
	if l.conn != nil {
		l.conn.Close()
	}
	return nil
}

func (l *linuxSleepMonitor) Events() <-chan SleepEvent {
	return l.events
}
