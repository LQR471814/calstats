package calendar

import (
	"calutil/internal/tel"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/teambition/rrule-go"
	"github.com/zeebo/xxh3"
	"golang.org/x/time/rate"
)

type eventId struct {
	Uid string
	RId string

	// ShouldOverride determines whether an override should be created for this
	// event. It will be true if the given event is a recurrence instance and
	// it does not already have an override.
	ShouldOverride bool
}

type Caldav struct {
	client *caldav.Client
	ids    map[uint64]eventId
}

type CaldavOptions struct {
	Username string
	Password string
	Insecure bool
}

func NewCaldav(server string, opts CaldavOptions) (client Caldav, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("new caldav client: %w", err)
			return
		}
	}()

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

	inner, err := caldav.NewClient(webdavHttp, server)
	if err != nil {
		return
	}
	return Caldav{
		client: inner,
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
					ical.PropDescription,
					ical.PropLocation,
					ical.PropDateTimeStart,
					ical.PropDateTimeEnd,
					ical.PropCategories,
					ical.PropRecurrenceDates,
					ical.PropRecurrenceID,
					ical.PropRecurrenceRule,
					ical.PropTrigger,
				},
			}},
		},
	})
	if err != nil {
		return nil, err
	}

	var events []caldavEvent
	for _, eobj := range res {
		for _, e := range eobj.Data.Events() {
			parsed, err := parseEvent(e, intvEnd, tz)
			if err != nil {
				tel.Log.Warn("caldav", "skip corrupted event", "err", err)
				continue
			}
			events = append(events, parsed)
		}
	}

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
				Id:      intId(e.Uid, formatICalDatetime(e.RId)),
				Name:    e.Name,
				Tags:    e.Categories,
				Start:   e.Start,
				End:     e.End,
				Trigger: EventTrigger(e.Trigger),
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
				if !recurTime.Equal(ov.RId) {
					continue
				}
				outev, ok := c.adjustEventBounds(Event{
					Id:      intId(ov.Uid, formatICalDatetime(ov.RId)),
					Name:    ov.Name,
					Tags:    ov.Categories,
					Start:   ov.Start,
					End:     ov.End,
					Trigger: EventTrigger(ov.Trigger),
				}, intvStart, intvEnd)
				if ok {
					out = append(out, outev)
				}
				continue recur
			}

			outev, ok := c.adjustEventBounds(Event{
				Id:      intId(re.original.Uid, formatICalDatetime(re.original.RId)),
				Name:    re.original.Name,
				Tags:    re.original.Categories,
				Start:   recurTime,
				End:     recurTime.Add(re.original.Duration),
				Trigger: EventTrigger(re.original.Trigger),
			}, intvStart, intvEnd)
			if ok {
				out = append(out, outev)
			}
		}
	}

	return out, nil
}

func (c Caldav) UpdateEvents(ctx context.Context, calendar Calendar, updates []UpdateEvent) (err error) {
	urls, err := batchDo(ctx, updates, func(u UpdateEvent) (href string, err error) {
		res, err := c.client.QueryCalendar(ctx, calendar.Id, &caldav.CalendarQuery{
			CompFilter: caldav.CompFilter{
				Name: ical.CompCalendar,
				Comps: []caldav.CompFilter{{
					Name: ical.CompEvent,
					Props: []caldav.PropFilter{{
						Name: ical.PropUID,
						TextMatch: &caldav.TextMatch{
							Text: u.Uid,
						},
					}},
				}},
			},
			CompRequest: caldav.CalendarCompRequest{
				Name: ical.CompCalendar,
				Comps: []caldav.CalendarCompRequest{{
					Name: ical.CompEvent,
					Props: []string{
						ical.PropURL,
					},
				}},
			},
		})
		if err != nil {
			return
		}
		if len(res) != 1 {
			err = fmt.Errorf("invalid number of returned events (%d)", len(res))
			return
		}
		events := res[0].Data.Events()
		if len(events) != 1 {
			err = fmt.Errorf("invalid number of returned events (%d)", len(res))
			return
		}
		href = events[0].Props.Get(ical.PropURL).Value
		return
	})
	if err != nil {
		return
	}

	res, err := c.client.MultiGetCalendar(ctx, calendar.Id, &caldav.CalendarMultiGet{
		Paths: urls,
	})
	if err != nil {
		return
	}

	for _, eobj := range res {
	}
}

func batchDo[I, O any](ctx context.Context, jobs []I, fn func(I) (O, error)) ([]O, error) {
	limiter := rate.NewLimiter(1, 4)
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}

	var errs []error
	outputs := make([]O, len(jobs))
	for i, input := range jobs {
		err := limiter.Wait(ctx)
		if err != nil {
			tel.Log.Warn("caldav", "rate limit error", "err", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			output, err := fn(input)

			defer lock.Unlock()
			lock.Lock()
			outputs[i] = output
			errs = append(errs, err)
		}()
	}

	return outputs, errors.Join(errs...)
}

type caldavEvent struct {
	Uid         string
	Name        string
	Location    string
	Description string
	Categories  []string
	ExDates     []time.Time
	Start, End  time.Time
	Duration    time.Duration
	RRule       *rrule.RRule
	RDates      string
	RId         time.Time
	Trigger     EventTrigger
}

