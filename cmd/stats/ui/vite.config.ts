import path from "node:path";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
	plugins: [
		tailwindcss(),
		svelte(),
	],
	resolve: {
		alias: {
			$lib: path.resolve("./src/lib"),
			$api: path.resolve("./src/lib/api/v1"),
		},
	},
});
