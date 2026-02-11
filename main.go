package main

import (
	"errors"
	"os/exec"
	"time"

	"github.com/a8m/envsubst"
	"github.com/jinzhu/now"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

type Config struct {
	DayBeginStr     string   `mapstructure:"day_begin"`
	NightBeginStr   string   `mapstructure:"night_begin"`
	DayAction       []string `mapstructure:"day_action"`
	NightAction     []string `mapstructure:"night_action"`
	WallpaperCron   string   `mapstructure:"wallpaper_cron"`
	WallpaperAction []string `mapstructure:"wallpaper_action"`
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
	config.WallpaperCron, err = envsubst.String(config.WallpaperCron)
	if err != nil {
		return err
	}
	config.WallpaperAction, err = envSubstStringSlice(config.WallpaperAction)
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
	if len(action) == 0 {
		return
	}
	cmd := exec.Command(action[0], action[1:]...)
	slog.Info("before exec action", "cmd", cmd)
	if err := cmd.Run(); err != nil {
		slog.Error("execute script failed", "err", err)
	}
	slog.Info("after exec action", "cmd", cmd)
}

func dayNightSwitch(variant string, dayAction, nightAction []string, wallpaperAction []string) {
	switch variant {
	case VariantLight:
		execAction(dayAction)
	case VariantDark:
		execAction(nightAction)
	}
	execAction(wallpaperAction)
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

	// Create platform-specific sleep monitor
	sleepMonitor := NewSleepMonitor()
	if err := sleepMonitor.Start(); err != nil {
		slog.Error("failed to start sleep monitor", "err", err)
		panic(err)
	}
	defer sleepMonitor.Stop()

	// Create an event channel for both sleep monitor and regular timer events
	eventChan := make(chan struct{}, 10)
	slog.Info("start")

	var timer *time.Timer
	setTimerAndSwitchDayNight := func() {
		variant, duration := dayNight(dayBeginDuration, nightBeginDuration)
		slog.Info("switch to", "variant", variant)
		dayNightSwitch(variant, config.DayAction, config.NightAction, config.WallpaperAction)
		slog.Info("sleep", "duration", duration)
		timer = time.AfterFunc(duration, func() {
			eventChan <- struct{}{}
		})
	}
	setTimerAndSwitchDayNight()

	c := cron.New()
	_, err = c.AddFunc(config.WallpaperCron, func() {
		if len(config.WallpaperAction) > 0 {
			execAction(config.WallpaperAction)
		}
	})
	if err != nil {
		slog.Warn("wallpaper action cron failed", "err", err)
	}
	c.Start()
	defer c.Stop()

	for {
		select {
		case sleepEvent := <-sleepMonitor.Events():
			if sleepEvent.IsWake {
				slog.Info("wakeup detected")
				timer.Stop()
				eventChan <- struct{}{}
			}
		case <-eventChan:
			setTimerAndSwitchDayNight()
		}
	}
}
