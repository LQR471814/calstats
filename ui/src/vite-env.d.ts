/// <reference types="svelte" />
/// <reference types="vite/client" />

import type { Temporal } from "@js-temporal/polyfill";

declare global {
	interface Date {
		toTemporalInstant(this: Date): Temporal.Instant;
	}
}
