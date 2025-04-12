package calendar

import (
	"context"
	"time"
)

type Event struct {
	Name       string
	Tags       []string
	Start, End time.Time
}

func (e Event) Duration() time.Duration {
	return e.End.Sub(e.Start)
}

func NewEvent(name string, start, end time.Time, tags []string) Event {
	return Event{
		Name:  name,
		Start: start,
		End:   end,
		Tags:  tags,
	}
}

type Calendar struct {
	Id   string
	Name string
}

type Source interface {
	Calendars(ctx context.Context) ([]Calendar, error)
	Events(ctx context.Context, calendar Calendar, start, end time.Time, tz *time.Location) ([]Event, error)
}
