<script lang="ts">
	import { EventModel, IntervalOption } from "./event-state.svelte";
	import { cn } from "$lib/utils"
	import * as Select from "$lib/components/ui/select";
	import Button from "$lib/components/ui/button/button.svelte";
	import Refresh from "@lucide/svelte/icons/refresh-cw";
	import LoaderCircle from "@lucide/svelte/icons/loader-circle";

	const { model, className }: { model: EventModel, className?: string } = $props();

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

<div class={cn("flex flex-col gap-3", className)}>
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

	<Select.Root type="single" bind:value={model.option as unknown as string}>
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
