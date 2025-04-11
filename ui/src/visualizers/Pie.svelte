<script lang="ts">
	import type { CategoryStat } from "../analysis";
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Label } from "$lib/components/ui/label";
	import * as d3 from "d3";
	import { cn } from "$lib/utils";

	let { data }: { data: CategoryStat[] } = $props();

	let disabled = $state<string[]>([]);

	const color = d3.scaleOrdinal(d3.schemeObservable10);
	const arcs = $derived.by(() => {
		const pie = d3.pie();

		const arcData = new Array<number>(data.length);
		for (let i = 0; i < data.length; i++) {
			if (disabled.includes(data[i].category)) {
				arcData[i] = 0;
				continue;
			}
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
				{stat.formatTime()}
			{/if}
		</text>
	</svg>

	<div class="flex flex-col gap-2 flex-wrap max-h-[200px] w-fit">
		{#each data as d, i}
			{@const c = color(d.category)}
			{@const checked = !disabled.includes(d.category)}
			<div class="w-fit">
				<Checkbox
					id={`pie-checkbox-${i}`}
					style={`border-color: ${c}; background-color: ${checked ? c : "transparent"}`}
					bind:checked={() => checked,
					() => {
						const idx = disabled.indexOf(d.category);
						if (idx >= 0) {
							disabled.splice(idx, 1);
							return;
						}
						disabled.push(d.category);
					}}
					aria-labelledby={`pie-label-${i}`}
				/>
				<Label
					id={`pie-label-${i}`}
					for={`pie-checkbox-${i}`}
					class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
				>
					{d.category}
				</Label>
			</div>
		{/each}
	</div>
</div>
