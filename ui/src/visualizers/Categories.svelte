<script lang="ts">
	import type { CategoryStat } from "../analysis";
	import * as d3 from "d3";
	import ArrowDown from "@lucide/svelte/icons/chevron-down";

	const color = d3.scaleOrdinal(d3.schemeObservable10);

	let { data }: { data: CategoryStat[] } = $props();

	const scalar = $derived(1 / data[0].proportion);

	let expanded = $state<string>();
</script>

{#snippet rect(name: string, time: string, proportion: number, color: string)}
	<div
		class="rounded-lg px-2 py-1"
		style:background-color={color}
		style:width={`${proportion * 100}%`}
	>
		<p class="text-sm text-nowrap">{name}</p>
		<span class="text-sm text-nowrap">
			{time} ({Math.round(proportion * 1000) / 10}%)
		</span>
	</div>
{/snippet}

<div class="flex flex-col gap-6 w-[300px]">
	<h3>Categories</h3>
	<div class="flex flex-col gap-2">
		{#each data as d}
			{#if d.proportion > 0}
				{@const c = color(d.category)}
				<button
					class="flex gap-3 group relative text-left"
					onclick={() => {
						expanded = d.category;
					}}
				>
					<div
						class="rounded-lg px-2 py-1"
						style:background-color={c}
						style:width={`${d.proportion * scalar * 100}%`}
					>
						<p class="text-sm text-nowrap">{d.category}</p>
						<span class="text-sm text-nowrap">
							{d.formatTime()} ({Math.round(d.proportion * 1000) /
								10}%)
						</span>
					</div>
					<div
						class="absolute -left-8 top-1/2 -translate-y-1/2 transition-all opacity-0 group-hover:opacity-100"
					>
						<ArrowDown class="my-auto min-w-[24px] max-w-[24px]" />
					</div>
				</button>
				{#if expanded === d.category}
					<div class="pl-3">
						{#each d.events as e}
							<div>
								<p class="text-sm text-nowrap">{d.category}</p>
								<span class="text-sm text-nowrap">
									{d.formatTime()} ({Math.round(
										d.proportion * 1000,
									) / 10}%)
								</span>
							</div>
						{/each}
					</div>
				{/if}
			{/if}
		{/each}
	</div>
</div>
