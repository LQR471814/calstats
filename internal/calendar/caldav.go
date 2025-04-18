package calendar

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"schedule-utils/internal/tel"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/teambition/rrule-go"
)

type Caldav struct {
	client *caldav.Client
}

type CaldavOptions struct {
	Username string
	Password string
	Insecure bool
}

func NewCaldav(server string, opts CaldavOptions) (Caldav, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.Insecure,
		},
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	webdavHttp := webdav.HTTPClient(httpClient)
	if opts.Username != "" && opts.Password != "" {
		webdavHttp = webdav.HTTPClientWithBasicAuth(httpClient, opts.Username, opts.Password)
	}

	client, err := caldav.NewClient(webdavHttp, server)
	if err != nil {
		return Caldav{}, fmt.Errorf("make caldav client: %w", err)
	}
	return Caldav{
		client: client,
	}, nil
}

func (c Caldav) Calendars(ctx context.Context) ([]Calendar, error) {
	homeSet, err := c.client.FindCalendarHomeSet(ctx, "")
	if err != nil {
		return nil, err
	}
	calendars, err := c.client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, err
	}
	out := make([]Calendar, len(calendars))
	for i, c := range calendars {
		out[i] = Calendar{
			Id:   c.Path,
			Name: c.Name,
		}
	}
	return out, nil
}

type caldavEvent struct {
	Uid        string
	Name       string
	Categories []string
	ExDates    []time.Time
	Start, End time.Time
	Duration   time.Duration
	RRule      *rrule.RRule
	RDates     string
	RId        time.Time
}

func (c Caldav) parseEvents(objs []caldav.CalendarObject, intvEnd time.Time, tz *time.Location) []caldavEvent {
	var err error
	var events []caldavEvent

	for _, eobj := range objs {
		for _, e := range eobj.Data.Events() {
			nameProp := e.Props.Get(ical.PropSummary)
			if nameProp == nil {
				tel.Log.Warn("caldav", "got event with no name, skipping...", "event", e)
				continue
			}
			name := nameProp.Value

			uidProp := e.Props.Get(ical.PropUID)
			if uidProp == nil {
				tel.Log.Warn("caldav", "got event with no UID, skipping...", "event", e)
				continue
			}
			uid := uidProp.Value

			catProp := e.Props.Get(ical.PropCategories)
			var categories []string
			if catProp != nil {
				categories, err = catProp.TextList()
				if err != nil {
					tel.Log.Warn("caldav", "parse event categories", "err", err)
					continue
				}
			}

			start, err := e.DateTimeStart(tz)
			if err != nil {
				tel.Log.Warn("caldav", "parse event start", "err", err)
				continue
			}
			end, err := e.DateTimeEnd(tz)
			if err != nil {
				tel.Log.Warn("caldav", "parse event end", "err", err)
				continue
			}
			duration := end.Sub(start)

			exProp := e.Props.Get(ical.PropExceptionDates)
			var exlist []time.Time
			if exProp != nil {
				tzId := exProp.Params.Get(ical.PropTimezoneID)
				var tz *time.Location
				if tzId != "" {
					tz, err = time.LoadLocation(tzId)
					if err != nil {
						tel.Log.Warn("caldav", "load timezone location", "err", err)
						continue
					}
				}

				datetime, err := exProp.DateTime(tz)
				if err != nil {
					tel.Log.Warn("caldav", "parse exception date", "err", err)
					continue
				}
				exlist = append(exlist, datetime)
			}

			var recurId time.Time
			recurIdProp := e.Props.Get(ical.PropRecurrenceID)
			if recurIdProp != nil && recurIdProp.Value != "" {
				recurId, err = recurIdProp.DateTime(tz)
				if err != nil {
					tel.Log.Warn("caldav", "parse recurrence id", "err", err)
					continue
				}
			}
			var rdates string
			rdateProp := e.Props.Get(ical.PropRecurrenceDates)
			if rdateProp != nil {
				rdates = rdateProp.Value
			}

			var rule *rrule.RRule
			rruleProp := e.Props.Get(ical.PropRecurrenceRule)
			if rruleProp != nil {
				ropts, err := rrule.StrToROptionInLocation(rruleProp.Value, tz)
				if err != nil {
					tel.Log.Warn("caldav", "parse recurrence rule", "err", err)
					continue
				}
				if ropts == nil {
					tel.Log.Warn("caldav", "parse recurrence rule", "err", fmt.Errorf("ropts is nil"))
					continue
				}

				if ropts.Until == (time.Time{}) || ropts.Until.After(intvEnd) {
					ropts.Until = intvEnd
				}
				// set default dtstart to original event's starting time
				if ropts.Dtstart == (time.Time{}) {
					ropts.Dtstart = start
				}

				rule, err = rrule.NewRRule(*ropts)
				if err != nil {
					tel.Log.Warn("caldav", "new rrule", "err", err)
					continue
				}
			}

			events = append(events, caldavEvent{
				Uid:        uid,
				Name:       name,
				Categories: categories,

				Start:    start,
				End:      end,
				Duration: duration,

				ExDates: exlist,
				RId:     recurId,
				RRule:   rule,
				RDates:  rdates,
			})
		}
	}

	return events
}

