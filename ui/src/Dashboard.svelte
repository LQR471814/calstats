<script lang="ts">
	import "./app.css";
	import { client } from "./rpc";
	import { toast } from "svelte-sonner";
	import { createQuery } from "@tanstack/svelte-query";
	import Pie from "./visualizers/Pie.svelte";
	import { getPieData } from "./analysis";
	import { createEventState } from "./event-state.svelte";

	const { interval, events } = createEventState()

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

	const pieData = $derived(
		events.response ? getPieData(events.response) : undefined,
	);
</script>

<main class="flex flex-col gap-6 p-6">
	<h1>Schedule</h1>

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

	{#if pieData}
		<Pie data={pieData} />
	{/if}
</main>
