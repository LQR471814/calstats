import { CalendarService } from "$api/api_pb";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

const transport = createConnectTransport({
	baseUrl: "http://127.0.0.1:8003",
});

export const client = createClient(CalendarService, transport);
