import path from "path"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    host: '0.0.0.0',
    allowedHosts: [ 'wild-central', 'wild-central.lan'],
    proxy: {
      '/api': {
        target: 'http://wild-central:5055',
        changeOrigin: true,
        secure: false,
        // Important for SSE to work
        configure: (proxy, _options) => {
          proxy.on('error', (err, _req, _res) => {
            console.log('proxy error', err);
          });
          proxy.on('proxyReq', (_proxyReq, req, _res) => {
            console.log('Proxying:', req.method, req.url, '->', 'http://wild-central:5055' + req.url);
          });
        },
      },
    },
  }
})
