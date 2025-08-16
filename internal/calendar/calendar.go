package calendar

import (
	"context"
	"time"
)

type EventTrigger struct {
	Relative time.Duration
	Absolute time.Time
	NotNone  bool
}

type Event struct {
	Id          uint64
	Name        string
	Location    string
	Description string
	Tags        []string
	Start, End  time.Time
	Trigger     EventTrigger
}

func (e Event) Duration() time.Duration {
	return e.End.Sub(e.Start)
}

type Calendar struct {
	Id   string
	Name string
}

type UpdateEvent struct {
	Id          uint64
	Name        *string
	Location    *string
	Description *string
	Tags        *[]string
	Start, End  *time.Time
	Trigger     *EventTrigger
}

type Source interface {
	Calendars(ctx context.Context) ([]Calendar, error)
	Events(ctx context.Context, calendar Calendar, start, end time.Time, tz *time.Location) ([]Event, error)
	Update(ctx context.Context, events []UpdateEvent) error
}
