// vite.config.mts
import crypto2 from 'crypto';
import fs from 'fs';
import path from 'path';
import { svgToReact } from 'file:///Users/eliu/Determined/determined/webui/react/node_modules/@hpe.com/vite-plugin-svg-to-jsx/build/out.js';
import react from 'file:///Users/eliu/Determined/determined/webui/react/node_modules/@vitejs/plugin-react-swc/index.mjs';
import checker from 'file:///Users/eliu/Determined/determined/webui/react/node_modules/vite-plugin-checker/dist/esm/main.js';
import tsconfigPaths from 'file:///Users/eliu/Determined/determined/webui/react/node_modules/vite-tsconfig-paths/dist/index.mjs';
import {
  configDefaults,
  defineConfig,
} from 'file:///Users/eliu/Determined/determined/webui/react/node_modules/vitest/dist/config.js';

// vite-plugin-csp.ts
import crypto from 'crypto';
var cspHtml = ({ cspRules, hashEnabled = {} }) => ({
  name: 'csp-html',
  transformIndexHtml: {
    async handler(html) {
      const finalCspRules = {
        'base-uri': ["'self'"],
        ...cspRules,
      };
      const hashRules = Object.entries(hashEnabled);
      if (hashRules.length) {
        const cheerio = await import(
          'file:///Users/eliu/Determined/determined/webui/react/node_modules/cheerio/lib/esm/index.js'
        );
        const $ = cheerio.load(html);
        hashRules.forEach(([directive, enabled]) => {
          if (!enabled) return;
          const [tag] = directive.split('-');
          $(tag).each((_, el) => {
            const source = $(el).html();
            if (source) {
              const hash = crypto.createHash('sha256').update(source).digest('base64');
              finalCspRules[directive] = (finalCspRules[directive] || []).concat([
                `'sha256-${hash}'`,
              ]);
            }
          });
        });
      }
      const content = Object.entries(finalCspRules)
        .map(([directive, sources]) => `${directive} ${sources.join(' ')}`)
        .join('; ');
      return [
        {
          attrs: {
            content,
            'http-equiv': 'Content-Security-Policy',
          },
          tag: 'meta',
        },
      ];
    },
    order: 'post',
  },
});

// vite-plugin-branding.ts
var brandHtml = () => {
  return {
    name: 'brandHtml',
    async transformIndexHtml(html) {
      if (process.env.DET_BUILD_EE === 'true') {
        const cheerio = await import(
          'file:///Users/eliu/Determined/determined/webui/react/node_modules/cheerio/lib/esm/index.js'
        );
        const $ = cheerio.load(html);
        $('meta[name="description"]').each(function () {
          $(this).attr('content', 'HPE Machine Learning Development Environment');
        });
        return $.html();
      }
      return html;
    },
  };
};

