import type { EventsResponse } from "$api/api_pb";

export type PieData = {
	category: string
	time: number
}[]

export function getPieData(events: EventsResponse): PieData | undefined {
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

	let total = 0
	for (const e of events.events) {
		if (!e.duration) {
			throw new Error("undefined duration")
		}
		total += Number(e.duration.seconds)
		categories[e.tags.length > 0 ?
			e.tags[0] :
			events.tags.length].time += Number(e.duration.seconds)
	}

	categories.sort((a, b) => b.time - a.time)

	return categories
}
