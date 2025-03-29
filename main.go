package main

import (
	"calendar-summary/internal/calendar"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"time"

	"github.com/titanous/json5"
)

type ServerConfig struct {
	Url      string `json:"url"`
	Insecure bool   `json:"insecure"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	Server       ServerConfig `json:"server"`
	CalendarName string       `json:"calendar_name"`
}

func readConfig(path string) (Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	err = json5.Unmarshal(contents, &cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func fatalerr(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func printCal(ctx context.Context, source calendar.Source, calendarName string) error {
	calList, err := source.Calendars(ctx)
	if err != nil {
		return err
	}
	var cal calendar.Calendar
	for _, c := range calList {
		if c.Name == calendarName {
			cal = c
			break
		}
	}
	if cal.Id == "" {
		return fmt.Errorf("find calendar: not found '%s'", calendarName)
	}

	events, err := source.Events(ctx, cal, time.Time{}, time.Now().Add(365*24*time.Hour))
	if err != nil {
		return err
	}
	slices.SortFunc(events, func(a, b calendar.Event) int {
		return a.Start.Compare(b.Start)
	})

	for _, e := range events {
		slog.Info(
			"EVENT",
			"name", e.Name,
			"start", e.Start.Format(time.RFC1123),
			"end", e.End.Format(time.RFC1123),
			"tags", e.Tags,
		)
	}
	return nil
}

func main() {
	cfgPath := flag.String("config", "config.json5", "Configuration path.")
	flag.Parse()

	cfg, err := readConfig(*cfgPath)
	if err != nil {
		fatalerr("read config", "err", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	source, err := calendar.NewCaldav(cfg.Server.Url, calendar.CaldavOptions{
		Username: cfg.Server.Username,
		Password: cfg.Server.Password,
		Insecure: true,
	})
	if err != nil {
		fatalerr("create caldav source", "err", err)
	}

	err = printCal(ctx, source, cfg.CalendarName)
	if err != nil {
		fatalerr("print calendar", "err", err)
	}
}
