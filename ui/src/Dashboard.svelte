<script lang="ts">
	import "./app.css";
	import { client } from "./rpc";
	import { toast } from "svelte-sonner";
	import { createQuery } from "@tanstack/svelte-query";
	import Pie from "./visualizers/Pie.svelte";
	import { getCategoryStats } from "./analysis";
	import { EventModel } from "./event-state.svelte";
	import List from "./visualizers/List.svelte";
	import AnalysisInterval from "./AnalysisInterval.svelte";

	const metaQuery = createQuery({
		queryKey: ["meta"],
		queryFn: () => client.calendar({}),
	});

	$effect(() => {
		if (!$metaQuery.error) {
			return;
		}
		toast.error("RPC Error", {
			description: String($metaQuery.error),
			duration: 3000,
		});
	});

	const model = new EventModel();

	const catStats = $derived(
		model.events
			? getCategoryStats(model.interval, model.events)
			: undefined,
	);
</script>

<main class="flex gap-6 px-6">
	<div class="flex flex-col gap-6 flex-1 py-6">
		<h1>Schedule statistics</h1>

		<div class="grid grid-cols-[min-content_1fr] gap-3 max-w-[400px]">
			<span>Server</span>
			<code class="w-fit">
				{$metaQuery.data?.calendarServer ?? "loading..."}
			</code>
			<span>Calendars</span>
			{#if $metaQuery.data}
				<div>
					{#each $metaQuery.data.names as name, i}
						{#if i > 0}
							<span class="mr-1">,</span>
						{/if}
						<code>{name}</code>
					{/each}
				</div>
			{:else}
				<code>loading...</code>
			{/if}
		</div>

		<div class="flex flex-wrap gap-6">
			{#if catStats && model.events}
				<Pie data={catStats} />
				<List data={catStats} ev={model.events} />
			{/if}
		</div>
	</div>

	<div>
		<AnalysisInterval {model} className="sticky top-0 py-6" />
	</div>
</main>
