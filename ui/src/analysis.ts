import type { EventsResponse } from "$api/api_pb";
import type { Interval } from "./event-state.svelte";

export type PieData = {
	category: string
	time: number
}[]

export function getPieData(interval: Interval, events: EventsResponse): PieData | undefined {
	if (!events) {
		return
	}

	let categories: PieData = new Array(events.tags.length + 1)
	for (let i = 0; i < events.tags.length; i++) {
		categories[i] = {
			category: events.tags[i],
			time: 0,
		}
	}
	categories[events.tags.length] = {
		category: "Unknown",
		time: 0,
	}

	// count time spent in each category
	let countedSeconds = 0
	for (const e of events.events) {
		if (!e.duration) {
			throw new Error("undefined duration")
		}

		const seconds = Number(e.duration.seconds)
		const tagIdx = e.tags.length > 0 ?
			e.tags[0] : events.tags.length

		categories[tagIdx].time += seconds
		countedSeconds += seconds
	}

	// count untracked or time without an event on it
	const totalDuration = interval.end.since(interval.start)
	const untrackedTime = totalDuration.subtract({
		seconds: countedSeconds,
	})
	categories[events.tags.length].time += untrackedTime.total({ unit: "seconds" })

	// sort categories
	categories.sort((a, b) => b.time - a.time)

	return categories
}
