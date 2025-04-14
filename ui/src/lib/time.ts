import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Temporal } from "@js-temporal/polyfill";
import { ZonedDateTime as I18nZonedDateTime } from "@internationalized/date";

export function instantToTimestamp(instant: Temporal.Instant): Timestamp {
	return {
		$typeName: "google.protobuf.Timestamp",
		seconds: BigInt(Math.floor(instant.epochMilliseconds / 1000)),
		nanos: 0,
	};
}

export function zonedToI18n(datetime: Temporal.ZonedDateTime): I18nZonedDateTime {
	return new I18nZonedDateTime(
		datetime.year,
		datetime.month,
		datetime.day,
		datetime.timeZoneId,
		datetime.offsetNanoseconds / 1_000_000,
		datetime.hour,
		datetime.minute,
		datetime.second,
		datetime.millisecond,
	)
}
