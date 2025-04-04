<script lang="ts">
	import { instantToTimestamp } from "$lib/time";
	import { Temporal } from "@js-temporal/polyfill";
	import { client } from "./rpc";

	const now = Temporal.Now.zonedDateTimeISO();

	const start = now.subtract({ weeks: 1 });
	const end = now.add({ weeks: 1 });

	client.events({
		interval: {
			start: instantToTimestamp(start.toInstant()),
			end: instantToTimestamp(end.toInstant()),
		},
	}).then((res) => {
		console.log(res)
	});
</script>

<main>
	<p>hello</p>
</main>
