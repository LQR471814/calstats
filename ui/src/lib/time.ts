import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Temporal } from "@js-temporal/polyfill";

export function instantToTimestamp(instant: Temporal.Instant): Timestamp {
	return {
		$typeName: "google.protobuf.Timestamp",
		seconds: BigInt(Math.floor(instant.epochMilliseconds / 1000)),
		nanos: 0,
	};
}
