package main

import (
	v1 "calutil/api/v1"
	"calutil/internal/calendar"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CalendarService struct {
	mutex       sync.Mutex
	eventLookup []eventRef
	sources     []source
}

type eventRef struct {
	cal *calendar.Calendar
	uid string
}

type source struct {
	calendar.Source
	serverUrl string
	calendars []string
}

func NewCalendarService(sources []source) *CalendarService {
	return &CalendarService{
		sources: sources,
	}
}

func (s *CalendarService) Events(ctx context.Context, req *connect.Request[v1.EventsRequest]) (*connect.Response[v1.EventsResponse], error) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	tagIdxTable := map[string]uint32{}
	nameIdxTable := map[string]uint32{}
	curTagIdx := uint32(0)
	curNameIdx := uint32(0)

	var pbEvents []*v1.Event

	for _, source := range s.sources {
		cals, err := source.Calendars(ctx)
		if err != nil {
			return nil, err
		}
		var filtered []calendar.Calendar
		for _, c := range cals {
			if slices.Contains(source.calendars, c.Name) {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("find calendar: not found '%s'", source.calendars)
		}

		tzId := req.Msg.Timezone
		tz, err := time.LoadLocation(tzId)
		if err != nil {
			return nil, fmt.Errorf("load timezone: %w", err)
		}

		for _, cal := range filtered {
			eventList, err := source.Events(
				ctx, cal,
				req.Msg.Interval.Start.AsTime(),
				req.Msg.Interval.End.AsTime(),
				tz,
			)
			if err != nil {
				return nil, err
			}

			for _, event := range eventList {
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

				id := len(s.eventLookup)
				s.eventLookup = append(s.eventLookup, eventRef{
					cal: &cal,
					uid: event.Uid,
				})

				nameIdx, ok := nameIdxTable[event.Name]
				if !ok {
					nameIdxTable[event.Name] = curNameIdx
					nameIdx = curNameIdx
					curNameIdx++
				}

				eventOutput := &v1.Event{
					Id:          uint32(id),
					Name:        nameIdx,
					Location:    event.Location,
					Description: event.Description,
					Tags:        tags,
					Interval: &v1.Interval{
						Start: timestamppb.New(event.Start),
						End:   timestamppb.New(event.End),
					},
					Duration: durationpb.New(event.Duration()),
				}
				if event.Trigger.Absolute != (time.Time{}) {
					eventOutput.Trigger = &v1.Event_Absolute{
						Absolute: timestamppb.New(event.Trigger.Absolute),
					}
				} else if event.Trigger.NotNone {
					eventOutput.Trigger = &v1.Event_Relative{
						Relative: durationpb.New(event.Trigger.Relative),
					}
				}

				pbEvents = append(pbEvents, eventOutput)
			}
		}

	}

	slices.SortFunc(pbEvents, func(a, b *v1.Event) int {
		diff := a.Interval.Start.AsTime().Compare(b.Interval.Start.AsTime())
		if diff != 0 {
			return diff
		}
		// longer events go first in the event that multiple events have the same start time
		return -int(a.Duration.AsDuration() - b.Duration.AsDuration())
	})

	nameLookup := make([]string, len(nameIdxTable))
	for name, i := range nameIdxTable {
		nameLookup[int(i)] = name
	}
	tagLookup := make([]string, len(tagIdxTable))
	for tag, i := range tagIdxTable {
		tagLookup[int(i)] = tag
	}

	return connect.NewResponse(&v1.EventsResponse{
		EventNames: nameLookup,
		Tags:       tagLookup,
		Events:     pbEvents,
	}), nil
}

func (s *CalendarService) UpdateEvents(ctx context.Context, req *connect.Request[v1.UpdateEventsRequest]) (*connect.Response[v1.UpdateEventsResponse], error) {

}

func (s *CalendarService) Calendar(ctx context.Context, req *connect.Request[v1.CalendarRequest]) (*connect.Response[v1.CalendarResponse], error) {
	sources := make([]*v1.CalendarResponse_Source, len(s.sources))
	for i, s := range s.sources {
		sources[i] = &v1.CalendarResponse_Source{
			CalendarServer: s.serverUrl,
			Names:          s.calendars,
		}
	}
	return connect.NewResponse(&v1.CalendarResponse{
		Sources: sources,
	}), nil
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
				Event: calendar.Event{
					Uid:   a.Uid,
					Name:  a.Name,
					Start: b.End,
					End:   a.End,
					Tags:  a.Tags,
				},
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

func prettyPrint(value any) string {
	expected, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(expected)
}