// intId hashes a calendar event's UID and Recurrence ID into a single uint64.
func intId(uid string, rid string) uint64 {
	return xxh3.Hash([]byte(uid + rid))
}

// formatICalDatetime formats a given [time.Time] in the ical datetime format.
func formatICalDatetime(t time.Time) string {
	return t.Format("20060102T150405")
}

func parseEvent(e ical.Event, intvEnd time.Time, tz *time.Location) (event caldavEvent, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parse event: %w", err)
		}
	}()

	(&event).ParseUID(e)
	(&event).ParseName(e)
	(&event).ParseLocation(e)
	(&event).ParseDescription(e)

	err = (&event).ParseStart(e, tz)
	if err != nil {
		return
	}
	err = (&event).ParseEnd(e, tz)
	if err != nil {
		return
	}
	event.Duration = event.End.Sub(event.Start)

	err = (&event).ParseCategories(e)
	if err != nil {
		return
	}
	err = (&event).ParseExceptions(e)
	if err != nil {
		return
	}
	err = (&event).ParseRecurrence(e, tz, event.Start, intvEnd)
	if err != nil {
		return
	}
	err = (&event).ParseTrigger(e, tz)
	if err != nil {
		return
	}

	if event.Uid == "" {
		err = fmt.Errorf("uid is nil")
		return
	}
	if event.Name == "" {
		err = fmt.Errorf("name is nil")
		return
	}

	return
}

func (ce *caldavEvent) ParseUID(e ical.Event) {
	uidProp := e.Props.Get(ical.PropUID)
	if uidProp == nil {
		return
	}
	ce.Uid = uidProp.Value
}

func (ce *caldavEvent) ParseName(e ical.Event) {
	nameProp := e.Props.Get(ical.PropSummary)
	if nameProp == nil {
		return
	}
	ce.Name = nameProp.Value
}

func (ce *caldavEvent) ParseDescription(e ical.Event) {
	descProp := e.Props.Get(ical.PropDescription)
	if descProp == nil {
		return
	}
	ce.Description = descProp.Value
}

func (ce *caldavEvent) ParseLocation(e ical.Event) {
	locProp := e.Props.Get(ical.PropLocation)
	if locProp == nil {
		return
	}
	ce.Location = locProp.Value
}

func (ce *caldavEvent) ParseCategories(e ical.Event) error {
	catProp := e.Props.Get(ical.PropCategories)
	if catProp == nil {
		return nil
	}
	categories, err := catProp.TextList()
	if err != nil {
		return err
	}
	ce.Categories = categories
	return nil
}

func (ce *caldavEvent) ParseStart(e ical.Event, tz *time.Location) error {
	start, err := e.DateTimeStart(tz)
	if err != nil {
		return err
	}
	ce.Start = start
	return nil
}

func (ce *caldavEvent) ParseEnd(e ical.Event, tz *time.Location) error {
	end, err := e.DateTimeEnd(tz)
	if err != nil {
		return err
	}
	ce.End = end
	return nil
}

func (ce *caldavEvent) ParseExceptions(e ical.Event) error {
	exProp := e.Props.Get(ical.PropExceptionDates)
	if exProp == nil {
		return nil
	}

	tzId := exProp.Params.Get(ical.PropTimezoneID)
	var err error
	var tz *time.Location
	if tzId != "" {
		tz, err = time.LoadLocation(tzId)
		if err != nil {
			return err
		}
	}

	var datetime time.Time
	datetime, err = exProp.DateTime(tz)
	if err != nil {
		return err
	}

	ce.ExDates = append(ce.ExDates, datetime)
	return nil
}

func (ce *caldavEvent) ParseRecurrence(e ical.Event, tz *time.Location, start, intvEnd time.Time) (err error) {
	recurIdProp := e.Props.Get(ical.PropRecurrenceID)
	if recurIdProp != nil && recurIdProp.Value != "" {
		ce.RId, err = recurIdProp.DateTime(tz)
		if err != nil {
			return
		}
	}

	rdateProp := e.Props.Get(ical.PropRecurrenceDates)
	if rdateProp != nil {
		ce.RDates = rdateProp.Value
	}

	rruleProp := e.Props.Get(ical.PropRecurrenceRule)
	if rruleProp != nil {
		var ropts *rrule.ROption
		ropts, err = rrule.StrToROptionInLocation(rruleProp.Value, tz)
		if err != nil {
			return
		}
		if ropts == nil {
			err = fmt.Errorf("ropts is nil")
			return
		}

		if ropts.Until == (time.Time{}) || ropts.Until.After(intvEnd) {
			ropts.Until = intvEnd
		}
		// set default dtstart to original event's starting time
		if ropts.Dtstart == (time.Time{}) {
			ropts.Dtstart = start
		}

		ce.RRule, err = rrule.NewRRule(*ropts)
		if err != nil {
			return
		}
	}

	return
}

func (ce *caldavEvent) ParseTrigger(e ical.Event, tz *time.Location) (err error) {
	triggerProp := e.Props.Get(ical.PropTrigger)
	if triggerProp == nil {
		return
	}
	ce.Trigger.Relative, err = triggerProp.Duration()
	if err == nil {
		return
	}
	ce.Trigger.Absolute, err = triggerProp.DateTime(tz)
	if err != nil {
		return err
	}
	return
}
