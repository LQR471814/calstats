package calendar

import (
	"calendar-summary/internal/tel"
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

func (c Caldav) applyRecurrence(end time.Time, ev Event, recurrence string) {
	type byDayRules struct {
		Offset   int
		Weekdays []time.Weekday
	}

	type recurRules struct {
		FREQ     string
		INTERVAL int
		UNTIL    time.Time
		COUNT    int

		BYDAY      byDayRules
		BYMONTHDAY []int
		BYMONTH    []time.Month
		BYYEARDAY  []int
		BYWEEKNO   []int
		BYHOUR     []int
		BYMINUTE   []int

		WKST time.Weekday
	}

	var rules recurRules

	for _, pair := range strings.Split(recurrence, ";") {
		segments := strings.Split(pair, "=")
		if len(segments) < 2 {
			tel.Log.Warn("caldav", "invalid RRULE pair", "pair", pair)
			continue
		}
		key := segments[0]
		value := segments[1]
		switch key {
		case "FREQ":
			rules.FREQ = value
		case "INTERVAL":
			val, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				tel.Log.Error("caldav", "invalid INTERVAL value", "value", value, "err", err)
				continue
			}
			rules.INTERVAL = int(val)
		case "UNTIL":
			t, err := time.Parse("20060102T150405Z", value)
			if err != nil {
				tel.Log.Error("caldav", "invalid UNTIL value", "value", value, "err", err)
				continue
			}
			rules.UNTIL = t
		case "COUNT":
			count, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				tel.Log.Error("caldav", "invalid COUNT value", "value", value, "err", err)
				continue
			}
			rules.COUNT = int(count)
		case "BYDAY":
			negative := false
			offset := ""
			var weekdays []string
			for i, c := range value {
				if c == '-' && i == 0 {
					negative = true
					continue
				}
				if c >= '0' && c <= '9' {
					offset += string(c)
					continue
				}
				if c == ',' {
					weekdays = append(weekdays, "")
					continue
				}
				if c >= 'A' && c <= 'Z' {
					weekdays[len(weekdays)-1] += string(c)
					continue
				}
			}
			if offset != "" {
				parsed, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					tel.Log.Error("caldav", "invalid BYDAY value", "value", value, "err", err)
					continue
				}
				rules.BYDAY.Offset = int(parsed)
				if negative {
					rules.BYDAY.Offset *= -1
				}
			}
			for _, w := range weekdays {
				switch w {
				case "MO":
					rules.BYDAY.Weekdays = append(rules.BYDAY.Weekdays, time.Monday)
				case "TU":
					rules.BYDAY.Weekdays = append(rules.BYDAY.Weekdays, time.Tuesday)
				case "WE":
					rules.BYDAY.Weekdays = append(rules.BYDAY.Weekdays, time.Wednesday)
				case "TH":
					rules.BYDAY.Weekdays = append(rules.BYDAY.Weekdays, time.Thursday)
				case "FR":
					rules.BYDAY.Weekdays = append(rules.BYDAY.Weekdays, time.Friday)
				case "SA":
					rules.BYDAY.Weekdays = append(rules.BYDAY.Weekdays, time.Saturday)
				case "SU":
					rules.BYDAY.Weekdays = append(rules.BYDAY.Weekdays, time.Sunday)
				}
			}
		}
	}

}

func (c Caldav) Events(ctx context.Context, calendar Calendar, start, end time.Time) ([]Event, error) {
	query := &caldav.CalendarQuery{
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{{
				Name:  "VEVENT",
				Start: start,
				End:   end,
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
		return nil, fmt.Errorf("get cal events: %w", err)
	}

	var out []Event
	for _, eobj := range events {
		for _, e := range eobj.Data.Events() {
			start, err := e.DateTimeStart(time.Local)
			if err != nil {
				return nil, err
			}
			end, err := e.DateTimeEnd(time.Local)
			if err != nil {
				return nil, err
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
					return nil, err
				}
			}

			ev := Event{
				Name:  name,
				Tags:  tags,
				Start: start,
				End:   end,
			}
			out = append(out, ev)

			recurrence := e.Props.Get(ical.PropRecurrenceRule)
			if recurrence != nil {
			}
		}
	}

	return out, nil
}
