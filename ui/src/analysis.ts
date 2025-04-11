import type { EventsResponse } from "$api/api_pb";
import type { Interval } from "./event-state.svelte";

export class CategoryStat {
	category: string
	time: number

	constructor(category: string, time: number) {
		this.category = category
		this.time = time
	}

	formatTime(): string {
		const out: string[] = [];

		const years = Math.floor(this.time / 31_536_000);
		const yearsR = this.time % 31_536_000;
		if (years > 1) {
			out.push(`${years} years`);
		} else if (years === 1) {
			out.push(`${years} year`);
		}

		const weeks = Math.floor(yearsR / 604_800);
		const weeksR = this.time % 604_800;
		if (weeks > 1) {
			out.push(`${weeks} weeks`);
		} else if (weeks === 1) {
			out.push(`${weeks} week`);
		}

		const days = Math.floor(weeksR / 86_400);
		const daysR = this.time % 86400;
		if (days > 1) {
			out.push(`${days} days`);
		} else if (days === 1) {
			out.push(`${days} day`);
		}

		const hours = Math.floor(daysR / 3_600);
		const hoursR = this.time % 3600;
		if (hours > 1) {
			out.push(`${hours} hours`);
		} else if (hours === 1) {
			out.push(`${hours} hour`);
		}

		const minutes = Math.floor(hoursR / 60);
		if (minutes > 1) {
			out.push(`${minutes} minutes`);
		} else if (minutes === 1) {
			out.push(`${minutes} minute`);
		}

		return out.slice(0, 2).join(" ");
	}
}

export function getPieData(interval: Interval, events: EventsResponse): CategoryStat[] | undefined {
	if (!events) {
		return
	}

	let categories: CategoryStat[] = new Array(events.tags.length + 1)
	for (let i = 0; i < events.tags.length; i++) {
		categories[i] = new CategoryStat(events.tags[i], 0)
	}
	categories[events.tags.length] = new CategoryStat("Unknown", 0)

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
