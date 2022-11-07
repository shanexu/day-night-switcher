package main

import (
	"log"
	"os/exec"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/jinzhu/now"
)

func dayNightThemeSwitch(str string) {
	cmd := exec.Command("/home/shane/bin/night-theme-switch.sh", str)
	if err := cmd.Run(); err != nil {
		log.Println(err)
	}
}

func dayNight() (string, time.Duration) {
	n := now.New(time.Now())
	dayBegin := n.BeginningOfDay().Add(6 * time.Hour)
	nightBegin := n.BeginningOfDay().Add(18 * time.Hour)
	if n.Before(dayBegin) {
		return "dark", n.Sub(dayBegin) * (-1)
	}
	if n.Before(nightBegin) {
		return "light", n.Sub(nightBegin) * (-1)
	}
	return "dark", n.Sub(dayBegin.Add(24*time.Hour)) * (-1)
}

func main() {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Fatalln("Failed to connect to session bus:", err)
	}
	defer conn.Close()

	if err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
	); err != nil {
		panic(err)
	}

	dbusChan := make(chan *dbus.Signal, 10)
	conn.Signal(dbusChan)

	eventChan := make(chan struct{}, 10)
	log.Println("start")

	var timer *time.Timer
	setTimerAndSwitchDayNight := func() {
		variant, duration := dayNight()
		log.Println("switch to", variant)
		dayNightThemeSwitch(variant)
		log.Println("sleep", duration)
		timer = time.AfterFunc(duration, func() {
			eventChan <- struct{}{}
		})
	}
	setTimerAndSwitchDayNight()

	for {
		select {
		case signal := <-dbusChan:
			if signal.Name == "org.freedesktop.login1.Manager.PrepareForSleep" {
				if len(signal.Body) == 1 {
					prepareForSleep, ok := signal.Body[0].(bool)
					if ok && !prepareForSleep {
						log.Println("wakeup")
						timer.Stop()
						eventChan <- struct{}{}
					}
				}
			}
		case <-eventChan:
			setTimerAndSwitchDayNight()
		}
	}
}
