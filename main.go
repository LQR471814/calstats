package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"schedule-statistics/api/v1/v1connect"
	"schedule-statistics/internal/calendar"
	"schedule-statistics/internal/tel"
	"slices"
	"time"

	"connectrpc.com/connect"
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
	Server    ServerConfig `json:"server"`
	Calendars []string     `json:"calendars"`
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

func printCal(ctx context.Context, source calendar.Source, calname string) error {
	calList, err := source.Calendars(ctx)
	if err != nil {
		return err
	}
	var cal calendar.Calendar
	for _, c := range calList {
		if c.Name == calname {
			cal = c
			break
		}
	}
	if cal.Id == "" {
		return fmt.Errorf("find calendar: not found '%s'", calname)
	}

	now := time.Now()

	start := now.Add(-time.Duration(now.Hour()) * time.Hour)
	start = start.Add(-time.Duration(now.Minute()) * time.Minute)
	start = start.Add(-time.Duration(now.Second()) * time.Second)
	start = start.Add(-time.Duration(now.Nanosecond()) * time.Nanosecond)

	end := start.Add(time.Hour*23 + time.Minute*59 + time.Second*59 + time.Nanosecond*999999999)

	events, err := source.Events(ctx, cal, start, end, time.Local)
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
			"start", e.Start.In(time.Local).Format(time.RFC1123),
			"end", e.End.In(time.Local).Format(time.RFC1123),
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

//go:embed ui/dist/*
var ui embed.FS

func main() {
	cfgPath := flag.String("config", "config.json5", "Configuration path.")
	port := flag.Int("port", 3000, "The port to host on.")
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
		for _, cal := range cfg.Calendars {
			tel.Log.Info("main", "===========", "calendar", cal)
			err = printCal(ctx, source, cal)
			if err != nil {
				fatalerr("print calendar", "err", err)
			}
		}
		return
	}

	tel.Log.Info("main", "listening on...", "port", *port)

	mux := http.NewServeMux()

	buildFs, err := fs.Sub(ui, "ui/dist")
	if err != nil {
		fatalerr("access ui assets", "err", err)
	}
	mux.Handle("/", http.FileServerFS(buildFs))

	handle, handler := v1connect.NewCalendarServiceHandler(
		CalendarService{
			calendars:      cfg.Calendars,
			calendarServer: cfg.Server.Url,
			source:         source,
		},
		connect.WithInterceptors(
			connect.UnaryInterceptorFunc(tel.LogErrorsInterceptor),
		),
	)
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
