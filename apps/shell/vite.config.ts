import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// In dev, the shell is served by Vite (:5173) and the osd daemon runs on :7700.
// In production, osd serves the built shell itself, so /api is same-origin.
export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:7700',
        changeOrigin: false,
        ws: true,
      },
    },
  },
  build: {
    target: 'chrome120',
    sourcemap: false,
  },
});
