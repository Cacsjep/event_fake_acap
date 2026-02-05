import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'

export default defineConfig({
  plugins: [vue(), vuetify({ autoImport: true })],
  build: {
    outDir: '../eva/html',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://10.0.0.48:8746',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
