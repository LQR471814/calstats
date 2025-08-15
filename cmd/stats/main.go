package main

import (
	"calutils/internal/tel"
	"net/http"
	"os"

	connectcors "connectrpc.com/cors"
	"github.com/hujun-open/myflags/v2"
	"github.com/rs/cors"
)

func main() {
	cli := Cli{
		Config: "config.json5",
		Serve: cliServe{
			Port: 3000,
		},
	}

	filler := myflags.NewFiller("calutils", "view schedule statistics with ease")
	err := filler.Fill(&cli)
	if err != nil {
		fatalerr("parse cli args", "err", err)
	}
	err = filler.Execute()
	if err != nil {
		fatalerr("exec command", "err", err)
	}
}

func fatalerr(msg string, args ...any) {
	tel.Log.Error("main", msg, args...)
	os.Exit(1)
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

