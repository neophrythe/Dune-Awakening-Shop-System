import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The Go service serves the API on the same origin in production. In dev, proxy
// /api and /healthz to the running backend (default :8091).
export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: {
    proxy: {
      '/api': 'http://localhost:8091',
      '/healthz': 'http://localhost:8091',
    },
  },
})
