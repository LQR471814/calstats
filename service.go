package main

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "schedule-statistics/api/v1"
	"schedule-statistics/internal/calendar"
	"slices"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CalendarService struct {
	source         calendar.Caldav
	calendarServer string
	calendars      []string
}

func prettyPrint(value any) string {
	expected, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(expected)
}

func DeoverlapEvents(eventList *[]calendar.Event) {
	// tel.Log.Debug("deoverlap", "=======")

	if len(*eventList) < 2 {
		return
	}

	type event struct {
		id int
		calendar.Event
	}

	events := make([]event, len(*eventList))
	for i, e := range *eventList {
		events[i] = event{
			id:    i,
			Event: e,
		}
	}

	for i := 1; i < len(events); i++ {
		a := events[i-1]
		b := events[i]

		// for _, e := range events {
		// 	fmt.Println(e.Name, e.Start.Format(time.Kitchen), e.End.Format(time.Kitchen))
		// }

		// Case where B is inside A
		if a.Start.Before(b.Start) && b.End.Before(a.End) {
			// tel.Log.Debug("deoverlap", "case: B inside A")

			events = slices.Insert(events, i+1, event{
				id: i - 1,
				Event: calendar.NewEvent(
					a.Name,
					b.End,
					a.End,
					a.Tags,
				),
			})
			a.End = b.Start
			events[i-1] = a
			continue
		}

		// Case where B is at the start of A but less long than A
		if a.Start.Equal(b.Start) && b.End.Before(a.End) {
			// tel.Log.Debug("deoverlap", "case: B.start = A.start but B.end < A.end")

			a.Start = b.End
			// swap positions since A starts later than B now
			events[i-1] = b
			events[i] = a

			continue
		}

		// Cases where B's end is after or equal to A's end
		if b.End.After(a.End) || b.End.Equal(a.End) {
			// Case where B starts in A but does not end in A
			if b.Start.After(a.Start) && b.Start.Before(a.End) {
				// tel.Log.Debug("deoverlap", "case: B starts in A but does not end in A")

				a.End = b.Start
				events[i-1] = a
				continue
			}

			// Case where B ends in A but does not start in A
			if a.Start.After(b.Start) && a.Start.Before(b.End) {
				// tel.Log.Debug("deoverlap", "case: B ends in A but does not start in A")

				b.End = a.Start
				events[i] = b
				continue
			}
		}

		// Case where A and B are touching each other and are the same event
		if a.id == b.id && a.End.Equal(b.Start) {
			a.End = b.End
			events[i-1] = a
			events = slices.Delete(events, i, i+1)
		}
	}

	// for _, e := range events {
	// 	fmt.Println(e.Name, e.Start.Format(time.Kitchen), e.End.Format(time.Kitchen))
	// }

	out := make([]calendar.Event, len(events))
	for i, e := range events {
		out[i] = e.Event
	}
	*eventList = out
}

func (s CalendarService) Events(ctx context.Context, req *connect.Request[v1.EventsRequest]) (*connect.Response[v1.EventsResponse], error) {
	calList, err := s.source.Calendars(ctx)
	if err != nil {
		return nil, err
	}
	var cals []calendar.Calendar
	for _, c := range calList {
		if slices.Contains(s.calendars, c.Name) {
			cals = append(cals, c)
		}
	}
	if len(cals) == 0 {
		return nil, fmt.Errorf("find calendar: not found '%s'", s.calendars)
	}

	var eventList []calendar.Event

	tzId := req.Msg.Timezone
	tz, err := time.LoadLocation(tzId)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}

	for _, c := range cals {
		events, err := s.source.Events(
			ctx, c,
			req.Msg.Interval.Start.AsTime(),
			req.Msg.Interval.End.AsTime(),
			tz,
		)
		if err != nil {
			return nil, err
		}
		eventList = append(eventList, events...)
	}

	slices.SortFunc(eventList, func(a, b calendar.Event) int {
		diff := a.Start.Compare(b.Start)
		if diff != 0 {
			return diff
		}
		// longer events go first in the event that multiple events have the same start time
		return -int(a.Duration() - b.Duration())
	})

	curTagIdx := uint32(0)
	tagIdxTable := map[string]uint32{}
	curNameIdx := uint32(0)
	nameIdxTable := map[string]uint32{}
	pbEvents := make([]*v1.Event, len(eventList))
	for eventIdx, event := range eventList {
		var tags []uint32
		if len(event.Tags) > 0 {
			tags = make([]uint32, len(event.Tags))
			for i, tagName := range event.Tags {
				tagIdx, ok := tagIdxTable[tagName]
				if !ok {
					tagIdxTable[tagName] = curTagIdx
					tagIdx = curTagIdx
					curTagIdx++
				}
				tags[i] = tagIdx
			}
		}
		nameIdx, ok := nameIdxTable[event.Name]
		if !ok {
			nameIdxTable[event.Name] = curNameIdx
			nameIdx = curNameIdx
			curNameIdx++
		}
		pbEvents[eventIdx] = &v1.Event{
			Name: nameIdx,
			Tags: tags,
			Interval: &v1.Interval{
				Start: timestamppb.New(event.Start),
				End:   timestamppb.New(event.End),
			},
			Duration: durationpb.New(event.Duration()),
		}
	}

	nameLookup := make([]string, len(nameIdxTable))
	for n, idx := range nameIdxTable {
		nameLookup[int(idx)] = n
	}
	tagLookup := make([]string, len(tagIdxTable))
	for k, idx := range tagIdxTable {
		tagLookup[int(idx)] = k
	}

	return connect.NewResponse(&v1.EventsResponse{
		EventNames: nameLookup,
		Tags:       tagLookup,
		Events:     pbEvents,
	}), nil
}

func (s CalendarService) Calendar(ctx context.Context, req *connect.Request[v1.CalendarRequest]) (*connect.Response[v1.CalendarResponse], error) {
	return connect.NewResponse(&v1.CalendarResponse{
		CalendarServer: s.calendarServer,
		Names:          s.calendars,
	}), nil
}
