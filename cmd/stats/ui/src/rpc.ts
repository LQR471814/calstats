import { CalendarService } from "$api/api_pb";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

const transport = createConnectTransport({
	baseUrl: import.meta.env.PROD ? window.origin : "http://localhost:3000",
});

export const client = createClient(CalendarService, transport);
