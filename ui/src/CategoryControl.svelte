<script lang="ts">
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Label } from "$lib/components/ui/label";
	import { color } from "$lib/color"

	let {
		categories,
		disabled = $bindable([]),
	}: { categories: string[]; disabled: string[] } = $props();
</script>

<div class="flex flex-col gap-6">
	<h4>Categories</h4>

	<div class="flex flex-col gap-2 w-fit">
		{#each categories as cat, i}
			{@const c = color(cat)}
			{@const checked = !disabled.includes(cat)}
			<div class="w-fit">
				<Checkbox
					id={`pie-checkbox-${i}`}
					style={`border-color: ${c}; background-color: ${checked ? c : "transparent"}`}
					bind:checked={() => checked,
					() => {
						const idx = disabled.indexOf(cat);
						if (idx >= 0) {
							disabled.splice(idx, 1);
							return;
						}
						disabled.push(cat);
					}}
					aria-labelledby={`pie-label-${i}`}
				/>
				<Label
					id={`pie-label-${i}`}
					class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
				>
					{cat}
				</Label>
			</div>
		{/each}
	</div>
</div>
