import path from "node:path";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [svelte()],
	resolve: {
		alias: {
			$api: path.resolve("./src/lib/api/v1"),
		},
	},
});
