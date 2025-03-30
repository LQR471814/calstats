package main

import (
	v1 "calendar-summary/api/v1"
	"calendar-summary/internal/calendar"
	"context"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CalendarService struct {
	source       calendar.Caldav
	calendarName string
}

func (s CalendarService) Events(ctx context.Context, req *connect.Request[v1.EventsRequest]) (*connect.Response[v1.EventsResponse], error) {
	calList, err := s.source.Calendars(ctx)
	if err != nil {
		return nil, err
	}
	var cal calendar.Calendar
	for _, c := range calList {
		if c.Name == s.calendarName {
			cal = c
			break
		}
	}
	if cal.Id == "" {
		return nil, fmt.Errorf("find calendar: not found '%s'", s.calendarName)
	}

	eventList, err := s.source.Events(ctx, cal, time.Time{}, time.Now().Add(365*24*time.Hour))
	if err != nil {
		return nil, err
	}
	slices.SortFunc(eventList, func(a, b calendar.Event) int {
		return a.Start.Compare(b.Start)
	})

	curTagIdx := uint32(0)
	tagIdxTable := map[string]uint32{}
	pbEvents := make([]*v1.Event, len(eventList))
	for eventIdx, event := range eventList {
		var tags []uint32
		if len(event.Tags) > 0 {
			tags = make([]uint32, len(event.Tags))
			for i, tagName := range event.Tags {
				outTagIdx, ok := tagIdxTable[tagName]
				if !ok {
					tagIdxTable[tagName] = curTagIdx
					outTagIdx = curTagIdx
					curTagIdx++
				}
				tags[i] = outTagIdx
			}
		}
		pbEvents[eventIdx] = &v1.Event{
			Name:     event.Name,
			Tags:     tags,
			Start:    timestamppb.New(event.Start),
			End:      timestamppb.New(event.End),
			Duration: uint32(event.Duration.Minutes()),
		}
	}

	tagLookup := make([]string, len(tagIdxTable))
	for k, idx := range tagIdxTable {
		tagLookup[int(idx)] = k
	}

	return connect.NewResponse(&v1.EventsResponse{
		Tags:   tagLookup,
		Events: pbEvents,
	}), nil
}
