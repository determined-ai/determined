import crypto from 'crypto';
import fs from 'fs';
import path from 'path';

import react from '@vitejs/plugin-react-swc';
import { Plugin, UserConfig } from 'vite';
import checker from 'vite-plugin-checker';
import tsconfigPaths from 'vite-tsconfig-paths';
import { configDefaults, defineConfig } from 'vitest/config';

import { cspHtml } from './vite-plugin-csp';

// want to fallback in case of empty string, hence no ??
const webpackProxyUrl = process.env.DET_WEBPACK_PROXY_URL || 'http://localhost:8080';

const devServerRedirects = (redirects: Record<string, string>): Plugin => {
  let config: UserConfig;
  return {
    config(c) {
      config = c;
    },
    configureServer(server) {
      Object.entries(redirects).forEach(([from, to]) => {
        const fromUrl = `${config.base || ''}${from}`;
        server.middlewares.use(fromUrl, (req, res, next) => {
          if (req.originalUrl === fromUrl) {
            res.writeHead(302, {
              Location: `${config.base || ''}${to}`,
            });
            res.end();
          } else {
            next();
          }
        });
      });
    },
    name: 'dev-server-redirects',
  };
};

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
      input: {
        design: path.resolve(__dirname, 'design', 'index.html'),
        main: path.resolve(__dirname, 'index.html'),
      },
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
        additionalData: fs.readFileSync('./src/styles/global.scss'),
      },
    },
  },
  define: {
    'process.env.IS_DEV': JSON.stringify(mode === 'development'),
    'process.env.PUBLIC_URL': JSON.stringify((mode !== 'test' && publicUrl) || ''),
    'process.env.SERVER_ADDRESS': JSON.stringify(process.env.SERVER_ADDRESS),
    'process.env.VERSION': '"0.23.3-rc0"',
  },
  optimizeDeps: {
    include: ['notebook'],
  },
  plugins: [
    tsconfigPaths(),
    react(),
    publicUrlBaseHref(),
    mode !== 'test' &&
      checker({
        typescript: true,
      }),
    devServerRedirects({
      '/design': '/design/',
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
  preview: {
    port: 3001,
    strictPort: true,
  },
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
  test: {
    css: {
      modules: {
        classNameStrategy: 'non-scoped',
      },
    },
    deps: {
      // necessary to fix react-dnd jsx runtime issue
      registerNodeLoader: true,
    },
    environment: 'jsdom',
    exclude: [...configDefaults.exclude, './src/e2e/*'],
    globals: true,
    setupFiles: ['./src/setupTests.ts'],
  },
}));
