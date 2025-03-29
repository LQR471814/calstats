package calendar

import (
	"context"
	"time"
)

type Event struct {
	Name       string
	Tags       []string
	Start, End time.Time
	Duration   time.Duration
}

type Calendar struct {
	Id   string
	Name string
}

type Source interface {
	Calendars(ctx context.Context) ([]Calendar, error)
	Events(ctx context.Context, calendar Calendar, start, end time.Time) ([]Event, error)
}
