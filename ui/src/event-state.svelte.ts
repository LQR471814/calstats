import { instantToTimestamp } from "$lib/time";
import { Temporal } from "@js-temporal/polyfill";
import { client } from "./rpc";
import { toast } from "svelte-sonner"
import type { EventsResponse } from "$api/api_pb";

export type Interval = {
	start: Temporal.ZonedDateTime;
	end: Temporal.ZonedDateTime;
};

export enum IntervalOption {
	THIS_DAY = 0,
	THIS_WEEK = 1,
	THIS_MONTH = 2,
	THIS_YEAR = 3,
	LAST_3_MONTHS = 4,
	LAST_6_MONTHS = 5,
	CUSTOM = 6,
}

export function createEventState() {
	const now = Temporal.Now.zonedDateTimeISO();

	const interval = $state<{
		custom: Interval
		option: IntervalOption
	}>({
		option: IntervalOption.THIS_WEEK,
		custom: {
			start: now.subtract({ weeks: 1 }),
			end: now.add({ weeks: 1 }),
		}
	});

	const _interval = $derived.by((): Interval => {
		switch (interval.option) {
			case IntervalOption.CUSTOM:
				return interval.custom;
			case IntervalOption.THIS_DAY: {
				const start = now.subtract({
					hours: now.hour,
					minutes: now.minute,
					seconds: now.second,
					milliseconds: now.millisecond,
					nanoseconds: now.nanosecond,
				});
				const end = start.add({
					hours: 23,
					minutes: 59,
					seconds: 59,
					milliseconds: 999,
					nanoseconds: 999,
				});
				return { start, end };
			}
			case IntervalOption.THIS_WEEK: {
				const start = now.subtract({
					days: now.dayOfWeek - 1,
					hours: now.hour,
					minutes: now.minute,
					seconds: now.second,
					milliseconds: now.millisecond,
					nanoseconds: now.nanosecond,
				});
				const end = start.add({
					days: now.daysInWeek,
				});
				return { start, end };
			}
			case IntervalOption.THIS_MONTH: {
				const start = now.subtract({
					days: now.day - 1,
					hours: now.hour,
					minutes: now.minute,
					seconds: now.second,
					milliseconds: now.millisecond,
					nanoseconds: now.nanosecond,
				});
				const end = start.add({
					months: 1,
				});
				return { start, end };
			}
			case IntervalOption.THIS_YEAR: {
				const start = now.subtract({
					days: now.dayOfYear,
					hours: now.hour,
					minutes: now.minute,
					seconds: now.second,
					milliseconds: now.millisecond,
					nanoseconds: now.nanosecond,
				});
				const end = start.add({
					years: 1,
				});
				return { start, end };
			}
			case IntervalOption.LAST_3_MONTHS: {
				const start = now.subtract({
					months: 3,
					hours: now.hour,
					minutes: now.minute,
					seconds: now.second,
					milliseconds: now.millisecond,
					nanoseconds: now.nanosecond,
				});
				const end = now;
				return { start, end };
			}
			case IntervalOption.LAST_6_MONTHS: {
				const start = now.subtract({
					months: 6,
					hours: now.hour,
					minutes: now.minute,
					seconds: now.second,
					milliseconds: now.millisecond,
					nanoseconds: now.nanosecond,
				});
				const end = now;
				return { start, end };
			}
		}
	});

	const events = $state<{ response?: EventsResponse }>({ response: undefined })

	$effect(() => {
		client.events({
			interval: {
				start: instantToTimestamp(_interval.start.toInstant()),
				end: instantToTimestamp(_interval.end.toInstant()),
			},
		})
			.then((res) => {
				events.response = res
			})
			.catch(err => {
				toast.error("RPC Error", {
					description: String(err),
					duration: 3000,
				})
			})
	})

	return { interval, events }
}

