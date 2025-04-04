package calendar

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
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

func (c Caldav) Events(ctx context.Context, calendar Calendar, intvStart, intvEnd time.Time) ([]Event, error) {
	query := &caldav.CalendarQuery{
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{{
				Name:  "VEVENT",
				Start: intvStart,
				End:   intvEnd,
			}},
		},
		CompRequest: caldav.CalendarCompRequest{
			Name: "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{{
				Name: "VEVENT",
				Props: []string{
					"SUMMARY",
					"UID",
					"DTSTART",
					"DTEND",
					"DURATION",
				},
			}},
		},
	}

	events, err := c.client.QueryCalendar(ctx, calendar.Id, query)
	if err != nil {
		return nil, wrapEventsErr(err)
	}

	var out []Event
	for _, eobj := range events {
		for _, e := range eobj.Data.Events() {
			start, err := e.DateTimeStart(time.Local)
			if err != nil {
				return nil, wrapEventsErr(err)
			}
			end, err := e.DateTimeEnd(time.Local)
			if err != nil {
				return nil, wrapEventsErr(err)
			}

			name := ""
			nameProp := e.Props.Get(ical.PropSummary)
			if nameProp != nil {
				name = nameProp.Value
			}

			categories := e.Props.Get(ical.PropCategories)
			var tags []string
			if categories != nil {
				tags, err = categories.TextList()
				if err != nil {
					return nil, wrapEventsErr(err)
				}
			}

			duration := end.Sub(start)

			recurrence := e.Props.Get(ical.PropRecurrenceRule)
			if recurrence != nil {
				// recurring event
				opts, err := parseRecurrence(recurrence.Value)
				if err != nil {
					return nil, wrapEventsErr(err)
				}
				if opts.Until == (time.Time{}) || opts.Until.After(intvEnd) {
					opts.Until = intvEnd
				}
				rule, err := rrule.NewRRule(opts)
				if err != nil {
					return nil, wrapEventsErr(err)
				}
				for _, recurTime := range rule.All() {
					out = append(out, Event{
						Name:  name,
						Tags:  tags,
						Start: recurTime,
						End:   recurTime.Add(duration),
					})
				}
				continue
			}

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
