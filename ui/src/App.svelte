<script lang="ts">
	import "./app.css";
	import Dashboard from "./Dashboard.svelte";
	import { ModeWatcher } from "mode-watcher";
	import { Toaster } from "$lib/components/ui/sonner";
	import {
		QueryClient,
		QueryClientProvider,
	} from "@sveltestack/svelte-query";

	const queryClient = new QueryClient();
</script>

<QueryClientProvider client={queryClient}>
	<Dashboard />
</QueryClientProvider>

<Toaster />
<ModeWatcher defaultMode="dark" />
