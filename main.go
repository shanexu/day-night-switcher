package main

import (
	"errors"
	"os/exec"
	"time"

	"github.com/a8m/envsubst"
	"github.com/godbus/dbus/v5"
	"github.com/jinzhu/now"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

type Config struct {
	DayBeginStr   string   `mapstructure:"day_begin"`
	NightBeginStr string   `mapstructure:"night_begin"`
	DayAction     []string `mapstructure:"day_action"`
	NightAction   []string `mapstructure:"night_action"`
}

func expanEnv(config *Config) error {
	var err error
	config.DayBeginStr, err = envsubst.String(config.DayBeginStr)
	if err != nil {
		return err
	}
	config.NightBeginStr, err = envsubst.String(config.NightBeginStr)
	if err != nil {
		return err
	}
	config.DayAction, err = envSubstStringSlice(config.DayAction)
	if err != nil {
		return err
	}
	config.NightAction, err = envSubstStringSlice(config.NightAction)
	if err != nil {
		return err
	}
	return nil
}

func envSubstStringSlice(strs []string) ([]string, error) {
	var result []string
	for _, str := range strs {
		s, err := envsubst.String(str)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

const (
	VariantLight = "light"
	VariantDark  = "dark"
)

func execAction(action []string) {
	cmd := exec.Command(action[0], action[1:]...)
	slog.Info("before exec action", "cmd", cmd)
	if err := cmd.Run(); err != nil {
		slog.Error("execute script failed", "err", err)
	}
}

func dayNightSwitch(variant string, dayAction, nightAction []string) {
	switch variant {
	case VariantLight:
		execAction(dayAction)
	case VariantDark:
		execAction(nightAction)
	}
}

func dayNight(dayBeginDuration, nightBeginDuration time.Duration) (string, time.Duration) {
	n := now.New(time.Now())
	dayBegin := n.BeginningOfDay().Add(dayBeginDuration)
	nightBegin := n.BeginningOfDay().Add(nightBeginDuration)
	if n.Before(dayBegin) {
		return VariantDark, n.Sub(dayBegin) * (-1)
	}
	if n.Before(nightBegin) {
		return VariantLight, n.Sub(nightBegin) * (-1)
	}
	return VariantDark, n.Sub(dayBegin.Add(24*time.Hour)) * (-1)
}

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("$HOME/.config/day-night-switcher")
	configNotFoundError := &viper.ConfigFileNotFoundError{}
	err := viper.ReadInConfig()
	if err != nil {
		if !errors.As(err, configNotFoundError) {
			slog.Error("failed to read in config", "err", err)
			panic(err)
		}
	}

	config := &Config{
		DayBeginStr:   "06:00:00",
		NightBeginStr: "18:00:00",
	}
	if err := viper.Unmarshal(&config); err != nil {
		panic(err)
	}

	if err := expanEnv(config); err != nil {
		panic(err)
	}
	dayBegin, err := time.Parse(time.TimeOnly, config.DayBeginStr)
	if err != nil {
		panic(err)
	}
	nightBegin, err := time.Parse(time.TimeOnly, config.NightBeginStr)
	if err != nil {
		panic(err)
	}
	if !dayBegin.Before(nightBegin) {
		panic("day_begin must less then night_begin")
	}
	dayBeginDuration := now.New(dayBegin).Sub(now.New(dayBegin).BeginningOfDay())
	nightBeginDuration := now.New(nightBegin).Sub(now.New(nightBegin).BeginningOfDay())

	if len(config.DayAction) == 0 {
		panic("day_action must not be empty")
	}
	if len(config.NightAction) == 0 {
		panic("night_action must not be empty")
	}

	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		slog.Error("failed to connect to session bus", "err", err)
		panic(err)
	}
	defer conn.Close()

	if err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
	); err != nil {
		slog.Error("failed to add match signal", "err", err)
		panic(err)
	}

	dbusChan := make(chan *dbus.Signal, 10)
	conn.Signal(dbusChan)

	eventChan := make(chan struct{}, 10)
	slog.Info("start")

	var timer *time.Timer
	setTimerAndSwitchDayNight := func() {
		variant, duration := dayNight(dayBeginDuration, nightBeginDuration)
		slog.Info("switch to", "variant", variant)
		dayNightSwitch(variant, config.DayAction, config.NightAction)
		slog.Info("sleep", "duration", duration)
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
						slog.Info("wakeup")
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
