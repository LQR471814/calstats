<script lang="ts">
	import { formatDuration, type CategoryStat } from "../analysis";
	import * as d3 from "d3";
	import { cn } from "$lib/utils";
	import { color } from "$lib/color"

	let { data }: { data: CategoryStat[] } = $props();

	const arcs = $derived.by(() => {
		const pie = d3.pie();

		const arcData = new Array<number>(data.length);
		for (let i = 0; i < data.length; i++) {
			arcData[i] = data[i].time;
		}
		return pie(arcData);
	});

	const innerRadius = 80;
	const outerRadius = 125;
	const width = outerRadius * 2;

	let hovered = $state<number>();
	let percent = $state<number>();
	let stat = $state<CategoryStat>();
</script>

<div class="flex flex-col gap-6">
	<h3>Pie chart</h3>

	<svg
		class="mx-auto"
		viewBox={`-${outerRadius} -${outerRadius} ${width} ${width}`}
		{width}
		height={width}
	>
		<g>
			{#each data as d, i}
				{#if arcs[i]}
					{@const hover = () => {
						stat = d;
						percent =
							(arcs[i].endAngle - arcs[i].startAngle) /
							(2 * Math.PI);
						hovered = i;
					}}
					{@const blur = () => {
						stat = undefined;
						percent = undefined;
						hovered = undefined;
					}}
					<path
						fill={color(d.category)}
						class={cn(
							"select-none outline-none",
							hovered !== undefined && hovered !== i
								? "opacity-10"
								: "",
						)}
						d={d3.arc()({
							innerRadius,
							outerRadius,
							startAngle: arcs[i].startAngle,
							endAngle: arcs[i].endAngle,
						})}
						role="tooltip"
						aria-label={d.category}
						onmouseover={hover}
						onmouseout={blur}
						onfocus={hover}
						onblur={blur}
					></path>
				{/if}
			{/each}
		</g>

		<text
			class="font-bold text-lg pointer-events-none"
			text-anchor="middle"
			fill="currentColor"
			dy="-1em"
		>
			{stat?.category}
		</text>
		<text
			class="pointer-events-none"
			text-anchor="middle"
			fill="currentColor"
			dy="0.2em"
		>
			{#if percent !== undefined}
				{Math.round(percent * 1000) / 10}%
			{/if}
		</text>
		<text
			class="pointer-events-none"
			text-anchor="middle"
			fill="currentColor"
			dy="1.4em"
		>
			{#if stat !== undefined}
				{formatDuration(stat.time)}
			{/if}
		</text>
	</svg>

</div>