// vite.config.mts
var __vite_injected_original_dirname = '/Users/eliu/Determined/determined/webui/react';
var webpackProxyUrl = process.env.DET_WEBPACK_PROXY_URL || 'http://localhost:8080';
var websocketProxyUrl = process.env.DET_WEBSOCKET_PROXY_URL || 'ws://localhost:8080';
var publicUrlBaseHref = () => {
  let config;
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
var publicUrl = (process.env.PUBLIC_URL || '') === '/' ? void 0 : process.env.PUBLIC_URL;
var vite_config_default = defineConfig(({ mode }) => ({
  base: publicUrl,
  build: {
    commonjsOptions: {
      include: [/node_modules/, /notebook/],
    },
    outDir: 'build',
    rollupOptions: {
      input: {
        main: path.resolve(__vite_injected_original_dirname, 'index.html'),
      },
      output: {
        manualChunks: (id) => {
          if (id.includes('node_modules')) {
            return 'vendor';
          }
          if (id.endsWith('.svg')) {
            return 'icons';
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
        const hash = crypto2.createHash('sha256').update(filename).digest('hex').substring(0, 5);
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
    'process.env.VERSION': '"0.31.0-dev0"',
  },
  optimizeDeps: {
    include: ['notebook'],
  },
  plugins: [
    tsconfigPaths(),
    svgToReact({
      plugins: [
        {
          name: 'preset-default',
          params: {
            overrides: {
              convertColors: {
                currentColor: '#000',
              },
              removeViewBox: false,
            },
          },
        },
      ],
    }),
    react(),
    publicUrlBaseHref(),
    brandHtml(),
    mode !== 'test' &&
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
  preview: {
    port: 3001,
    proxy: {
      '/api': { target: webpackProxyUrl },
      '/proxy': { target: webpackProxyUrl },
      '/stream': {
        target: websocketProxyUrl,
        ws: true,
      },
    },
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
    port: 3e3,
    proxy: {
      '/api': { target: webpackProxyUrl },
      '/proxy': { target: webpackProxyUrl },
      '/stream': {
        target: websocketProxyUrl,
        ws: true,
      },
    },
    strictPort: true,
  },
  test: {
    coverage: {
      ...configDefaults.coverage,
      include: ['src'],
      exclude: [
        ...(configDefaults.coverage.exclude ?? []),
        'src/vendor/**/*',
        'src/services/api-ts-sdk/*',
      ],
    },
    css: {
      modules: {
        classNameStrategy: 'non-scoped',
      },
    },
    deps: {
      // resolve css imports
      inline: [/hew/],
      // necessary to fix react-dnd jsx runtime issue
      registerNodeLoader: true,
    },
    environment: 'jsdom',
    exclude: [...configDefaults.exclude, './src/e2e/**/*'],
    globals: true,
    setupFiles: ['./src/setupTests.ts'],
    testNamePattern: process.env.INCLUDE_FLAKY === 'true' ? /@flaky/ : /^(?!.*@flaky)/,
  },
}));
export { vite_config_default as default };
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcubXRzIiwgInZpdGUtcGx1Z2luLWNzcC50cyIsICJ2aXRlLXBsdWdpbi1icmFuZGluZy50cyJdLAogICJzb3VyY2VzQ29udGVudCI6IFsiY29uc3QgX192aXRlX2luamVjdGVkX29yaWdpbmFsX2Rpcm5hbWUgPSBcIi9Vc2Vycy9lbGl1L0RldGVybWluZWQvZGV0ZXJtaW5lZC93ZWJ1aS9yZWFjdFwiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9maWxlbmFtZSA9IFwiL1VzZXJzL2VsaXUvRGV0ZXJtaW5lZC9kZXRlcm1pbmVkL3dlYnVpL3JlYWN0L3ZpdGUuY29uZmlnLm10c1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vVXNlcnMvZWxpdS9EZXRlcm1pbmVkL2RldGVybWluZWQvd2VidWkvcmVhY3Qvdml0ZS5jb25maWcubXRzXCI7aW1wb3J0IGNyeXB0byBmcm9tICdjcnlwdG8nO1xuaW1wb3J0IGZzIGZyb20gJ2ZzJztcbmltcG9ydCBwYXRoIGZyb20gJ3BhdGgnO1xuXG5pbXBvcnQgeyBzdmdUb1JlYWN0IH0gZnJvbSAnQGhwZS5jb20vdml0ZS1wbHVnaW4tc3ZnLXRvLWpzeCc7XG5pbXBvcnQgcmVhY3QgZnJvbSAnQHZpdGVqcy9wbHVnaW4tcmVhY3Qtc3djJztcbmltcG9ydCB7IFBsdWdpbiwgVXNlckNvbmZpZyB9IGZyb20gJ3ZpdGUnO1xuaW1wb3J0IGNoZWNrZXIgZnJvbSAndml0ZS1wbHVnaW4tY2hlY2tlcic7XG5pbXBvcnQgdHNjb25maWdQYXRocyBmcm9tICd2aXRlLXRzY29uZmlnLXBhdGhzJztcbmltcG9ydCB7IGNvbmZpZ0RlZmF1bHRzLCBkZWZpbmVDb25maWcgfSBmcm9tICd2aXRlc3QvY29uZmlnJztcblxuaW1wb3J0IHsgY3NwSHRtbCB9IGZyb20gJy4vdml0ZS1wbHVnaW4tY3NwJztcbmltcG9ydCB7IGJyYW5kSHRtbCB9IGZyb20gXCIuL3ZpdGUtcGx1Z2luLWJyYW5kaW5nXCI7XG5cbi8vIHdhbnQgdG8gZmFsbGJhY2sgaW4gY2FzZSBvZiBlbXB0eSBzdHJpbmcsIGhlbmNlIG5vID8/XG5jb25zdCB3ZWJwYWNrUHJveHlVcmwgPSBwcm9jZXNzLmVudi5ERVRfV0VCUEFDS19QUk9YWV9VUkwgfHwgJ2h0dHA6Ly9sb2NhbGhvc3Q6ODA4MCc7XG5jb25zdCB3ZWJzb2NrZXRQcm94eVVybCA9IHByb2Nlc3MuZW52LkRFVF9XRUJTT0NLRVRfUFJPWFlfVVJMIHx8ICd3czovL2xvY2FsaG9zdDo4MDgwJztcblxuY29uc3QgcHVibGljVXJsQmFzZUhyZWYgPSAoKTogUGx1Z2luID0+IHtcbiAgbGV0IGNvbmZpZzogVXNlckNvbmZpZztcbiAgcmV0dXJuIHtcbiAgICBjb25maWcoYykge1xuICAgICAgY29uZmlnID0gYztcbiAgICB9LFxuICAgIG5hbWU6ICdwdWJsaWMtdXJsLWJhc2UtaHJlZicsXG4gICAgdHJhbnNmb3JtSW5kZXhIdG1sOiB7XG4gICAgICBoYW5kbGVyKCkge1xuICAgICAgICByZXR1cm4gY29uZmlnLmJhc2VcbiAgICAgICAgICA/IFtcbiAgICAgICAgICAgICAge1xuICAgICAgICAgICAgICAgIGF0dHJzOiB7XG4gICAgICAgICAgICAgICAgICBocmVmOiBjb25maWcuYmFzZSxcbiAgICAgICAgICAgICAgICB9LFxuICAgICAgICAgICAgICAgIHRhZzogJ21ldGEnLFxuICAgICAgICAgICAgICB9LFxuICAgICAgICAgICAgXVxuICAgICAgICAgIDogW107XG4gICAgICB9LFxuICAgIH0sXG4gIH07XG59O1xuXG4vLyBwdWJsaWNfdXJsIGFzIC8gYnJlYWtzIHRoZSBsaW5rIGNvbXBvbmVudCAtLSBhc3N1bWluZyB0aGF0IENSQSBkaWQgc29tZXRoaW5nXG4vLyB0byBwcmV2ZW50IHRoYXQsIGlka1xuY29uc3QgcHVibGljVXJsID0gKHByb2Nlc3MuZW52LlBVQkxJQ19VUkwgfHwgJycpID09PSAnLycgPyB1bmRlZmluZWQgOiBwcm9jZXNzLmVudi5QVUJMSUNfVVJMO1xuXG4vLyBodHRwczovL3ZpdGVqcy5kZXYvY29uZmlnL1xuZXhwb3J0IGRlZmF1bHQgZGVmaW5lQ29uZmlnKCh7IG1vZGUgfSkgPT4gKHtcbiAgYmFzZTogcHVibGljVXJsLFxuICBidWlsZDoge1xuICAgIGNvbW1vbmpzT3B0aW9uczoge1xuICAgICAgaW5jbHVkZTogWy9ub2RlX21vZHVsZXMvLCAvbm90ZWJvb2svXSxcbiAgICB9LFxuICAgIG91dERpcjogJ2J1aWxkJyxcbiAgICByb2xsdXBPcHRpb25zOiB7XG4gICAgICBpbnB1dDoge1xuICAgICAgICBtYWluOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnaW5kZXguaHRtbCcpLFxuICAgICAgfSxcbiAgICAgIG91dHB1dDoge1xuICAgICAgICBtYW51YWxDaHVua3M6IChpZCkgPT4ge1xuICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnbm9kZV9tb2R1bGVzJykpIHtcbiAgICAgICAgICAgIHJldHVybiAndmVuZG9yJztcbiAgICAgICAgICB9XG4gICAgICAgICAgaWYgKGlkLmVuZHNXaXRoKCcuc3ZnJykpIHtcbiAgICAgICAgICAgIHJldHVybiAnaWNvbnMnO1xuICAgICAgICAgIH1cbiAgICAgICAgfSxcbiAgICAgIH0sXG4gICAgfSxcbiAgICBzb3VyY2VtYXA6IG1vZGUgPT09ICdwcm9kdWN0aW9uJyxcbiAgfSxcbiAgY3NzOiB7XG4gICAgbW9kdWxlczoge1xuICAgICAgZ2VuZXJhdGVTY29wZWROYW1lOiAobmFtZSwgZmlsZW5hbWUpID0+IHtcbiAgICAgICAgY29uc3QgYmFzZW5hbWUgPSBwYXRoLmJhc2VuYW1lKGZpbGVuYW1lKS5zcGxpdCgnLicpWzBdO1xuICAgICAgICBjb25zdCBoYXNoYWJsZSA9IGAke2Jhc2VuYW1lfV8ke25hbWV9YDtcbiAgICAgICAgY29uc3QgaGFzaCA9IGNyeXB0by5jcmVhdGVIYXNoKCdzaGEyNTYnKS51cGRhdGUoZmlsZW5hbWUpLmRpZ2VzdCgnaGV4Jykuc3Vic3RyaW5nKDAsIDUpO1xuXG4gICAgICAgIHJldHVybiBgJHtoYXNoYWJsZX1fJHtoYXNofWA7XG4gICAgICB9LFxuICAgIH0sXG4gICAgcHJlcHJvY2Vzc29yT3B0aW9uczoge1xuICAgICAgc2Nzczoge1xuICAgICAgICBhZGRpdGlvbmFsRGF0YTogZnMucmVhZEZpbGVTeW5jKCcuL3NyYy9zdHlsZXMvZ2xvYmFsLnNjc3MnKSxcbiAgICAgIH0sXG4gICAgfSxcbiAgfSxcbiAgZGVmaW5lOiB7XG4gICAgJ3Byb2Nlc3MuZW52LklTX0RFVic6IEpTT04uc3RyaW5naWZ5KG1vZGUgPT09ICdkZXZlbG9wbWVudCcpLFxuICAgICdwcm9jZXNzLmVudi5QVUJMSUNfVVJMJzogSlNPTi5zdHJpbmdpZnkoKG1vZGUgIT09ICd0ZXN0JyAmJiBwdWJsaWNVcmwpIHx8ICcnKSxcbiAgICAncHJvY2Vzcy5lbnYuU0VSVkVSX0FERFJFU1MnOiBKU09OLnN0cmluZ2lmeShwcm9jZXNzLmVudi5TRVJWRVJfQUREUkVTUyksXG4gICAgJ3Byb2Nlc3MuZW52LlZFUlNJT04nOiAnXCIwLjMxLjAtZGV2MFwiJyxcbiAgfSxcbiAgb3B0aW1pemVEZXBzOiB7XG4gICAgaW5jbHVkZTogWydub3RlYm9vayddLFxuICB9LFxuICBwbHVnaW5zOiBbXG4gICAgdHNjb25maWdQYXRocygpLFxuICAgIHN2Z1RvUmVhY3Qoe1xuICAgICAgcGx1Z2luczogW1xuICAgICAgICB7XG4gICAgICAgICAgbmFtZTogJ3ByZXNldC1kZWZhdWx0JyxcbiAgICAgICAgICBwYXJhbXM6IHtcbiAgICAgICAgICAgIG92ZXJyaWRlczoge1xuICAgICAgICAgICAgICBjb252ZXJ0Q29sb3JzOiB7XG4gICAgICAgICAgICAgICAgY3VycmVudENvbG9yOiAnIzAwMCcsXG4gICAgICAgICAgICAgIH0sXG4gICAgICAgICAgICAgIHJlbW92ZVZpZXdCb3g6IGZhbHNlLFxuICAgICAgICAgICAgfSxcbiAgICAgICAgICB9LFxuICAgICAgICB9LFxuICAgICAgXSxcbiAgICB9KSxcbiAgICByZWFjdCgpLFxuICAgIHB1YmxpY1VybEJhc2VIcmVmKCksXG4gICAgYnJhbmRIdG1sKCksXG4gICAgbW9kZSAhPT0gJ3Rlc3QnICYmXG4gICAgICBjaGVja2VyKHtcbiAgICAgICAgdHlwZXNjcmlwdDogdHJ1ZSxcbiAgICAgIH0pLFxuICAgIGNzcEh0bWwoe1xuICAgICAgY3NwUnVsZXM6IHtcbiAgICAgICAgJ2ZyYW1lLXNyYyc6IFtcIidzZWxmJ1wiLCAnbmV0bGlmeS5kZXRlcm1pbmVkLmFpJ10sXG4gICAgICAgICdvYmplY3Qtc3JjJzogW1wiJ25vbmUnXCJdLFxuICAgICAgICAnc2NyaXB0LXNyYyc6IFtcIidzZWxmJ1wiLCAnY2RuLnNlZ21lbnQuY29tJ10sXG4gICAgICAgICdzdHlsZS1zcmMnOiBbXCInc2VsZidcIiwgXCIndW5zYWZlLWlubGluZSdcIl0sXG4gICAgICB9LFxuICAgICAgaGFzaEVuYWJsZWQ6IHtcbiAgICAgICAgJ3NjcmlwdC1zcmMnOiB0cnVlLFxuICAgICAgICAnc3R5bGUtc3JjJzogZmFsc2UsXG4gICAgICB9LFxuICAgIH0pLFxuICBdLFxuICBwcmV2aWV3OiB7XG4gICAgcG9ydDogMzAwMSxcbiAgICBwcm94eToge1xuICAgICAgJy9hcGknOiB7IHRhcmdldDogd2VicGFja1Byb3h5VXJsIH0sXG4gICAgICAnL3Byb3h5JzogeyB0YXJnZXQ6IHdlYnBhY2tQcm94eVVybCB9LFxuICAgICAgJy9zdHJlYW0nOiB7XG4gICAgICAgIHRhcmdldDogd2Vic29ja2V0UHJveHlVcmwsXG4gICAgICAgIHdzOiB0cnVlLFxuICAgICAgfSxcbiAgICB9LFxuICAgIHN0cmljdFBvcnQ6IHRydWUsXG4gIH0sXG4gIHJlc29sdmU6IHtcbiAgICBhbGlhczoge1xuICAgICAgLy8gbmVlZGVkIGZvciByZWFjdC1kbmRcbiAgICAgICdyZWFjdC9qc3gtcnVudGltZS5qcyc6ICdyZWFjdC9qc3gtcnVudGltZScsXG4gICAgfSxcbiAgfSxcbiAgc2VydmVyOiB7XG4gICAgb3BlbjogdHJ1ZSxcbiAgICBwb3J0OiAzMDAwLFxuICAgIHByb3h5OiB7XG4gICAgICAnL2FwaSc6IHsgdGFyZ2V0OiB3ZWJwYWNrUHJveHlVcmwgfSxcbiAgICAgICcvcHJveHknOiB7IHRhcmdldDogd2VicGFja1Byb3h5VXJsIH0sXG4gICAgICAnL3N0cmVhbSc6IHtcbiAgICAgICAgdGFyZ2V0OiB3ZWJzb2NrZXRQcm94eVVybCxcbiAgICAgICAgd3M6IHRydWUsXG4gICAgICB9LFxuICAgIH0sXG4gICAgc3RyaWN0UG9ydDogdHJ1ZSxcbiAgfSxcbiAgdGVzdDoge1xuICAgIGNvdmVyYWdlOiB7XG4gICAgICAuLi5jb25maWdEZWZhdWx0cy5jb3ZlcmFnZSxcbiAgICAgIGluY2x1ZGU6IFsnc3JjJ10sXG4gICAgICBleGNsdWRlOiBbXG4gICAgICAgIC4uLihjb25maWdEZWZhdWx0cy5jb3ZlcmFnZS5leGNsdWRlID8/IFtdKSxcbiAgICAgICAgJ3NyYy92ZW5kb3IvKiovKicsXG4gICAgICAgICdzcmMvc2VydmljZXMvYXBpLXRzLXNkay8qJyxcbiAgICAgIF0sXG4gICAgfSxcbiAgICBjc3M6IHtcbiAgICAgIG1vZHVsZXM6IHtcbiAgICAgICAgY2xhc3NOYW1lU3RyYXRlZ3k6ICdub24tc2NvcGVkJyxcbiAgICAgIH0sXG4gICAgfSxcbiAgICBkZXBzOiB7XG4gICAgICAvLyByZXNvbHZlIGNzcyBpbXBvcnRzXG4gICAgICBpbmxpbmU6IFsvaGV3L10sXG5cbiAgICAgIC8vIG5lY2Vzc2FyeSB0byBmaXggcmVhY3QtZG5kIGpzeCBydW50aW1lIGlzc3VlXG4gICAgICByZWdpc3Rlck5vZGVMb2FkZXI6IHRydWUsXG4gICAgfSxcbiAgICBlbnZpcm9ubWVudDogJ2pzZG9tJyxcbiAgICBleGNsdWRlOiBbLi4uY29uZmlnRGVmYXVsdHMuZXhjbHVkZSwgJy4vc3JjL2UyZS8qKi8qJ10sXG4gICAgZ2xvYmFsczogdHJ1ZSxcbiAgICBzZXR1cEZpbGVzOiBbJy4vc3JjL3NldHVwVGVzdHMudHMnXSxcbiAgICB0ZXN0TmFtZVBhdHRlcm46IHByb2Nlc3MuZW52LklOQ0xVREVfRkxBS1kgPT09ICd0cnVlJyA/IC9AZmxha3kvIDogL14oPyEuKkBmbGFreSkvLFxuICB9LFxufSkpO1xuIiwgImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvVXNlcnMvZWxpdS9EZXRlcm1pbmVkL2RldGVybWluZWQvd2VidWkvcmVhY3RcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIi9Vc2Vycy9lbGl1L0RldGVybWluZWQvZGV0ZXJtaW5lZC93ZWJ1aS9yZWFjdC92aXRlLXBsdWdpbi1jc3AudHNcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfaW1wb3J0X21ldGFfdXJsID0gXCJmaWxlOi8vL1VzZXJzL2VsaXUvRGV0ZXJtaW5lZC9kZXRlcm1pbmVkL3dlYnVpL3JlYWN0L3ZpdGUtcGx1Z2luLWNzcC50c1wiO2ltcG9ydCBjcnlwdG8gZnJvbSAnY3J5cHRvJztcblxuaW1wb3J0IHR5cGUgeyBQbHVnaW4gfSBmcm9tICd2aXRlJztcblxuLy8gaW5jb21wbGV0ZSBsaXN0IG9mIGRpcmVjdGl2ZXNcbnR5cGUgQ3NwSGFzaERpcmVjdGl2ZSA9ICdzY3JpcHQtc3JjJyB8ICdzdHlsZS1zcmMnO1xudHlwZSBDc3BEaXJlY3RpdmUgPSAnYmFzZS11cmknIHwgJ2ZyYW1lLXNyYycgfCAnb2JqZWN0LXNyYycgfCBDc3BIYXNoRGlyZWN0aXZlO1xuXG50eXBlIENzcFJ1bGVDb25maWcgPSB7XG4gIFtrZXkgaW4gQ3NwRGlyZWN0aXZlXT86IHN0cmluZ1tdO1xufTtcblxudHlwZSBDc3BIYXNoQ29uZmlnID0ge1xuICBba2V5IGluIENzcEhhc2hEaXJlY3RpdmVdPzogYm9vbGVhbjtcbn07XG5cbmludGVyZmFjZSBDc3BIdG1sUGx1Z2luQ29uZmlnIHtcbiAgY3NwUnVsZXM6IENzcFJ1bGVDb25maWc7XG4gIGhhc2hFbmFibGVkOiBDc3BIYXNoQ29uZmlnO1xufVxuXG5leHBvcnQgY29uc3QgY3NwSHRtbCA9ICh7IGNzcFJ1bGVzLCBoYXNoRW5hYmxlZCA9IHt9IH06IENzcEh0bWxQbHVnaW5Db25maWcpOiBQbHVnaW4gPT4gKHtcbiAgbmFtZTogJ2NzcC1odG1sJyxcbiAgdHJhbnNmb3JtSW5kZXhIdG1sOiB7XG4gICAgYXN5bmMgaGFuZGxlcihodG1sOiBzdHJpbmcpIHtcbiAgICAgIGNvbnN0IGZpbmFsQ3NwUnVsZXM6IENzcFJ1bGVDb25maWcgPSB7XG4gICAgICAgICdiYXNlLXVyaSc6IFtcIidzZWxmJ1wiXSxcbiAgICAgICAgLi4uY3NwUnVsZXMsXG4gICAgICB9O1xuICAgICAgY29uc3QgaGFzaFJ1bGVzID0gT2JqZWN0LmVudHJpZXMoaGFzaEVuYWJsZWQpIGFzIFtDc3BIYXNoRGlyZWN0aXZlLCBib29sZWFuXVtdO1xuICAgICAgaWYgKGhhc2hSdWxlcy5sZW5ndGgpIHtcbiAgICAgICAgY29uc3QgY2hlZXJpbyA9IGF3YWl0IGltcG9ydCgnY2hlZXJpbycpO1xuICAgICAgICBjb25zdCAkID0gY2hlZXJpby5sb2FkKGh0bWwpO1xuICAgICAgICBoYXNoUnVsZXMuZm9yRWFjaCgoW2RpcmVjdGl2ZSwgZW5hYmxlZF06IFtDc3BIYXNoRGlyZWN0aXZlLCBib29sZWFuXSkgPT4ge1xuICAgICAgICAgIGlmICghZW5hYmxlZCkgcmV0dXJuO1xuICAgICAgICAgIGNvbnN0IFt0YWddID0gZGlyZWN0aXZlLnNwbGl0KCctJyk7XG4gICAgICAgICAgJCh0YWcpLmVhY2goKF8sIGVsKSA9PiB7XG4gICAgICAgICAgICBjb25zdCBzb3VyY2UgPSAkKGVsKS5odG1sKCk7XG4gICAgICAgICAgICBpZiAoc291cmNlKSB7XG4gICAgICAgICAgICAgIGNvbnN0IGhhc2ggPSBjcnlwdG8uY3JlYXRlSGFzaCgnc2hhMjU2JykudXBkYXRlKHNvdXJjZSkuZGlnZXN0KCdiYXNlNjQnKTtcbiAgICAgICAgICAgICAgZmluYWxDc3BSdWxlc1tkaXJlY3RpdmVdID0gKGZpbmFsQ3NwUnVsZXNbZGlyZWN0aXZlXSB8fCBbXSkuY29uY2F0KFtcbiAgICAgICAgICAgICAgICBgJ3NoYTI1Ni0ke2hhc2h9J2AsXG4gICAgICAgICAgICAgIF0pO1xuICAgICAgICAgICAgfVxuICAgICAgICAgIH0pO1xuICAgICAgICB9KTtcbiAgICAgIH1cbiAgICAgIGNvbnN0IGNvbnRlbnQgPSBPYmplY3QuZW50cmllcyhmaW5hbENzcFJ1bGVzKVxuICAgICAgICAubWFwKChbZGlyZWN0aXZlLCBzb3VyY2VzXSkgPT4gYCR7ZGlyZWN0aXZlfSAke3NvdXJjZXMuam9pbignICcpfWApXG4gICAgICAgIC5qb2luKCc7ICcpO1xuICAgICAgcmV0dXJuIFtcbiAgICAgICAge1xuICAgICAgICAgIGF0dHJzOiB7XG4gICAgICAgICAgICBjb250ZW50LFxuICAgICAgICAgICAgJ2h0dHAtZXF1aXYnOiAnQ29udGVudC1TZWN1cml0eS1Qb2xpY3knLFxuICAgICAgICAgIH0sXG4gICAgICAgICAgdGFnOiAnbWV0YScsXG4gICAgICAgIH0sXG4gICAgICBdO1xuICAgIH0sXG4gICAgb3JkZXI6ICdwb3N0JyxcbiAgfSxcbn0pO1xuIiwgImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvVXNlcnMvZWxpdS9EZXRlcm1pbmVkL2RldGVybWluZWQvd2VidWkvcmVhY3RcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIi9Vc2Vycy9lbGl1L0RldGVybWluZWQvZGV0ZXJtaW5lZC93ZWJ1aS9yZWFjdC92aXRlLXBsdWdpbi1icmFuZGluZy50c1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vVXNlcnMvZWxpdS9EZXRlcm1pbmVkL2RldGVybWluZWQvd2VidWkvcmVhY3Qvdml0ZS1wbHVnaW4tYnJhbmRpbmcudHNcIjtpbXBvcnQgdHlwZSB7IFBsdWdpbiB9IGZyb20gJ3ZpdGUnO1xuXG5leHBvcnQgY29uc3QgYnJhbmRIdG1sID0gKCk6IFBsdWdpbiA9PiB7XG4gIHJldHVybiB7XG4gICAgbmFtZTogJ2JyYW5kSHRtbCcsXG4gICAgYXN5bmMgdHJhbnNmb3JtSW5kZXhIdG1sKGh0bWw6IHN0cmluZykge1xuICAgICAgaWYgKHByb2Nlc3MuZW52LkRFVF9CVUlMRF9FRSA9PT0gJ3RydWUnKSB7XG4gICAgICAgIGNvbnN0IGNoZWVyaW8gPSBhd2FpdCBpbXBvcnQoJ2NoZWVyaW8nKTtcbiAgICAgICAgY29uc3QgJCA9IGNoZWVyaW8ubG9hZChodG1sKTtcbiAgICAgICAgJCgnbWV0YVtuYW1lPVwiZGVzY3JpcHRpb25cIl0nKS5lYWNoKGZ1bmN0aW9uICgpIHtcbiAgICAgICAgICAkKHRoaXMpLmF0dHIoJ2NvbnRlbnQnLCAnSFBFIE1hY2hpbmUgTGVhcm5pbmcgRGV2ZWxvcG1lbnQgRW52aXJvbm1lbnQnKTtcbiAgICAgICAgfSk7XG4gICAgICAgIHJldHVybiAkLmh0bWwoKTtcbiAgICAgIH1cbiAgICAgIHJldHVybiBodG1sO1xuICAgIH0sXG4gIH07XG59O1xuIl0sCiAgIm1hcHBpbmdzIjogIjtBQUEyVCxPQUFPQSxhQUFZO0FBQzlVLE9BQU8sUUFBUTtBQUNmLE9BQU8sVUFBVTtBQUVqQixTQUFTLGtCQUFrQjtBQUMzQixPQUFPLFdBQVc7QUFFbEIsT0FBTyxhQUFhO0FBQ3BCLE9BQU8sbUJBQW1CO0FBQzFCLFNBQVMsZ0JBQWdCLG9CQUFvQjs7O0FDVG9SLE9BQU8sWUFBWTtBQXFCN1UsSUFBTSxVQUFVLENBQUMsRUFBRSxVQUFVLGNBQWMsQ0FBQyxFQUFFLE9BQW9DO0FBQUEsRUFDdkYsTUFBTTtBQUFBLEVBQ04sb0JBQW9CO0FBQUEsSUFDbEIsTUFBTSxRQUFRLE1BQWM7QUFDMUIsWUFBTSxnQkFBK0I7QUFBQSxRQUNuQyxZQUFZLENBQUMsUUFBUTtBQUFBLFFBQ3JCLEdBQUc7QUFBQSxNQUNMO0FBQ0EsWUFBTSxZ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
