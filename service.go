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

var max_time = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)

// // checkForOverlap returns a interval representing the non-overlapped time
// func checkForOverlap(events []calendar.Event, out *[]calendar.Event, prevIdx, curIdx int) (start, end time.Time) {
// 	tel.Log.Debug("layer", "checkForOverlap", "prev", prevIdx, "cur", curIdx)
//
// 	// exit when curIdx reaches the end
// 	if curIdx == len(events) {
// 		return events[prevIdx].End, max_time
// 	}
//
// 	prev := events[prevIdx]
// 	cur := events[curIdx]
//
// 	// if the current event is not overlapping with the previous event, exit
// 	if cur.Start.After(prev.End) || cur.Start.Equal(prev.End) {
// 		tel.Log.Debug("layer", "not overlapping", "prev", prevIdx, "cur", curIdx)
// 		return max_time
// 	}
//
// 	// if current event start is after previous event start
// 	// add the sliver of the previous event not covered by
// 	// the current event to the output events
// 	if !prev.Start.Equal(cur.Start) {
// 		*out = append(*out, calendar.Event{
// 			Name:     prev.Name,
// 			Tags:     prev.Tags,
// 			Start:    prev.Start,
// 			End:      cur.Start,
// 			Duration: cur.Start.Sub(prev.Start),
// 		})
// 	}
//
// 	maxEnd := checkForOverlap(events, out, curIdx, curIdx+1)
//
// 	tel.Log.Debug("layer", "max end", "max_end", maxEnd)
//
// 	// this means the next event doesn't overlap with the current event
// 	if maxEnd.Equal(max_time) {
// 		// the current event can be added wholesale to the output since
// 		// all "larger" events will be behind it
// 		*out = append(*out, cur)
// 	} else if maxEnd.Before(cur.End) {
// 		// this means that the next event's max end time
// 		// (the maximum time that events that can layer over the current event)
// 		// is less than the current event's end time
// 		//
// 		// this means that there's a sliver of the current event's time that should
// 		// be added after all the events overlapping the current event
// 		*out = append(*out, calendar.Event{
// 			Name:     cur.Name,
// 			Tags:     cur.Tags,
// 			Start:    maxEnd,
// 			End:      cur.End,
// 			Duration: cur.End.Sub(maxEnd),
// 		})
// 	}
//
// 	if cur.End.After(maxEnd) {
// 		return cur.End
// 	}
// 	return maxEnd
// }

// // LayerEvents takes overlapping events and flattens them (with lesser duration events
// // on the top and greater duration events on the bottom)
// func LayerEvents(events []calendar.Event) []calendar.Event {
// 	var output []calendar.Event
// 	for i := 1; i < len(events); i++ {
// 		checkForOverlap(events, &output, i-1, i)
// 	}
// 	return output
// }

type eventPoint struct {
	// true for isStart, false for end
	isStart bool
	point   time.Time
	ref     *calendar.Event
}

func pinEventPoints(events []calendar.Event) []eventPoint {
	// each event will have 2 event points so the capacity should be 2*len(events)
	points := make([]eventPoint, 0, len(events)*2)
	for i, e := range events {
		points = append(points, eventPoint{
			isStart: true,
			point:   e.Start,
			ref:     &events[i], // reference the element in the array, do not allocate new memory
		})
		points = append(points, eventPoint{
			isStart: false,
			point:   e.End,
			ref:     &events[i],
		})
	}
	// sort points ascending according to time, this creates a list of time points
	// that shows how events go in and out chronologically
	slices.SortFunc(points, func(a, b eventPoint) int {
		return a.point.Compare(b.point)
	})
	return points
}

func LayerEvents(events []calendar.Event) []calendar.Event {
	if len(events) == 0 {
		return nil
	}

	var output []calendar.Event

	points := pinEventPoints(events)
	cur := points[0]
	for i := 1; i < len(events); i++ {
		next := points[i]

		// if ending of current
		if !next.isStart && next.ref == cur.ref {
			output = append(output, *cur.ref)
		}

		// if start of different event (while current hasn't ended)
		// because of the way the events are sorted, we can be assured
		//
		// that this different event is on a higher "layer" than our current
		// event. that is, events with a greater start time are always on higher
		// layers and events with a
		if next.isStart && next.ref != cur.ref {
			output = append(output, calendar.NewEvent(
				cur.ref.Name,
				cur.ref.Start,
				next.ref.Start,
				cur.ref.Tags,
			))
		}

		cur = next
	}

	return output
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
		diff := a.Start.Compare(b.Start)
		if diff != 0 {
			return diff
		}
		// longer events go first in the event that multiple events have the same start time
		return -int(a.Duration - b.Duration)
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
