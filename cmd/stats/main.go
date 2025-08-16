package main

import (
	"calutils/api/v1/v1connect"
	"calutils/internal/calendar"
	"calutils/internal/config"
	"calutils/internal/tel"
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"time"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"github.com/rs/cors"
	"github.com/titanous/json5"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

//go:embed ui/dist/*
var ui embed.FS

type Config struct {
	Port    int             `json:"port"`    // The port to host the UI and API on.
	Sources []config.Source `json:"sources"` // Define calendar sources.
}

const description = `Visualize how your time is spent.`

func init() {
	flag.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"%s\n\nUsage: %s [options]\n\nOptions:\n",
			description,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
}

func main() {
	configpath := flag.String(
		"config",
		"config.json5",
		"Path to the config file. If you specify `:stdin:`, the program will read the config from STDIN.",
	)
	flag.Parse()

	cfg, err := parseConfig(*configpath)
	if err != nil {
		tel.Log.Error("main", fmt.Errorf("parse config: %w", err).Error())
		os.Exit(1)
	}

	err = run(cfg)
	if err != nil {
		tel.Log.Error("main", err.Error())
		os.Exit(1)
	}
}

func parseConfig(path string) (config Config, err error) {
	var contents []byte

	var file io.ReadCloser
	if path == ":stdin:" {
		file = os.Stdin
	} else {
		file, err = os.Open(path)
		if err != nil {
			return
		}
	}

	contents, err = io.ReadAll(file)
	if err != nil {
		return
	}
	err = json5.Unmarshal(contents, &config)
	if err != nil {
		return
	}
	return
}

func run(cfg Config) (err error) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// add ui route
	mux := http.NewServeMux()
	buildFs, err := fs.Sub(ui, "ui/dist")
	if err != nil {
		err = fmt.Errorf("access ui assets: %w", err)
		return
	}
	mux.Handle("/", http.FileServerFS(buildFs))

	// initialize sources
	sources := make([]sourceConfig, len(cfg.Sources))
	for i, src := range cfg.Sources {
		var source calendar.Source
		source, err = src.Server.Source()
		if err != nil {
			err = fmt.Errorf("create calendar: %w", err)
			return
		}
		sources[i] = sourceConfig{
			cfg:    src,
			Source: source,
		}
	}

	// setup rpc
	handle, handler := v1connect.NewCalendarServiceHandler(
		NewCalendarService(sources),
		connect.WithInterceptors(
			connect.UnaryInterceptorFunc(tel.LogErrorsInterceptor),
		),
	)
	withCors := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
	})
	mux.Handle(handle, withCors.Handler(handler))

	// run servers
	tel.Log.Info("main", "listening on...", "port", cfg.Port)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	go func() {
		<-ctx.Done()
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		server.Shutdown(timeout)
	}()
	err = server.ListenAndServe()
	if err != nil {
		err = fmt.Errorf("listen server: %w", err)
	}
	return
}
