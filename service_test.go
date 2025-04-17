package main

import (
	"schedule-statistics/internal/calendar"
	"testing"
	"time"
)

func equalEvents(a, b calendar.Event) bool {
	return a.Name == b.Name &&
		a.Start.Equal(b.Start) &&
		a.End.Equal(b.End)
}

func TestLayerEvents(t *testing.T) {
	type testCase struct {
		input  []calendar.Event
		expect []calendar.Event
	}

	datetime := func(hour, minute int) time.Time {
		return time.Date(2000, time.January, 1, hour, minute, 0, 0, time.Local)
	}

	table := []testCase{
		{
			input: []calendar.Event{
				calendar.NewEvent("A", datetime(9, 30), datetime(10, 0), nil),
				calendar.NewEvent("B", datetime(9, 30), datetime(9, 45), nil),
				calendar.NewEvent("C", datetime(9, 50), datetime(10, 0), nil),
			},
			expect: []calendar.Event{
				calendar.NewEvent("B", datetime(9, 30), datetime(9, 45), nil),
				calendar.NewEvent("A", datetime(9, 45), datetime(9, 50), nil),
				calendar.NewEvent("C", datetime(9, 50), datetime(10, 0), nil),
			},
		},
		{
			input: []calendar.Event{
				calendar.NewEvent("A", datetime(9, 0), datetime(9, 15), nil),
				calendar.NewEvent("B", datetime(9, 15), datetime(9, 30), nil),
				calendar.NewEvent("C", datetime(9, 30), datetime(9, 45), nil),
			},
			expect: []calendar.Event{
				calendar.NewEvent("A", datetime(9, 0), datetime(9, 15), nil),
				calendar.NewEvent("B", datetime(9, 15), datetime(9, 30), nil),
				calendar.NewEvent("C", datetime(9, 30), datetime(9, 45), nil),
			},
		},
		{
			input: []calendar.Event{
				calendar.NewEvent("A", datetime(9, 0), datetime(9, 15), nil),
				calendar.NewEvent("B", datetime(9, 15), datetime(9, 45), nil),
				calendar.NewEvent("C", datetime(9, 30), datetime(9, 40), nil),
			},
			expect: []calendar.Event{
				calendar.NewEvent("A", datetime(9, 0), datetime(9, 15), nil),
				calendar.NewEvent("B", datetime(9, 15), datetime(9, 30), nil),
				calendar.NewEvent("C", datetime(9, 30), datetime(9, 40), nil),
				calendar.NewEvent("B", datetime(9, 40), datetime(9, 45), nil),
			},
		},
	}

	for _, test := range table {
		result := make([]calendar.Event, len(test.input))
		copy(result, test.input)
		DeoverlapEvents(&result)
		if len(test.expect) != len(result) {
			t.Fatalf(
				"event lists are not equal\n\nInput: %s\n\nExpected: %s\n\nResult: %s",
				prettyPrint(test.input),
				prettyPrint(test.expect),
				prettyPrint(result),
			)
		}
		for i := range test.expect {
			if !equalEvents(test.expect[i], result[i]) {
				t.Log(prettyPrint(result))
				t.Fatalf(
					"events are not equal\n\nInput: %s\n\nExpected: %s\n\nResult: %s",
					prettyPrint(test.input),
					prettyPrint(test.expect[i]),
					prettyPrint(result[i]),
				)
			}
		}
	}
}