// adjustEventBounds crops the event so that it is within the interval bounds [intvStart, intvEnd].
// If the event is outside the interval completely, it will return false in the second return value.
func (c Caldav) adjustEventBounds(ev Event, intvStart, intvEnd time.Time) (Event, bool) {
	if ev.End.Before(intvStart) || ev.Start.After(intvEnd) {
		return ev, false
	}
	if ev.Start.Before(intvStart) {
		ev.Start = intvStart
	}
	if ev.End.After(intvEnd) {
		ev.End = intvEnd
	}
	return ev, true
}

func (c Caldav) Events(ctx context.Context, calendar Calendar, intvStart, intvEnd time.Time, tz *time.Location) ([]Event, error) {
	res, err := c.client.QueryCalendar(ctx, calendar.Id, &caldav.CalendarQuery{
		CompFilter: caldav.CompFilter{
			Name: ical.CompCalendar,
			Comps: []caldav.CompFilter{{
				Name:  ical.CompEvent,
				Start: intvStart.In(time.UTC),
				End:   intvEnd.In(time.UTC),
			}},
		},
		CompRequest: caldav.CalendarCompRequest{
			Name: ical.CompCalendar,
			Comps: []caldav.CalendarCompRequest{{
				Name: ical.CompEvent,
				Props: []string{
					ical.PropUID,
					ical.PropSummary,
					ical.PropDateTimeStart,
					ical.PropDateTimeEnd,
					ical.PropCategories,
					ical.PropRecurrenceDates,
					ical.PropRecurrenceID,
					ical.PropRecurrenceRule,
				},
			}},
		},
	})
	if err != nil {
		return nil, err
	}

	events := c.parseEvents(res, intvEnd, tz)

	intvStart = intvStart.In(tz)
	intvEnd = intvEnd.In(tz)

	var out []Event

	type recurringEvent struct {
		original  caldavEvent
		overrides []caldavEvent
	}
	recurring := map[string]recurringEvent{}
	for _, e := range events {
		track := recurring[e.Uid]
		if e.RRule != nil { // original recurring event
			track.original = e
		} else if e.RId != (time.Time{}) { // override instance of recurring event
			track.overrides = append(track.overrides, e)
		} else { // single event
			outev, ok := c.adjustEventBounds(Event{
				Name:  e.Name,
				Tags:  e.Categories,
				Start: e.Start,
				End:   e.End,
			}, intvStart, intvEnd)
			if ok {
				out = append(out, outev)
			}
			continue
		}
		recurring[e.Uid] = track
	}

	for _, re := range recurring {
		if re.original.Name == "" {
			tel.Log.Warn("caldav", "recurring event without original event present", "re", re)
			continue
		}

	recur:
		for _, recurTime := range re.original.RRule.All() {
			if recurTime.After(intvEnd) {
				break
			}
			for _, e := range re.original.ExDates {
				if e.Equal(recurTime) {
					continue recur
				}
			}

			for _, ov := range re.overrides {
				if recurTime.Equal(ov.RId) {
					outev, ok := c.adjustEventBounds(Event{
						Name:  ov.Name,
						Tags:  ov.Categories,
						Start: ov.Start,
						End:   ov.End,
					}, intvStart, intvEnd)
					if ok {
						out = append(out, outev)
					}
					continue recur
				}
			}

			outev, ok := c.adjustEventBounds(Event{
				Name:  re.original.Name,
				Tags:  re.original.Categories,
				Start: recurTime,
				End:   recurTime.Add(re.original.Duration),
			}, intvStart, intvEnd)
			if ok {
				out = append(out, outev)
			}
		}
	}

	return out, nil
}
