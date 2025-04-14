<script lang="ts">
	import { EventModel, IntervalOption } from "./event-state.svelte";
	import { cn } from "$lib/utils";
	import * as Select from "$lib/components/ui/select";
	import Button from "$lib/components/ui/button/button.svelte";
	import Refresh from "@lucide/svelte/icons/refresh-cw";
	import LoaderCircle from "@lucide/svelte/icons/loader-circle";
	import CalendarIcon from "lucide-svelte/icons/calendar";
	import { Calendar } from "$lib/components/ui/calendar";
	import * as Popover from "$lib/components/ui/popover";
	import { Temporal } from "@js-temporal/polyfill";
	import { zonedToI18n } from "$lib/time";

	const { model, className }: { model: EventModel; className?: string } =
		$props();

	const intvOptLabel: { [key in IntervalOption]: string } = {
		[IntervalOption.THIS_DAY]: "Today",
		[IntervalOption.THIS_WEEK]: "This week",
		[IntervalOption.THIS_MONTH]: "This month",
		[IntervalOption.THIS_YEAR]: "This year",
		[IntervalOption.LAST_3_MONTHS]: "Last 3 months",
		[IntervalOption.LAST_6_MONTHS]: "Last 6 months",
		[IntervalOption.CUSTOM]: "Custom",
	};

	function pad2Digit(value: number): string {
		return value.toString().padStart(2, "0");
	}

	let pressed = $state(false);
</script>

{#snippet datePicker(
	value: Temporal.ZonedDateTime,
	onchange: (date: Temporal.ZonedDateTime) => void,
	min?: Temporal.ZonedDateTime,
	max?: Temporal.ZonedDateTime,
)}
	<Popover.Root>
		<Popover.Trigger>
			<Button
				variant="outline"
				class="w-fit justify-start text-left font-normal"
			>
				<CalendarIcon class="mr-2 h-4 w-4" />
				{value.year}-{pad2Digit(value.month)}-{pad2Digit(value.day)}
			</Button>
		</Popover.Trigger>
		<Popover.Content class="w-auto p-0">
			<Calendar
				type="single"
				minValue={min ? zonedToI18n(min) : undefined}
				maxValue={max ? zonedToI18n(max) : undefined}
				bind:value={() => zonedToI18n(value),
				(date) => {
					if (!date) {
						return;
					}
					onchange(
						new Temporal.ZonedDateTime(
							BigInt(date.toDate().getTime()) * BigInt(1_000_000),
							date.timeZone,
						),
					);
				}}
				initialFocus
			/>
		</Popover.Content>
	</Popover.Root>
{/snippet}

<div class={cn("flex flex-col gap-3", className)}>
	<h4>Analysis interval</h4>

	<div class="grid gap-3 grid-cols-[min-content_1fr]">
		<span class="my-auto">From</span>
		{@render datePicker(
			model.interval.start,
			(date) => {
				model.option = IntervalOption.CUSTOM;
				model.customBounds.start = date;
			},
			undefined,
			model.interval.end,
		)}

		<span class="my-auto">To</span>
		{@render datePicker(
			model.interval.end,
			(date) => {
				model.option = IntervalOption.CUSTOM;
				model.customBounds.end = date;
			},
			model.interval.start,
			undefined,
		)}
	</div>

	<Select.Root type="single" bind:value={model.option as unknown as string}>
		<Select.Trigger class="w-full">
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
