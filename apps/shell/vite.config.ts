import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// In dev, the shell is served by Vite (:5173) and the ghostd daemon runs on :7700.
// In production, ghostd serves the built shell itself, so /api is same-origin.
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
      // Installed .osapp packages are served by ghostd at /apps/<id>/; proxy
      // them in dev so the sandboxed iframe (ADR 0009) resolves to the daemon.
      '/apps': {
        target: 'http://127.0.0.1:7700',
        changeOrigin: false,
      },
    },
  },
  build: {
    target: 'chrome120',
    sourcemap: false,
  },
});
