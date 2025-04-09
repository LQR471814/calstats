<script lang="ts">
	import { events } from "../event-state.svelte";
	import { pieData } from "../analysis.svelte";
	import * as d3 from "d3";

	const width = 500;
	const height = Math.min(500, width / 2);
	const outerRadius = height / 2 - 10;
	const innerRadius = outerRadius * 0.75;
	const tau = 2 * Math.PI;
	const color = d3.scaleOrdinal(d3.schemeObservable10);

	const data = $derived(() => {
		const out: { [key: string]: number }[] = [];
		for (const category of pieData()) {
			out.push({
				[category.category]: category.time,
			});
		}
		return out;
	});

	$effect(() => {
		const svg = d3
			.create("svg")
			.attr("viewBox", [-width / 2, -height / 2, width, height]);

		const arc = d3.arc().innerRadius(innerRadius).outerRadius(outerRadius);

		const pie = d3
			.pie()
			.sort(null)
			.value((d) => d["apples"]);

		const path = svg
			.datum(data())
			.selectAll("path")
			.data(pie)
			.join("path")
			.attr("fill", (d, i) => color(i.toString()))
			.attr("d", arc)
			.each(function (d) {
				this._current = d;
			});
	});
</script>

