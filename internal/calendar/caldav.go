package calendar

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"schedule-statistics/internal/tel"
	"strconv"
	"strings"
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

func parseWeekday(weekday string) (rrule.Weekday, error) {
	switch weekday {
	case "MO":
		return rrule.MO, nil
	case "TU":
		return rrule.TU, nil
	case "WE":
		return rrule.WE, nil
	case "TH":
		return rrule.TH, nil
	case "FR":
		return rrule.FR, nil
	case "SA":
		return rrule.SA, nil
	case "SU":
		return rrule.SU, nil
	}
	return rrule.MO, fmt.Errorf("invalid weekday %s", weekday)
}

func wrapRRULEErr(err error) error {
	return fmt.Errorf("RRULE: %w", err)
}

func parseRecurrence(text string) (rrule.ROption, error) {
	var rules rrule.ROption

	for _, prop := range strings.Split(text, ";") {
		segments := strings.Split(prop, "=")
		if len(segments) < 2 {
			return rules, wrapRRULEErr(fmt.Errorf("invalid property '%s'", prop))
		}
		key := segments[0]
		value := segments[1]

		switch key {
		case "FREQ":
			switch value {
			case "YEARLY":
				rules.Freq = rrule.YEARLY
			case "MONTHLY":
				rules.Freq = rrule.MONTHLY
			case "WEEKLY":
				rules.Freq = rrule.WEEKLY
			case "DAILY":
				rules.Freq = rrule.DAILY
			case "HOURLY":
				rules.Freq = rrule.HOURLY
			case "MINUTELY":
				rules.Freq = rrule.MINUTELY
			case "SECONDLY":
				rules.Freq = rrule.SECONDLY
			default:
				return rules, wrapRRULEErr(fmt.Errorf("invalid FREQ value '%s'", value))
			}

		case "INTERVAL":
			val, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return rules, wrapRRULEErr(fmt.Errorf("invalid INTERNAL value '%s': %w", value, err))
			}
			rules.Interval = int(val)

		case "UNTIL":
			t, err := time.Parse("20060102T150405Z", value)
			if err != nil {
				return rules, wrapRRULEErr(fmt.Errorf("invalid UNTIL value '%s': %w", value, err))
			}
			rules.Until = t

		case "COUNT":
			count, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return rules, wrapRRULEErr(fmt.Errorf("invalid COUNT value '%s': %w", value, err))
			}
			rules.Count = int(count)

		case "BYDAY":
			segments := strings.Split(value, ",")

			for _, dayDef := range segments {
				offsetStr := ""
				day := ""

				for _, c := range dayDef {
					if (c >= '0' && c <= '9') || c == '-' {
						offsetStr += string(c)
						continue
					}
					if c >= 'A' && c <= 'Z' {
						day += string(c)
						continue
					}
				}

				var offset int64
				if offsetStr != "" {
					var err error
					offset, err = strconv.ParseInt(offsetStr, 10, 64)
					if err != nil {
						return rules, wrapRRULEErr(fmt.Errorf("invalid BYDAY value: %w", err))
					}
				}
				weekday, err := parseWeekday(day)
				if err != nil {
					return rules, wrapRRULEErr(fmt.Errorf("invalid BYDAY value: %w", err))
				}
				rules.Byweekday = append(rules.Byweekday, weekday.Nth(int(offset)))
			}

		case "BYMONTHDAY":
			values := strings.Split(value, ",")
			for _, v := range values {
				parsed, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return rules, wrapRRULEErr(fmt.Errorf("invalid BYMONTHDAY value '%s': %w", value, err))
				}
				rules.Bymonthday = append(rules.Bymonthday, int(parsed))
			}

		case "BYMONTH":
			values := strings.Split(value, ",")
			for _, v := range values {
				parsed, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return rules, wrapRRULEErr(fmt.Errorf(
						"invalid BYMONTH value '%s' (whole: '%s'): %w",
						v, value, err,
					))
				}
				rules.Bymonth = append(rules.Bymonth, int(parsed))
			}

		case "BYYEARDAY":
			values := strings.Split(value, ",")
			for _, v := range values {
				parsed, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return rules, wrapRRULEErr(fmt.Errorf(
						"invalid BYYEARDAY value '%s' (whole: '%s'): %w",
						v, value, err,
					))
				}
				rules.Byyearday = append(rules.Byyearday, int(parsed))
			}

		case "BYWEEKNO":
			values := strings.Split(value, ",")
			for _, v := range values {
				parsed, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return rules, wrapRRULEErr(fmt.Errorf(
						"invalid BYWEEKNO value '%s' (whole: %s): %w",
						v, value, err,
					))
				}
				rules.Byweekno = append(rules.Byweekno, int(parsed))
			}

		case "BYHOUR":
			values := strings.Split(value, ",")
			for _, v := range values {
				parsed, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return rules, wrapRRULEErr(fmt.Errorf(
						"invalid BYHOUR value '%s' (whole: %s): %w",
						v, value, err,
					))
				}
				rules.Byhour = append(rules.Byhour, int(parsed))
			}

		case "BYMINUTE":
			values := strings.Split(value, ",")
			for _, v := range values {
				parsed, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return rules, wrapRRULEErr(fmt.Errorf(
						"invalid BYMINUTE value '%s' (whole: %s): %w",
						v, value, err,
					))
				}
				rules.Byminute = append(rules.Byminute, int(parsed))
			}

		case "BYSECOND":
			values := strings.Split(value, ",")
			for _, v := range values {
				parsed, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return rules, wrapRRULEErr(fmt.Errorf(
						"invalid BYSECOND value '%s' (whole: %s): %w",
						v, value, err,
					))
				}
				rules.Bysecond = append(rules.Bysecond, int(parsed))
			}

		case "WKST":
			var err error
			rules.Wkst, err = parseWeekday(value)
			if err != nil {
				return rules, wrapRRULEErr(fmt.Errorf(
					"invalid WKST value '%s': %w",
					value, err,
				))
			}
		}
	}

	return rules, nil
}

