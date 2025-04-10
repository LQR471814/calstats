<script lang="ts">
	import type { PieData } from "../analysis.svelte";
	import * as d3 from "d3";

	let { data }: { data: PieData } = $props();
	const color = d3.scaleOrdinal(d3.schemeObservable10);
	const arcs = $derived.by(() => {
		const pie = d3.pie();
		return pie(data.map((d) => d.time));
	});
</script>

<svg>
	<text text-anchor="middle" fill="currentColor">Label</text>
	<tspan x={0} y={0} dy="-0.1em" font-size="3em">Percent%</tspan>
	<tspan x={0} y={0} dy="1.5em">of total time spent on this category.</tspan>
	<g>
		{#each data as d, i}
			<path
				fill={color(d.category)}
				d={d3.arc()({
					innerRadius: 100,
					outerRadius: 200,
					startAngle: arcs[i].startAngle,
					endAngle: arcs[i].endAngle,
				})}
			></path>
		{/each}
	</g>
</svg>
