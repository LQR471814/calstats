package main

import (
	"calutils/internal/calendar"
	"calutils/internal/config"
	"calutils/internal/tel"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/titanous/json5"
)

const description = `Manage thunderbird categories with the filesystem.`

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
	err := run()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type Config struct {
	Source        config.Source `json:"source"`         // Calendar source to use.
	ProjectsDir   string        `json:"projects_dir"`   // Directory to read categories from.
	Ignore        []string      `json:"ignore"`         // Ignore certain directories with a glob pattern.
	UserJSOutputs []string      `json:"userjs_outputs"` // Paths to user.js file outputs
}

func run() (err error) {
	cfgfile := flag.String("config", "config.json5", "Path to configuration file. If `:stdin:` is specified, config will be read from STDIN.")
	flag.Parse()

	var file io.ReadCloser
	if *cfgfile == ":stdin:" {
		file = os.Stdin
	} else {
		file, err = os.Open(*cfgfile)
		if err != nil {
			return
		}
	}

	contents, err := io.ReadAll(file)
	if err != nil {
		return
	}
	var cfg Config
	err = json5.Unmarshal(contents, &cfg)
	if err != nil {
		return
	}

	source, err := cfg.Source.Server.Source()
	if err != nil {
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err = watch(ctx, cfg, source)
	return
}

func watch(ctx context.Context, cfg Config, source calendar.Source) (err error) {
	var watcher *fsnotify.Watcher
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	err = filepath.Walk(cfg.ProjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			err := watcher.Add(path)
			if err != nil {
				log.Println("Error adding directory:", err)
			} else {
				fmt.Println("Watching directory:", path)
			}
		}
		return nil
	})
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watcher.Events:
			tel.Log.Debug("main", "event", event.Name)

			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
			case event.Op&fsnotify.Rename == fsnotify.Rename:
			case event.Op&fsnotify.Remove == fsnotify.Remove:
			}
		case err := <-watcher.Errors:
			tel.Log.Error("main", "watch", "err", err)
		}
	}
}
