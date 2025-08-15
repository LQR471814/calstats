import { CalendarService } from "$api/api_pb";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

const transport = createConnectTransport({
	baseUrl: window.origin,
});

export const client = createClient(CalendarService, transport);
