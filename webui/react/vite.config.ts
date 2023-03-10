import crypto from 'crypto';
import fs from 'fs';
import path from 'path';

import react from '@vitejs/plugin-react-swc';
import MagicString from 'magic-string';
import { defineConfig, Plugin, UserConfig } from 'vite';
import checker from 'vite-plugin-checker';
import tsconfigPaths from 'vite-tsconfig-paths';

import { cspHtml } from './src/shared/configs/vite-plugin-csp';

// want to fallback in case of empty string, hence no ??
const webpackProxyUrl = process.env.DET_WEBPACK_PROXY_URL || 'http://localhost:8080';

// https://github.com/swagger-api/swagger-codegen/issues/10027
const portableFetchFix = () => ({
  name: 'fix-portable-fetch',
  transform: (source: string, id: string) => {
    if (id.endsWith('api-ts-sdk/api.ts')) {
      const newSource = new MagicString(
        source.replace(
          'import * as portableFetch from "portable-fetch"',
          'import portableFetch from "portable-fetch"',
        ),
      );
      return {
        code: newSource.toString(),
        map: newSource.generateMap(),
      };
    }
  },
});

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
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          if (id.includes('node_modules')) {
            return 'vendor';
          }
        },
      },
    },
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
    'process.env.VERSION': '"0.20.1-dev0"',
  },
  optimizeDeps: {
    include: ['notebook'],
  },
  plugins: [
    tsconfigPaths(),
    react(),
    portableFetchFix(),
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
