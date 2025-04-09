<script lang="ts">
	import "./app.css";
	import { client } from "./rpc";
	import { toast } from "svelte-sonner";
	import { useQuery } from "@sveltestack/svelte-query";

	const metaQuery = useQuery("meta", {
		queryFn: () => client.calendar({}),
		onError: (err) => {
			toast.error("RPC Error", {
				description: String(err),
				duration: 3000,
			});
		},
	});
</script>

<main class="flex flex-col gap-6 p-6">
	<h1>Schedule</h1>
	<div class="grid grid-cols-[min-content_1fr] gap-3 max-w-[400px]">
		<span>Server</span>
		<code class="w-fit">{$metaQuery.data?.calendarServer ?? "loading..."}</code>
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
</main>
