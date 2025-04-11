<script lang="ts">
	import { type CategoryStat, formatDuration } from "../analysis";
	import * as d3 from "d3";
	import ArrowDown from "@lucide/svelte/icons/chevron-down";
	import type { EventsResponse } from "$api/api_pb";
	import { cn } from "$lib/utils";

	const color = d3.scaleOrdinal(d3.schemeObservable10);

	let { data, ev }: { data: CategoryStat[]; ev: EventsResponse } = $props();

	const normFactor = $derived(1 / data[0].proportion);

	let expanded = $state<string>();
</script>

{#snippet rect(
	name: string,
	time: string,
	proportion: number,
	normalizeFactor: number,
	color: string,
)}
	<div
		class="rounded-lg px-2 py-1"
		style:background-color={color}
		style:width={`${proportion * normalizeFactor * 100}%`}
	>
		<p class="text-sm text-nowrap">{name}</p>
		<span class="text-sm text-nowrap">
			{time} ({Math.round(proportion * 1000) / 10}%)
		</span>
	</div>
{/snippet}

<div class="flex flex-col gap-6 w-[300px]">
	<h3>List</h3>
	<div class="flex flex-col gap-2">
		{#each data.filter((d) => d.proportion > 0) as d}
			{@const catColor = color(d.category)}
			{@const isExpanded = expanded === d.category}
			{@const isNotExpandable = d.events.length === 0}

			<button
				class={cn(
					"flex gap-3 group relative text-left",
					isNotExpandable ? "cursor-not-allowed" : "",
				)}
				disabled={isNotExpandable}
				onclick={() => {
					if (isExpanded) {
						expanded = undefined;
						return;
					}
					expanded = d.category;
				}}
			>
				{@render rect(
					d.category,
					formatDuration(d.time),
					d.proportion,
					normFactor,
					catColor,
				)}
				{#if d.events.length > 0}
					<div
						class="absolute -left-8 top-1/2 -translate-y-1/2 transition-all opacity-0 group-hover:opacity-100"
					>
						<ArrowDown
							class={cn(
								"my-auto min-w-[24px] max-w-[24px] transition-all -rotate-90",
								isExpanded ? "rotate-0" : "",
							)}
						/>
					</div>
				{/if}
			</button>

			{#if isExpanded}
				{@const normFactor =
					d.time / Number(d.events[0].duration!.seconds)}
				<div class="pl-3 flex flex-col gap-1">
					{#each d.events as e}
						{@const name = ev.eventNames[e.name]}
						{@render rect(
							name,
							formatDuration(Number(e.duration!.seconds)),
							Number(e.duration!.seconds) / d.time,
							normFactor,
							catColor,
						)}
					{/each}
				</div>
			{/if}
		{/each}
	</div>
</div>
