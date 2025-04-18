package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"schedule-utils/api/v1/v1connect"
	"schedule-utils/internal/calendar"
	"schedule-utils/internal/tel"
	"slices"
	"time"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"github.com/hujun-open/cobra"
	"github.com/hujun-open/myflags/v2"
	"github.com/rs/cors"
	"github.com/titanous/json5"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

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

type cliServe struct {
	Port int `usage:"The port to host on."`
}

type Cli struct {
	Config string   `usage:"Configuration file path."`
	Serve  cliServe `action:"RunServe" usage:"Serve UI and API."`
	Debug  struct{} `action:"RunDebug" usage:"Print events from calendar."`
}

func (c Cli) readConfig() Config {
	contents, err := os.ReadFile(c.Config)
	if err != nil {
		fatalerr("read config", "err", err)
	}
	var cfg Config
	err = json5.Unmarshal(contents, &cfg)
	if err != nil {
		fatalerr("read config", "err", err)
	}
	return cfg
}

func (c Cli) getCalendar(cfg ServerConfig) calendar.Source {
	source, err := calendar.NewCaldav(cfg.Url, calendar.CaldavOptions{
		Username: cfg.Username,
		Password: cfg.Password,
		Insecure: true,
	})
	if err != nil {
		fatalerr("create caldav source", "err", err)
	}
	return source
}

func (c Cli) RunDebug(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	cfg := c.readConfig()
	for _, src := range cfg {
		source := c.getCalendar(src.Server)
		for _, cal := range src.Calendars {
			tel.Log.Info("main", "===========", "calendar", cal)
			err := printCal(ctx, source, cal)
			if err != nil {
				fatalerr("print calendar", "err", err)
			}
		}
	}
}

func (c Cli) RunServe(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	cfg := c.readConfig()
	port := c.Serve.Port

	tel.Log.Info("main", "listening on...", "port", c.Serve.Port)

	mux := http.NewServeMux()

	buildFs, err := fs.Sub(ui, "ui/dist")
	if err != nil {
		fatalerr("access ui assets", "err", err)
	}
	mux.Handle("/", http.FileServerFS(buildFs))

	sources := make([]source, len(cfg))
	for i, src := range cfg {
		c.getCalendar(src.Server)
		sources[i] = source{}
	}

	handle, handler := v1connect.NewCalendarServiceHandler(
		CalendarService{},
		connect.WithInterceptors(
			connect.UnaryInterceptorFunc(tel.LogErrorsInterceptor),
		),
	)
	mux.Handle(handle, withCORS(handler))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
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

func main() {
	cli := Cli{
		Config: "config.json5",
		Serve: cliServe{
			Port: 3000,
		},
	}

	filler := myflags.NewFiller("schedule-utils", "view schedule statistics with ease")
	err := filler.Fill(&cli)
	if err != nil {
		fatalerr("parse cli args", "err", err)
	}
	err = filler.Execute()
	if err != nil {
		fatalerr("exec command", "err", err)
	}
}
