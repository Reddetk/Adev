import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// The dev server proxies the API to the Go server. The browser then sees one
// origin, so the API needs no CORS and no auth token plumbing. The build is
// relative, so `make run` can serve it on /app/ without an asset prefix.
export default defineConfig({
  base: './',
  plugins: [react()],
  server: {
    proxy: {
      '/api': { target: 'http://127.0.0.1:8080', changeOrigin: true },
    },
  },
});
