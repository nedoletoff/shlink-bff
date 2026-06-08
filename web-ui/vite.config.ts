import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { readFileSync } from 'fs';

const pkg = JSON.parse(readFileSync('./package.json', 'utf-8'));

export default defineConfig({
  plugins: [react()],
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: (id: string) => {
          if (id.includes('node_modules')) {
            if (['react', 'react-dom', 'react-router-dom'].some((p) => id.includes(p))) return 'vendor';
            if (['@mantine/core', '@mantine/hooks', '@mantine/notifications'].some((p) => id.includes(p))) return 'mantine';
            if (id.includes('recharts')) return 'charts';
          }
        },
      },
    },
  },
});
