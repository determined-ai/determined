import fs from 'fs';
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tsconfigPaths from 'vite-tsconfig-paths'

// https://vitejs.dev/config/
export default defineConfig({
  css: {
    preprocessorOptions: {
      scss: {
        additionalData: fs.readFileSync('./src/shared/styles/global.scss'),
      },
    }
  },
  define: {
    'process.env.IS_DEV': JSON.stringify(process.env.DET_NODE_ENV === 'development'),
    'process.env.SERVER_ADDRESS': JSON.stringify(process.env.SERVER_ADDRESS),
    'process.env.VERSION': '"0.19.11-dev0"',
  },
  plugins: [tsconfigPaths(), react()],
  resolve: {
    alias: [
      {
        find: 'react/jsx-runtime.js',
        replacement: 'react/jsx-runtime'
      }
    ]
  },
})
