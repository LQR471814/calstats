package config

import (
	"calstats/internal/calendar"
)

type Source struct {
	Server    Server   `json:"server"`    // Server configuration.
	Calendars []string `json:"calendars"` // Specify the calendars you want to include by their names.
}

type Server struct {
	Url      string `json:"url"`      // Specify the principal url of the caldav server, that is the caldav server + the user.
	Insecure bool   `json:"insecure"` // Ignore HTTPS issues.
	Username string `json:"username"` // Authentication username.
	Password string `json:"password"` // Authentication password.
}

func (cfg Server) Source() (source calendar.Source, err error) {
	source, err = calendar.NewCaldav(cfg.Url, calendar.CaldavOptions{
		Username: cfg.Username,
		Password: cfg.Password,
		Insecure: cfg.Insecure,
	})
	return
}