import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
export default defineConfig({
  plugins: [svelte()],
  build: {
    assetsInlineLimit: 0,
    rollupOptions: {
      input: { studio: "index.html", renderer: "renderer.html" },
    },
  },
  server: { host: "0.0.0.0", fs: { allow: [".."] } },
});
