import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { CalendarService } from "../../api/v1/api_pb";

const transport = createConnectTransport({
	baseUrl: "http://127.0.0.1:8003",
});

export const client = createClient(CalendarService, transport);
