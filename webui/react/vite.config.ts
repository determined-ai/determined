import crypto from 'crypto';
import fs from 'fs';
import path from 'path';

import react from '@vitejs/plugin-react-swc';
import { defineConfig, Plugin, UserConfig } from 'vite';
import tsconfigPaths from 'vite-tsconfig-paths';
import checker from 'vite-plugin-checker';

import { cspHtml } from './src/shared/configs/vite-plugin-csp';

// want to fallback in case of empty string, hence no ??
const webpackProxyUrl = process.env.DET_WEBPACK_PROXY_URL || 'http://localhost:8080';

const publicUrlBaseHref = (): Plugin => {
  let config: UserConfig;
  return {
    config(c) {
      config = c;
    },
    name: 'public-url-base-href',
    transformIndexHtml: {
      handler() {
        return config.base
          ? [
              {
                attrs: {
                  href: config.base,
                },
                tag: 'meta',
              },
            ]
          : [];
      },
    },
  };
};

// public_url as / breaks the link component -- assuming that CRA did something
// to prevent that, idk
const publicUrl = (process.env.PUBLIC_URL || '') === '/' ? undefined : process.env.PUBLIC_URL;

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  base: publicUrl,
  build: {
    commonjsOptions: {
      include: [/node_modules/, /notebook/],
    },
    outDir: 'build',
    sourcemap: mode === 'production',
  },
  css: {
    modules: {
      generateScopedName: (name, filename) => {
        const basename = path.basename(filename).split('.')[0];
        const hashable = `${basename}_${name}`;
        const hash = crypto.createHash('sha256').update(filename).digest('hex').substring(0, 5);

        return `${hashable}_${hash}`;
      },
    },
    preprocessorOptions: {
      scss: {
        additionalData: fs.readFileSync('./src/shared/styles/global.scss'),
      },
    },
  },
  define: {
    'process.env.IS_DEV': JSON.stringify(mode === 'development'),
    'process.env.PUBLIC_URL': JSON.stringify(publicUrl || ''),
    'process.env.SERVER_ADDRESS': JSON.stringify(process.env.SERVER_ADDRESS),
    'process.env.VERSION': '"0.19.11-dev0"',
  },
  optimizeDeps: {
    include: ['notebook'],
  },
  plugins: [
    tsconfigPaths(),
    react(),
    publicUrlBaseHref(),
    checker({
      typescript: true,
    }),
    cspHtml({
      cspRules: {
        'frame-src': ["'self'", 'netlify.determined.ai'],
        'object-src': ["'none'"],
        'script-src': ["'self'", 'cdn.segment.com'],
        'style-src': ["'self'", "'unsafe-inline'"],
      },
      hashEnabled: {
        'script-src': true,
        'style-src': false,
      },
    }),
  ],
  resolve: {
    alias: {
      // needed for react-dnd
      'react/jsx-runtime.js': 'react/jsx-runtime',
    },
  },
  server: {
    open: true,
    port: 3000,
    proxy: {
      '/api': { target: webpackProxyUrl },
      '/proxy': { target: webpackProxyUrl },
    },
    strictPort: true,
  },
}));
