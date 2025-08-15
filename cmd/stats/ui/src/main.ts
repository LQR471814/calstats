import { mount } from "svelte";
import "./app.css";
import { toTemporalInstant } from "@js-temporal/polyfill";
import App from "./App.svelte";

Date.prototype.toTemporalInstant = toTemporalInstant;

const app = mount(App, {
	target: document.getElementById("app")!,
});

export default app;
