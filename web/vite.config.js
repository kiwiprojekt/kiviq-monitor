import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  // Relative base so the built asset URLs resolve under a reverse-proxy
  // sub-path (Home Assistant ingress) as well as at the site root.
  base: './',
  plugins: [vue(), tailwindcss()],
  test: {
    environment: 'jsdom',
  },
})
