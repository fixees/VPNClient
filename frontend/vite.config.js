import { defineConfig } from 'vite'

// Relative asset paths are required for Wails embed asset server.
export default defineConfig({
  base: './',
  clearScreen: false,
  server: {
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
