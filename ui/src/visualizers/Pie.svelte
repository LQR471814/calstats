<script lang="ts">
	import type { PieData } from "../analysis";
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Label } from "$lib/components/ui/label";
	import * as d3 from "d3";
	import { cn } from "$lib/utils";

	let { data }: { data: PieData } = $props();

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
	let label = $state<string>();
	let percent = $state<number>();
	let duration = $state<number>();

	function formatDuration(duration: number) {
		const out: string[] = [];

		const years = Math.floor(duration / 31_536_000);
		const yearsR = duration % 31_536_000;
		if (years > 1) {
			out.push(`${years} years`);
		} else if (years === 1) {
			out.push(`${years} year`);
		}

		const weeks = Math.floor(yearsR / 604_800);
		const weeksR = duration % 604_800;
		if (weeks > 1) {
			out.push(`${weeks} weeks`);
		} else if (weeks === 1) {
			out.push(`${weeks} week`);
		}

		const days = Math.floor(weeksR / 86_400);
		const daysR = duration % 86400;
		if (days > 1) {
			out.push(`${days} days`);
		} else if (days === 1) {
			out.push(`${days} day`);
		}

		const hours = Math.floor(daysR / 3_600);
		const hoursR = duration % 3600;
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
</script>

<h3>Pie chart</h3>

<svg
	viewBox={`-${outerRadius} -${outerRadius} ${width} ${width}`}
	{width}
	height={width}
>
	<g>
		{#each data as d, i}
			{#if arcs[i]}
				{@const hover = () => {
					label = d.category;
					duration = d.time;
					percent =
						(arcs[i].endAngle - arcs[i].startAngle) / (2 * Math.PI);
					hovered = i;
				}}
				{@const blur = () => {
					label = undefined;
					duration = undefined;
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
		{label}
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
		{#if duration !== undefined}
			{formatDuration(duration)}
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
				{checked}
				aria-labelledby={`pie-label-${i}`}
				onclick={() => {
					const idx = disabled.indexOf(d.category);
					if (idx >= 0) {
						disabled.splice(idx, 1);
						return;
					}
					disabled.push(d.category);
				}}
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
