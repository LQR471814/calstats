<script lang="ts">
	import type { CategoryStat } from "../analysis";
	import * as d3 from "d3";

	const color = d3.scaleOrdinal(d3.schemeObservable10);

	let { data }: { data: CategoryStat[] } = $props();
</script>

<div class="flex flex-col gap-6 w-[400px]">
	<h3>Categories</h3>
	<div class="flex flex-col gap-2">
		{#each data as d}
			<div
				class="rounded-lg px-2 py-1"
				style:background-color={color(d.category)}
				style:width={`${d.proportion * 100}%`}
			>
				<p class="text-sm text-nowrap">{d.category}</p>
				<span class="text-sm text-nowrap">
					{d.formatTime()} ({Math.round(d.proportion * 1000) / 10}%)
				</span>
			</div>
		{/each}
	</div>
</div>
