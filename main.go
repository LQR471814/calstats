package main

import (
	"calendar-summary/api/v1/v1connect"
	"calendar-summary/internal/calendar"
	"calendar-summary/internal/tel"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"time"

	connectcors "connectrpc.com/cors"
	"github.com/rs/cors"
	"github.com/titanous/json5"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	tel.Log.Error("main", msg, args...)
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

	now := time.Now()
	start := now.AddDate(0, -1, 0)
	end := now.AddDate(0, 1, 0)

	events, err := source.Events(ctx, cal, start, end)
	if err != nil {
		return err
	}
	slices.SortFunc(events, func(a, b calendar.Event) int {
		return a.Start.Compare(b.Start)
	})

	for _, e := range events {
		tel.Log.Info(
			"main",
			"EVENT",
			"name", e.Name,
			"start", e.Start.Format(time.RFC1123),
			"end", e.End.Format(time.RFC1123),
			"tags", e.Tags,
		)
	}
	return nil
}

func withCORS(connectHandler http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
	})
	return c.Handler(connectHandler)
}

func main() {
	cfgPath := flag.String("config", "config.json5", "Configuration path.")
	port := flag.Int("port", 8003, "The port to host on.")
	debug := flag.Bool("debug", false, "Print the contents of the configured calendar and exit.")
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

	if *debug {
		err = printCal(ctx, source, cfg.CalendarName)
		if err != nil {
			fatalerr("print calendar", "err", err)
		}
		return
	}

	tel.Log.Info("main", "listening to gRPC...", "port", *port)

	mux := http.NewServeMux()
	handle, handler := v1connect.NewCalendarServiceHandler(CalendarService{
		calendarName: cfg.CalendarName,
		source:       source,
	})
	mux.Handle(handle, withCORS(handler))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			fatalerr("listen server", "err", err)
		}
	}()

	<-ctx.Done()
}
