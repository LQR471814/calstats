<script lang="ts">
	import "./app.css";
	import { client } from "./rpc";
	import { toast } from "svelte-sonner";
	import { createQuery } from "@tanstack/svelte-query";
	import Pie from "./visualizers/Pie.svelte";
	import { getCategoryStats } from "./analysis";
	import { EventModel, IntervalOption } from "./event-state.svelte";
	import * as Select from "$lib/components/ui/select";
	import Button from "$lib/components/ui/button/button.svelte";
	import Refresh from "@lucide/svelte/icons/refresh-cw";
	import LoaderCircle from "@lucide/svelte/icons/loader-circle";
	import List from "./visualizers/List.svelte";

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

	const intvOptLabel: { [key in IntervalOption]: string } = {
		[IntervalOption.THIS_DAY]: "Today",
		[IntervalOption.THIS_WEEK]: "This week",
		[IntervalOption.THIS_MONTH]: "This month",
		[IntervalOption.THIS_YEAR]: "This year",
		[IntervalOption.LAST_3_MONTHS]: "Last 3 months",
		[IntervalOption.LAST_6_MONTHS]: "Last 6 months",
		[IntervalOption.CUSTOM]: "Custom",
	};

	function padDateDigit(value: number): string {
		return value.toString().padStart(2, "0");
	}

	let pressed = $state(false);
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
		<div class="flex flex-col gap-3 top-0 sticky py-6">
			<h4>Analysis interval</h4>
			<div class="grid gap-3 grid-cols-[min-content_1fr]">
				<span>From</span>
				<code>
					{model.interval.start.year}-{padDateDigit(
						model.interval.start.month,
					)}-{padDateDigit(model.interval.start.day)}
				</code>
				<span>To</span>
				<code>
					{model.interval.end.year}-{padDateDigit(
						model.interval.end.month,
					)}-{padDateDigit(model.interval.end.day)}
				</code>
			</div>
			<Select.Root
				type="single"
				bind:value={model.option as unknown as string}
			>
				<Select.Trigger class="w-[180px]">
					{intvOptLabel[model.option]}
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
			<div class="flex justify-end">
				<Button
					class="w-fit"
					variant={pressed ? "ghost" : "default"}
					disabled={pressed}
					onclick={() => {
						pressed = true;
						model.refresh().finally(() => {
							pressed = false;
						});
					}}
				>
					{#if pressed}
						<LoaderCircle class="animate-spin" />
					{:else}
						<Refresh />
					{/if}
					Refresh
				</Button>
			</div>
		</div>
	</div>
</main>