func wrapEventsErr(err error) error {
	return fmt.Errorf("get events: %w", err)
}

type caldavEvent struct {
	Uid        string
	Name       string
	Categories []string
	ExDates    []time.Time
	Start, End time.Time
	Duration   time.Duration
	RRule      rrule.ROption
	RDates     string
	RId        string
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

			categories := e.Props.Get(ical.PropCategories)
			var tags []string
			if categories != nil {
				tags, err = categories.TextList()
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

			var recurId string
			if e.Props.Get(ical.PropRecurrenceID) != nil {
				recurId = e.Props.Get(ical.PropRecurrenceID).Value
			}
			var rdates string
			if e.Props.Get(ical.PropRecurrenceDates) != nil {
				rdates = e.Props.Get(ical.PropRecurrenceDates).Value
			}

			var ropts rrule.ROption
			recurrence := e.Props.Get(ical.PropRecurrenceRule)
			if recurrence != nil {
				ropts, err = parseRecurrence(recurrence.Value)
				if err != nil {
					tel.Log.Warn("caldav", "parse recurrence rule", "err", err)
					continue
				}
				if ropts.Until == (time.Time{}) || ropts.Until.After(intvEnd) {
					ropts.Until = intvEnd
				}
			}

			events = append(events, caldavEvent{
				Uid:        uid,
				Name:       name,
				Categories: tags,

				Start:    start,
				End:      end,
				Duration: duration,

				ExDates: exlist,
				RId:     recurId,
				RRule:   ropts,
				RDates:  rdates,
			})
		}
	}

	return events
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
		return nil, wrapEventsErr(err)
	}

	events := c.parseEvents(res, intvEnd, tz)

	intvStart = intvStart.In(tz)
	intvEnd = intvEnd.In(tz)

	var out []Event
	for _, eobj := range events {
		for _, e := range eobj.Data.Events() {

			recurrence := e.Props.Get(ical.PropRecurrenceRule)

			tel.Log.Debug(
				"caldav", "event",
				"name", name,
				"start", start,
				"end", end,
				"id", recurId,
				"dates", dates,
				"rule", recurrence,
				"uid", e.Props.Get(ical.PropUID).Value,
			)

			if recurrence != nil {
			recur:
				for _, recurTime := range rule.All() {
					if recurTime.After(intvStart) {
						break
					}
					for _, e := range exlist {
						if e.Equal(recurTime) {
							tel.Log.Debug("caldav", "skipped exception", "event", name)
							continue recur
						}
					}

					start := recurTime
					end := recurTime.Add(duration)

					if end.Before(intvStart) || start.After(intvEnd) {
						continue
					}
					if start.Before(intvStart) {
						start = intvStart
					}
					if end.After(intvEnd) {
						end = intvEnd
					}

					tel.Log.Debug("caldav", "recurring event", "name", name, "start", start, "end", end)

					out = append(out, Event{
						Name:  name,
						Tags:  tags,
						Start: start,
						End:   end,
					})
				}
				continue
			}

			if end.Before(intvStart) || start.After(intvEnd) {
				continue
			}
			if start.Before(intvStart) {
				start = intvStart
			}
			if end.After(intvEnd) {
				end = intvEnd
			}

			tel.Log.Debug("caldav", "single event", "name", name, "start", start, "end", end)

			// single event
			ev := Event{
				Name:  name,
				Tags:  tags,
				Start: start,
				End:   end,
			}
			out = append(out, ev)
		}
	}

	return out, nil
}
