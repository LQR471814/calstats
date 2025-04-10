<script lang="ts">
	import "./app.css";
	import { client } from "./rpc";
	import { toast } from "svelte-sonner";
	import { createQuery } from "@tanstack/svelte-query";
	import Pie from "./visualizers/Pie.svelte";
	import { getPieData } from "./analysis";
	import { createEventState, IntervalOption } from "./event-state.svelte";
	import { CalendarDate } from "@internationalized/date";
	import * as Select from "$lib/components/ui/select";

	const { interval, events } = createEventState();

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

	<div>
		<Select.Root>
			<Select.Trigger
				class="w-[180px]"
				value={interval.option}
				onchange={(e) => {
					console.log("changed to", e.currentTarget.value);
				}}
			>
				<Select.Value />
			</Select.Trigger>
			<Select.Content>
				<Select.Label>Analysis Interval</Select.Label>

				<Select.Item value={IntervalOption.THIS_DAY} label="Today">
					Today
				</Select.Item>
				<Select.Item value={IntervalOption.THIS_WEEK} label="This week">
					This week
				</Select.Item>
				<Select.Item
					value={IntervalOption.THIS_MONTH}
					label="This month"
				>
					This month
				</Select.Item>
				<Select.Item value={IntervalOption.THIS_YEAR} label="This year">
					This year
				</Select.Item>
				<Select.Item
					value={IntervalOption.LAST_3_MONTHS}
					label="Last 3 months"
				>
					Last 3 months
				</Select.Item>
				<Select.Item
					value={IntervalOption.LAST_6_MONTHS}
					label="Last 6 months"
				>
					Last 6 months
				</Select.Item>
				<Select.Item value={IntervalOption.CUSTOM} label="Custom">
					Custom
				</Select.Item>
			</Select.Content>
		</Select.Root>
	</div>
</main>
