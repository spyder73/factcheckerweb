import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// API client builds absolute URLs from VITE_API_URL (defaults to
// http://localhost:8080). No dev proxy needed — having a proxy alongside
// the absolute-URL strategy creates confusion about which path the
// browser actually hits.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
  },
})
