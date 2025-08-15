import type { Event, EventsResponse } from "$api/api_pb";
import type { Interval } from "./event-model.svelte";

export function formatDuration(seconds: number): string {
	const out: string[] = [];

	const years = Math.floor(seconds / 31_536_000);
	const yearsR = seconds % 31_536_000;
	if (years > 1) {
		out.push(`${years} years`);
	} else if (years === 1) {
		out.push(`${years} year`);
	}

	const weeks = Math.floor(yearsR / 604_800);
	const weeksR = seconds % 604_800;
	if (weeks > 1) {
		out.push(`${weeks} weeks`);
	} else if (weeks === 1) {
		out.push(`${weeks} week`);
	}

	const days = Math.floor(weeksR / 86_400);
	const daysR = seconds % 86400;
	if (days > 1) {
		out.push(`${days} days`);
	} else if (days === 1) {
		out.push(`${days} day`);
	}

	const hours = Math.floor(daysR / 3_600);
	const hoursR = seconds % 3600;
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

export class CategoryStat {
	category: string;
	time: number;
	proportion: number;
	events: Event[];

	constructor(category: string, time: number, proportion: number) {
		this.category = category;
		this.time = time;
		this.proportion = proportion;
		this.events = [];
	}

	add(e: Event) {
		this.events.push(e);
		if (!e.duration) {
			throw new Error("undefined duration");
		}
		this.time += Number(e.duration.seconds);
	}
}

export function getCategoryStats(
	interval: Interval,
	events: EventsResponse,
	disabled: string[],
): CategoryStat[] | undefined {
	if (!events) {
		return;
	}

	const tags = [...events.tags, "Unknown"];
	const unknownTagIdx = tags.length - 1;
	const disabledTable = new Array<boolean>(tags.length);
	for (const d of disabled) {
		disabledTable[tags.indexOf(d)] = true;
	}

	const categories: CategoryStat[] = new Array(tags.length);
	for (let i = 0; i < tags.length; i++) {
		categories[i] = new CategoryStat(tags[i], 0, 0);
	}

	// count time spent in each category
	let trackedSeconds = 0;
	let disabledSeconds = 0;
	for (const e of events.events) {
		if (!e.duration) {
			throw new Error("undefined duration");
		}
		const tagIdx = e.tags.length > 0 ? e.tags[0] : unknownTagIdx;
		trackedSeconds += Number(e.duration.seconds); // add counted seconds regardless of disabled tags
		if (!disabledTable[tagIdx]) {
			categories[tagIdx].add(e);
		} else {
			disabledSeconds += Number(e.duration.seconds);
		}
	}

	// count untracked or time without an event on it
	const totalDuration = interval.end.since(interval.start);

	const untrackedSeconds = totalDuration
		.subtract({ seconds: trackedSeconds })
		.total({ unit: "seconds" });
	if (!disabledTable[unknownTagIdx]) {
		categories[unknownTagIdx].time += untrackedSeconds;
	} else {
		disabledSeconds += untrackedSeconds;
	}

	const totalSeconds =
		totalDuration.total({ unit: "seconds" }) - disabledSeconds;
	for (const cat of categories) {
		cat.proportion = cat.time / totalSeconds;
		cat.events.sort(
			(a, b) => Number(b.duration!.seconds) - Number(a.duration!.seconds),
		);
	}

	// sort categories
	categories.sort((a, b) => b.time - a.time);

	return categories;
}
