import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const backendUrl = process.env.VITE_API_PROXY || 'http://localhost:8080'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      // xfwd passes on the host the browser used, which login needs for
      // the address the provider sends people back to.
      '/api': { target: backendUrl, changeOrigin: true, xfwd: true },
      '/ws': { target: backendUrl.replace('http', 'ws'), ws: true }
    }
  }
})
