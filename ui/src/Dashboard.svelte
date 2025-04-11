<script lang="ts">
	import "./app.css";
	import { client } from "./rpc";
	import { toast } from "svelte-sonner";
	import { createQuery } from "@tanstack/svelte-query";
	import Pie from "./visualizers/Pie.svelte";
	import { getPieData } from "./analysis";
	import { EventState, IntervalOption } from "./event-state.svelte";
	import * as Select from "$lib/components/ui/select";

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

	const state = new EventState();

	const pieData = $derived(
		state.events ? getPieData(state.events) : undefined,
	);

	const intvOptLabel: { [key in IntervalOption]: string } = {
		[IntervalOption.THIS_DAY]: "Today",
		[IntervalOption.THIS_WEEK]: "This week",
		[IntervalOption.THIS_MONTH]: "This month",
		[IntervalOption.THIS_YEAR]: "This year",
		[IntervalOption.LAST_3_MONTHS]: "Last 3 months",
		[IntervalOption.LAST_6_MONTHS]: "Last 6 months",
		[IntervalOption.CUSTOM]: "Custom",
	};
</script>

<main class="flex gap-6 p-6">
	<div class="flex flex-col gap-6 flex-1">
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
	</div>

	<div class="flex flex-col gap-3">
		<h4>Analysis interval</h4>
		<Select.Root
			type="single"
			bind:value={state.option as unknown as string}
		>
			<Select.Trigger class="w-[180px]">
				{intvOptLabel[state.option]}
			</Select.Trigger>
			<Select.Content>
				{#each Object.keys(IntervalOption) as key}
					{@const value =
						IntervalOption[key as keyof typeof IntervalOption]}
					{@const label = intvOptLabel[value]}
					<Select.Item value={value as unknown as string} {label}>
						{label}
					</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
	</div>
</main>
