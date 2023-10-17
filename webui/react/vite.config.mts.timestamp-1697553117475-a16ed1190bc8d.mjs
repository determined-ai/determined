// vite.config.mts
import crypto2 from "crypto";
import fs2 from "fs";
import path from "path";
import react from "file:///Users/julianglover/Software/determined/webui/react/node_modules/@vitejs/plugin-react-swc/index.mjs";
import checker from "file:///Users/julianglover/Software/determined/webui/react/node_modules/vite-plugin-checker/dist/esm/main.js";
import tsconfigPaths from "file:///Users/julianglover/Software/determined/webui/react/node_modules/vite-tsconfig-paths/dist/index.mjs";
import { configDefaults, defineConfig } from "file:///Users/julianglover/Software/determined/webui/react/node_modules/vitest/dist/config.js";

// vite-plugin-csp.ts
import crypto from "crypto";
var cspHtml = ({ cspRules, hashEnabled = {} }) => ({
  name: "csp-html",
  transformIndexHtml: {
    async handler(html) {
      const finalCspRules = {
        "base-uri": ["'self'"],
        ...cspRules
      };
      const hashRules = Object.entries(hashEnabled);
      if (hashRules.length) {
        const cheerio = await import("file:///Users/julianglover/Software/determined/webui/react/node_modules/cheerio/lib/esm/index.js");
        const $ = cheerio.load(html);
        hashRules.forEach(([directive, enabled]) => {
          if (!enabled)
            return;
          const [tag] = directive.split("-");
          $(tag).each((_, el) => {
            const source = $(el).html();
            if (source) {
              const hash = crypto.createHash("sha256").update(source).digest("base64");
              finalCspRules[directive] = (finalCspRules[directive] || []).concat([
                `'sha256-${hash}'`
              ]);
            }
          });
        });
      }
      const content = Object.entries(finalCspRules).map(([directive, sources]) => `${directive} ${sources.join(" ")}`).join("; ");
      return [
        {
          attrs: {
            content,
            "http-equiv": "Content-Security-Policy"
          },
          tag: "meta"
        }
      ];
    },
    order: "post"
  }
});

// vite-plugin-svg-to-jsx.ts
import { promises as fs } from "fs";
import { transform as swcTransform } from "file:///Users/julianglover/Software/determined/webui/react/node_modules/@swc/core/index.js";
import { jsx, toJs } from "file:///Users/julianglover/Software/determined/webui/react/node_modules/estree-util-to-js/index.js";
import { fromHtml } from "file:///Users/julianglover/Software/determined/webui/react/node_modules/hast-util-from-html/index.js";
import { toEstree } from "file:///Users/julianglover/Software/determined/webui/react/node_modules/hast-util-to-estree/index.js";
import { optimize } from "file:///Users/julianglover/Software/determined/webui/react/node_modules/svgo/lib/svgo-node.js";
var propsId = {
  name: "props",
  type: "Identifier"
};
var ReactComponentId = {
  name: "ReactComponent",
  type: "Identifier"
};
var svgToReact = (config) => {
  return [
    {
      enforce: "pre",
      async load(fullPath) {
        const [filePath, query] = fullPath.split("?", 2);
        if (filePath.endsWith(".svg") && !query) {
          const svgCode = await fs.readFile(filePath, { encoding: "utf8" });
          const optimizedSvgCode = optimize(svgCode, config);
          const hast = fromHtml(optimizedSvgCode.data, {
            fragment: true,
            space: "svg"
          });
          const estree = toEstree(hast.children[0], {
            space: "svg"
          });
          const expressionStatement = estree.body[0];
          if (expressionStatement.type !== "ExpressionStatement") {
            throw new Error("Parse error when adding props to jsx");
          }
          const jsxExpression = expressionStatement.expression;
          if (jsxExpression.type !== "JSXElement") {
            throw new Error("Parse error when adding props to jsx");
          }
          jsxExpression.openingElement.attributes.push({
            argument: propsId,
            type: "JSXSpreadAttribute"
          });
          estree.body[0] = {
            declaration: {
              declarations: [
                {
                  id: ReactComponentId,
                  init: {
                    body: jsxExpression,
                    expression: true,
                    params: [propsId],
                    type: "ArrowFunctionExpression"
                  },
                  type: "VariableDeclarator"
                }
              ],
              kind: "const",
              type: "VariableDeclaration"
            },
            specifiers: [],
            type: "ExportNamedDeclaration"
          };
          estree.body.push({
            declaration: ReactComponentId,
            type: "ExportDefaultDeclaration"
          });
          const newCode = toJs(estree, { handlers: jsx });
          return swcTransform(newCode.value, {
            filename: filePath,
            jsc: {
              parser: { jsx: true, syntax: "ecmascript" },
              target: "es2020",
              transform: {
                react: {
                  runtime: "automatic"
                }
              }
            }
          });
        }
      },
      name: "svg-to-react:transform"
    }
  ];
};

// vite.config.mts
var __vite_injected_original_dirname = "/Users/julianglover/Software/determined/webui/react";
var webpackProxyUrl = process.env.DET_WEBPACK_PROXY_URL || "http://localhost:8080";
var devServerRedirects = (redirects) => {
  let config;
  return {
    config(c) {
      config = c;
    },
    configureServer(server) {
      Object.entries(redirects).forEach(([from, to]) => {
        const fromUrl = `${config.base || ""}${from}`;
        server.middlewares.use(fromUrl, (req, res, next) => {
          if (req.originalUrl === fromUrl) {
            res.writeHead(302, {
              Location: `${config.base || ""}${to}`
            });
            res.end();
          } else {
            next();
          }
        });
      });
    },
    name: "dev-server-redirects"
  };
};
var publicUrlBaseHref = () => {
  let config;
  return {
    config(c) {
      config = c;
    },
    name: "public-url-base-href",
    transformIndexHtml: {
      handler() {
        return config.base ? [
          {
            attrs: {
              href: config.base
            },
            tag: "meta"
          }
        ] : [];
      }
    }
  };
};
var publicUrl = (process.env.PUBLIC_URL || "") === "/" ? void 0 : process.env.PUBLIC_URL;
var vite_config_default = defineConfig(({ mode }) => ({
  base: publicUrl,
  build: {
    commonjsOptions: {
      include: [/node_modules/, /notebook/]
    },
    outDir: "build",
    rollupOptions: {
      input: {
        design: path.resolve(__vite_injected_original_dirname, "design", "index.html"),
        main: path.resolve(__vite_injected_original_dirname, "index.html")
      },
      output: {
        manualChunks: (id) => {
          if (id.includes("node_modules")) {
            return "vendor";
          }
          if (id.endsWith(".svg")) {
            return "icons";
          }
        }
      }
    },
    sourcemap: mode === "production"
  },
  css: {
    modules: {
      generateScopedName: (name, filename) => {
        const basename = path.basename(filename).split(".")[0];
        const hashable = `${basename}_${name}`;
        const hash = crypto2.createHash("sha256").update(filename).digest("hex").substring(0, 5);
        return `${hashable}_${hash}`;
      }
    },
    preprocessorOptions: {
      scss: {
        additionalData: fs2.readFileSync("./src/styles/global.scss")
      }
    }
  },
  define: {
    "process.env.IS_DEV": JSON.stringify(mode === "development"),
    "process.env.PUBLIC_URL": JSON.stringify(mode !== "test" && publicUrl || ""),
    "process.env.SERVER_ADDRESS": JSON.stringify(process.env.SERVER_ADDRESS),
    "process.env.VERSION": '"0.26.1-dev0"'
  },
  optimizeDeps: {
    include: ["notebook"]
  },
  plugins: [
    tsconfigPaths(),
    svgToReact({
      plugins: [
        {
          name: "preset-default",
          params: {
            overrides: {
              convertColors: {
                currentColor: "#000"
              },
              removeViewBox: false
            }
          }
        }
      ]
    }),
    react(),
    publicUrlBaseHref(),
    mode !== "test" && checker({
      typescript: true
    }),
    devServerRedirects({
      "/design": "/design/"
    }),
    cspHtml({
      cspRules: {
        "frame-src": ["'self'", "netlify.determined.ai"],
        "object-src": ["'none'"],
        "script-src": ["'self'", "cdn.segment.com"],
        "style-src": ["'self'", "'unsafe-inline'"]
      },
      hashEnabled: {
        "script-src": true,
        "style-src": false
      }
    })
  ],
  preview: {
    port: 3001,
    strictPort: true
  },
  resolve: {
    alias: {
      // needed for react-dnd
      "react/jsx-runtime.js": "react/jsx-runtime"
    }
  },
  server: {
    open: true,
    port: 3e3,
    proxy: {
      "/api": { target: webpackProxyUrl },
      "/proxy": { target: webpackProxyUrl }
    },
    strictPort: true
  },
  test: {
    css: {
      modules: {
        classNameStrategy: "non-scoped"
      }
    },
    deps: {
      // necessary to fix react-dnd jsx runtime issue
      registerNodeLoader: true
    },
    environment: "jsdom",
    exclude: [...configDefaults.exclude, "./src/e2e/*"],
    globals: true,
    setupFiles: ["./src/setupTests.ts"]
  }
}));
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcubXRzIiwgInZpdGUtcGx1Z2luLWNzcC50cyIsICJ2aXRlLXBsdWdpbi1zdmctdG8tanN4LnRzIl0sCiAgInNvdXJjZXNDb250ZW50IjogWyJjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZGlybmFtZSA9IFwiL1VzZXJzL2p1bGlhbmdsb3Zlci9Tb2Z0d2FyZS9kZXRlcm1pbmVkL3dlYnVpL3JlYWN0XCI7Y29uc3QgX192aXRlX2luamVjdGVkX29yaWdpbmFsX2ZpbGVuYW1lID0gXCIvVXNlcnMvanVsaWFuZ2xvdmVyL1NvZnR3YXJlL2RldGVybWluZWQvd2VidWkvcmVhY3Qvdml0ZS5jb25maWcubXRzXCI7Y29uc3QgX192aXRlX2luamVjdGVkX29yaWdpbmFsX2ltcG9ydF9tZXRhX3VybCA9IFwiZmlsZTovLy9Vc2Vycy9qdWxpYW5nbG92ZXIvU29mdHdhcmUvZGV0ZXJtaW5lZC93ZWJ1aS9yZWFjdC92aXRlLmNvbmZpZy5tdHNcIjtpbXBvcnQgY3J5cHRvIGZyb20gJ2NyeXB0byc7XG5pbXBvcnQgZnMgZnJvbSAnZnMnO1xuaW1wb3J0IHBhdGggZnJvbSAncGF0aCc7XG5cbmltcG9ydCByZWFjdCBmcm9tICdAdml0ZWpzL3BsdWdpbi1yZWFjdC1zd2MnO1xuaW1wb3J0IHsgUGx1Z2luLCBVc2VyQ29uZmlnIH0gZnJvbSAndml0ZSc7XG5pbXBvcnQgY2hlY2tlciBmcm9tICd2aXRlLXBsdWdpbi1jaGVja2VyJztcbmltcG9ydCB0c2NvbmZpZ1BhdGhzIGZyb20gJ3ZpdGUtdHNjb25maWctcGF0aHMnO1xuaW1wb3J0IHsgY29uZmlnRGVmYXVsdHMsIGRlZmluZUNvbmZpZyB9IGZyb20gJ3ZpdGVzdC9jb25maWcnO1xuXG5pbXBvcnQgeyBjc3BIdG1sIH0gZnJvbSAnLi92aXRlLXBsdWdpbi1jc3AnO1xuaW1wb3J0IHsgc3ZnVG9SZWFjdCB9IGZyb20gJy4vdml0ZS1wbHVnaW4tc3ZnLXRvLWpzeCc7XG5cbi8vIHdhbnQgdG8gZmFsbGJhY2sgaW4gY2FzZSBvZiBlbXB0eSBzdHJpbmcsIGhlbmNlIG5vID8/XG5jb25zdCB3ZWJwYWNrUHJveHlVcmwgPSBwcm9jZXNzLmVudi5ERVRfV0VCUEFDS19QUk9YWV9VUkwgfHwgJ2h0dHA6Ly9sb2NhbGhvc3Q6ODA4MCc7XG5cbmNvbnN0IGRldlNlcnZlclJlZGlyZWN0cyA9IChyZWRpcmVjdHM6IFJlY29yZDxzdHJpbmcsIHN0cmluZz4pOiBQbHVnaW4gPT4ge1xuICBsZXQgY29uZmlnOiBVc2VyQ29uZmlnO1xuICByZXR1cm4ge1xuICAgIGNvbmZpZyhjKSB7XG4gICAgICBjb25maWcgPSBjO1xuICAgIH0sXG4gICAgY29uZmlndXJlU2VydmVyKHNlcnZlcikge1xuICAgICAgT2JqZWN0LmVudHJpZXMocmVkaXJlY3RzKS5mb3JFYWNoKChbZnJvbSwgdG9dKSA9PiB7XG4gICAgICAgIGNvbnN0IGZyb21VcmwgPSBgJHtjb25maWcuYmFzZSB8fCAnJ30ke2Zyb219YDtcbiAgICAgICAgc2VydmVyLm1pZGRsZXdhcmVzLnVzZShmcm9tVXJsLCAocmVxLCByZXMsIG5leHQpID0+IHtcbiAgICAgICAgICBpZiAocmVxLm9yaWdpbmFsVXJsID09PSBmcm9tVXJsKSB7XG4gICAgICAgICAgICByZXMud3JpdGVIZWFkKDMwMiwge1xuICAgICAgICAgICAgICBMb2NhdGlvbjogYCR7Y29uZmlnLmJhc2UgfHwgJyd9JHt0b31gLFxuICAgICAgICAgICAgfSk7XG4gICAgICAgICAgICByZXMuZW5kKCk7XG4gICAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICAgIG5leHQoKTtcbiAgICAgICAgICB9XG4gICAgICAgIH0pO1xuICAgICAgfSk7XG4gICAgfSxcbiAgICBuYW1lOiAnZGV2LXNlcnZlci1yZWRpcmVjdHMnLFxuICB9O1xufTtcblxuY29uc3QgcHVibGljVXJsQmFzZUhyZWYgPSAoKTogUGx1Z2luID0+IHtcbiAgbGV0IGNvbmZpZzogVXNlckNvbmZpZztcbiAgcmV0dXJuIHtcbiAgICBjb25maWcoYykge1xuICAgICAgY29uZmlnID0gYztcbiAgICB9LFxuICAgIG5hbWU6ICdwdWJsaWMtdXJsLWJhc2UtaHJlZicsXG4gICAgdHJhbnNmb3JtSW5kZXhIdG1sOiB7XG4gICAgICBoYW5kbGVyKCkge1xuICAgICAgICByZXR1cm4gY29uZmlnLmJhc2VcbiAgICAgICAgICA/IFtcbiAgICAgICAgICAgICAge1xuICAgICAgICAgICAgICAgIGF0dHJzOiB7XG4gICAgICAgICAgICAgICAgICBocmVmOiBjb25maWcuYmFzZSxcbiAgICAgICAgICAgICAgICB9LFxuICAgICAgICAgICAgICAgIHRhZzogJ21ldGEnLFxuICAgICAgICAgICAgICB9LFxuICAgICAgICAgICAgXVxuICAgICAgICAgIDogW107XG4gICAgICB9LFxuICAgIH0sXG4gIH07XG59O1xuXG4vLyBwdWJsaWNfdXJsIGFzIC8gYnJlYWtzIHRoZSBsaW5rIGNvbXBvbmVudCAtLSBhc3N1bWluZyB0aGF0IENSQSBkaWQgc29tZXRoaW5nXG4vLyB0byBwcmV2ZW50IHRoYXQsIGlka1xuY29uc3QgcHVibGljVXJsID0gKHByb2Nlc3MuZW52LlBVQkxJQ19VUkwgfHwgJycpID09PSAnLycgPyB1bmRlZmluZWQgOiBwcm9jZXNzLmVudi5QVUJMSUNfVVJMO1xuXG4vLyBodHRwczovL3ZpdGVqcy5kZXYvY29uZmlnL1xuZXhwb3J0IGRlZmF1bHQgZGVmaW5lQ29uZmlnKCh7IG1vZGUgfSkgPT4gKHtcbiAgYmFzZTogcHVibGljVXJsLFxuICBidWlsZDoge1xuICAgIGNvbW1vbmpzT3B0aW9uczoge1xuICAgICAgaW5jbHVkZTogWy9ub2RlX21vZHVsZXMvLCAvbm90ZWJvb2svXSxcbiAgICB9LFxuICAgIG91dERpcjogJ2J1aWxkJyxcbiAgICByb2xsdXBPcHRpb25zOiB7XG4gICAgICBpbnB1dDoge1xuICAgICAgICBkZXNpZ246IHBhdGgucmVzb2x2ZShfX2Rpcm5hbWUsICdkZXNpZ24nLCAnaW5kZXguaHRtbCcpLFxuICAgICAgICBtYWluOiBwYXRoLnJlc29sdmUoX19kaXJuYW1lLCAnaW5kZXguaHRtbCcpLFxuICAgICAgfSxcbiAgICAgIG91dHB1dDoge1xuICAgICAgICBtYW51YWxDaHVua3M6IChpZCkgPT4ge1xuICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnbm9kZV9tb2R1bGVzJykpIHtcbiAgICAgICAgICAgIHJldHVybiAndmVuZG9yJztcbiAgICAgICAgICB9XG4gICAgICAgICAgaWYgKGlkLmVuZHNXaXRoKCcuc3ZnJykpIHtcbiAgICAgICAgICAgIHJldHVybiAnaWNvbnMnO1xuICAgICAgICAgIH1cbiAgICAgICAgfSxcbiAgICAgIH0sXG4gICAgfSxcbiAgICBzb3VyY2VtYXA6IG1vZGUgPT09ICdwcm9kdWN0aW9uJyxcbiAgfSxcbiAgY3NzOiB7XG4gICAgbW9kdWxlczoge1xuICAgICAgZ2VuZXJhdGVTY29wZWROYW1lOiAobmFtZSwgZmlsZW5hbWUpID0+IHtcbiAgICAgICAgY29uc3QgYmFzZW5hbWUgPSBwYXRoLmJhc2VuYW1lKGZpbGVuYW1lKS5zcGxpdCgnLicpWzBdO1xuICAgICAgICBjb25zdCBoYXNoYWJsZSA9IGAke2Jhc2VuYW1lfV8ke25hbWV9YDtcbiAgICAgICAgY29uc3QgaGFzaCA9IGNyeXB0by5jcmVhdGVIYXNoKCdzaGEyNTYnKS51cGRhdGUoZmlsZW5hbWUpLmRpZ2VzdCgnaGV4Jykuc3Vic3RyaW5nKDAsIDUpO1xuXG4gICAgICAgIHJldHVybiBgJHtoYXNoYWJsZX1fJHtoYXNofWA7XG4gICAgICB9LFxuICAgIH0sXG4gICAgcHJlcHJvY2Vzc29yT3B0aW9uczoge1xuICAgICAgc2Nzczoge1xuICAgICAgICBhZGRpdGlvbmFsRGF0YTogZnMucmVhZEZpbGVTeW5jKCcuL3NyYy9zdHlsZXMvZ2xvYmFsLnNjc3MnKSxcbiAgICAgIH0sXG4gICAgfSxcbiAgfSxcbiAgZGVmaW5lOiB7XG4gICAgJ3Byb2Nlc3MuZW52LklTX0RFVic6IEpTT04uc3RyaW5naWZ5KG1vZGUgPT09ICdkZXZlbG9wbWVudCcpLFxuICAgICdwcm9jZXNzLmVudi5QVUJMSUNfVVJMJzogSlNPTi5zdHJpbmdpZnkoKG1vZGUgIT09ICd0ZXN0JyAmJiBwdWJsaWNVcmwpIHx8ICcnKSxcbiAgICAncHJvY2Vzcy5lbnYuU0VSVkVSX0FERFJFU1MnOiBKU09OLnN0cmluZ2lmeShwcm9jZXNzLmVudi5TRVJWRVJfQUREUkVTUyksXG4gICAgJ3Byb2Nlc3MuZW52LlZFUlNJT04nOiAnXCIwLjI2LjEtZGV2MFwiJyxcbiAgfSxcbiAgb3B0aW1pemVEZXBzOiB7XG4gICAgaW5jbHVkZTogWydub3RlYm9vayddLFxuICB9LFxuICBwbHVnaW5zOiBbXG4gICAgdHNjb25maWdQYXRocygpLFxuICAgIHN2Z1RvUmVhY3Qoe1xuICAgICAgcGx1Z2luczogW1xuICAgICAgICB7XG4gICAgICAgICAgbmFtZTogJ3ByZXNldC1kZWZhdWx0JyxcbiAgICAgICAgICBwYXJhbXM6IHtcbiAgICAgICAgICAgIG92ZXJyaWRlczoge1xuICAgICAgICAgICAgICBjb252ZXJ0Q29sb3JzOiB7XG4gICAgICAgICAgICAgICAgY3VycmVudENvbG9yOiAnIzAwMCcsXG4gICAgICAgICAgICAgIH0sXG4gICAgICAgICAgICAgIHJlbW92ZVZpZXdCb3g6IGZhbHNlLFxuICAgICAgICAgICAgfSxcbiAgICAgICAgICB9LFxuICAgICAgICB9LFxuICAgICAgXSxcbiAgICB9KSxcbiAgICByZWFjdCgpLFxuICAgIHB1YmxpY1VybEJhc2VIcmVmKCksXG4gICAgbW9kZSAhPT0gJ3Rlc3QnICYmXG4gICAgICBjaGVja2VyKHtcbiAgICAgICAgdHlwZXNjcmlwdDogdHJ1ZSxcbiAgICAgIH0pLFxuICAgIGRldlNlcnZlclJlZGlyZWN0cyh7XG4gICAgICAnL2Rlc2lnbic6ICcvZGVzaWduLycsXG4gICAgfSksXG4gICAgY3NwSHRtbCh7XG4gICAgICBjc3BSdWxlczoge1xuICAgICAgICAnZnJhbWUtc3JjJzogW1wiJ3NlbGYnXCIsICduZXRsaWZ5LmRldGVybWluZWQuYWknXSxcbiAgICAgICAgJ29iamVjdC1zcmMnOiBbXCInbm9uZSdcIl0sXG4gICAgICAgICdzY3JpcHQtc3JjJzogW1wiJ3NlbGYnXCIsICdjZG4uc2VnbWVudC5jb20nXSxcbiAgICAgICAgJ3N0eWxlLXNyYyc6IFtcIidzZWxmJ1wiLCBcIid1bnNhZmUtaW5saW5lJ1wiXSxcbiAgICAgIH0sXG4gICAgICBoYXNoRW5hYmxlZDoge1xuICAgICAgICAnc2NyaXB0LXNyYyc6IHRydWUsXG4gICAgICAgICdzdHlsZS1zcmMnOiBmYWxzZSxcbiAgICAgIH0sXG4gICAgfSksXG4gIF0sXG4gIHByZXZpZXc6IHtcbiAgICBwb3J0OiAzMDAxLFxuICAgIHN0cmljdFBvcnQ6IHRydWUsXG4gIH0sXG4gIHJlc29sdmU6IHtcbiAgICBhbGlhczoge1xuICAgICAgLy8gbmVlZGVkIGZvciByZWFjdC1kbmRcbiAgICAgICdyZWFjdC9qc3gtcnVudGltZS5qcyc6ICdyZWFjdC9qc3gtcnVudGltZScsXG4gICAgfSxcbiAgfSxcbiAgc2VydmVyOiB7XG4gICAgb3BlbjogdHJ1ZSxcbiAgICBwb3J0OiAzMDAwLFxuICAgIHByb3h5OiB7XG4gICAgICAnL2FwaSc6IHsgdGFyZ2V0OiB3ZWJwYWNrUHJveHlVcmwgfSxcbiAgICAgICcvcHJveHknOiB7IHRhcmdldDogd2VicGFja1Byb3h5VXJsIH0sXG4gICAgfSxcbiAgICBzdHJpY3RQb3J0OiB0cnVlLFxuICB9LFxuICB0ZXN0OiB7XG4gICAgY3NzOiB7XG4gICAgICBtb2R1bGVzOiB7XG4gICAgICAgIGNsYXNzTmFtZVN0cmF0ZWd5OiAnbm9uLXNjb3BlZCcsXG4gICAgICB9LFxuICAgIH0sXG4gICAgZGVwczoge1xuICAgICAgLy8gbmVjZXNzYXJ5IHRvIGZpeCByZWFjdC1kbmQganN4IHJ1bnRpbWUgaXNzdWVcbiAgICAgIHJlZ2lzdGVyTm9kZUxvYWRlcjogdHJ1ZSxcbiAgICB9LFxuICAgIGVudmlyb25tZW50OiAnanNkb20nLFxuICAgIGV4Y2x1ZGU6IFsuLi5jb25maWdEZWZhdWx0cy5leGNsdWRlLCAnLi9zcmMvZTJlLyonXSxcbiAgICBnbG9iYWxzOiB0cnVlLFxuICAgIHNldHVwRmlsZXM6IFsnLi9zcmMvc2V0dXBUZXN0cy50cyddLFxuICB9LFxufSkpO1xuIiwgImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvVXNlcnMvanVsaWFuZ2xvdmVyL1NvZnR3YXJlL2RldGVybWluZWQvd2VidWkvcmVhY3RcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIi9Vc2Vycy9qdWxpYW5nbG92ZXIvU29mdHdhcmUvZGV0ZXJtaW5lZC93ZWJ1aS9yZWFjdC92aXRlLXBsdWdpbi1jc3AudHNcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfaW1wb3J0X21ldGFfdXJsID0gXCJmaWxlOi8vL1VzZXJzL2p1bGlhbmdsb3Zlci9Tb2Z0d2FyZS9kZXRlcm1pbmVkL3dlYnVpL3JlYWN0L3ZpdGUtcGx1Z2luLWNzcC50c1wiO2ltcG9ydCBjcnlwdG8gZnJvbSAnY3J5cHRvJztcblxuaW1wb3J0IHR5cGUgeyBQbHVnaW4gfSBmcm9tICd2aXRlJztcblxuLy8gaW5jb21wbGV0ZSBsaXN0IG9mIGRpcmVjdGl2ZXNcbnR5cGUgQ3NwSGFzaERpcmVjdGl2ZSA9ICdzY3JpcHQtc3JjJyB8ICdzdHlsZS1zcmMnO1xudHlwZSBDc3BEaXJlY3RpdmUgPSAnYmFzZS11cmknIHwgJ2ZyYW1lLXNyYycgfCAnb2JqZWN0LXNyYycgfCBDc3BIYXNoRGlyZWN0aXZlO1xuXG50eXBlIENzcFJ1bGVDb25maWcgPSB7XG4gIFtrZXkgaW4gQ3NwRGlyZWN0aXZlXT86IHN0cmluZ1tdO1xufTtcblxudHlwZSBDc3BIYXNoQ29uZmlnID0ge1xuICBba2V5IGluIENzcEhhc2hEaXJlY3RpdmVdPzogYm9vbGVhbjtcbn07XG5cbmludGVyZmFjZSBDc3BIdG1sUGx1Z2luQ29uZmlnIHtcbiAgY3NwUnVsZXM6IENzcFJ1bGVDb25maWc7XG4gIGhhc2hFbmFibGVkOiBDc3BIYXNoQ29uZmlnO1xufVxuXG5leHBvcnQgY29uc3QgY3NwSHRtbCA9ICh7IGNzcFJ1bGVzLCBoYXNoRW5hYmxlZCA9IHt9IH06IENzcEh0bWxQbHVnaW5Db25maWcpOiBQbHVnaW4gPT4gKHtcbiAgbmFtZTogJ2NzcC1odG1sJyxcbiAgdHJhbnNmb3JtSW5kZXhIdG1sOiB7XG4gICAgYXN5bmMgaGFuZGxlcihodG1sOiBzdHJpbmcpIHtcbiAgICAgIGNvbnN0IGZpbmFsQ3NwUnVsZXM6IENzcFJ1bGVDb25maWcgPSB7XG4gICAgICAgICdiYXNlLXVyaSc6IFtcIidzZWxmJ1wiXSxcbiAgICAgICAgLi4uY3NwUnVsZXMsXG4gICAgICB9O1xuICAgICAgY29uc3QgaGFzaFJ1bGVzID0gT2JqZWN0LmVudHJpZXMoaGFzaEVuYWJsZWQpIGFzIFtDc3BIYXNoRGlyZWN0aXZlLCBib29sZWFuXVtdO1xuICAgICAgaWYgKGhhc2hSdWxlcy5sZW5ndGgpIHtcbiAgICAgICAgY29uc3QgY2hlZXJpbyA9IGF3YWl0IGltcG9ydCgnY2hlZXJpbycpO1xuICAgICAgICBjb25zdCAkID0gY2hlZXJpby5sb2FkKGh0bWwpO1xuICAgICAgICBoYXNoUnVsZXMuZm9yRWFjaCgoW2RpcmVjdGl2ZSwgZW5hYmxlZF06IFtDc3BIYXNoRGlyZWN0aXZlLCBib29sZWFuXSkgPT4ge1xuICAgICAgICAgIGlmICghZW5hYmxlZCkgcmV0dXJuO1xuICAgICAgICAgIGNvbnN0IFt0YWddID0gZGlyZWN0aXZlLnNwbGl0KCctJyk7XG4gICAgICAgICAgJCh0YWcpLmVhY2goKF8sIGVsKSA9PiB7XG4gICAgICAgICAgICBjb25zdCBzb3VyY2UgPSAkKGVsKS5odG1sKCk7XG4gICAgICAgICAgICBpZiAoc291cmNlKSB7XG4gICAgICAgICAgICAgIGNvbnN0IGhhc2ggPSBjcnlwdG8uY3JlYXRlSGFzaCgnc2hhMjU2JykudXBkYXRlKHNvdXJjZSkuZGlnZXN0KCdiYXNlNjQnKTtcbiAgICAgICAgICAgICAgZmluYWxDc3BSdWxlc1tkaXJlY3RpdmVdID0gKGZpbmFsQ3NwUnVsZXNbZGlyZWN0aXZlXSB8fCBbXSkuY29uY2F0KFtcbiAgICAgICAgICAgICAgICBgJ3NoYTI1Ni0ke2hhc2h9J2AsXG4gICAgICAgICAgICAgIF0pO1xuICAgICAgICAgICAgfVxuICAgICAgICAgIH0pO1xuICAgICAgICB9KTtcbiAgICAgIH1cbiAgICAgIGNvbnN0IGNvbnRlbnQgPSBPYmplY3QuZW50cmllcyhmaW5hbENzcFJ1bGVzKVxuICAgICAgICAubWFwKChbZGlyZWN0aXZlLCBzb3VyY2VzXSkgPT4gYCR7ZGlyZWN0aXZlfSAke3NvdXJjZXMuam9pbignICcpfWApXG4gICAgICAgIC5qb2luKCc7ICcpO1xuICAgICAgcmV0dXJuIFtcbiAgICAgICAge1xuICAgICAgICAgIGF0dHJzOiB7XG4gICAgICAgICAgICBjb250ZW50LFxuICAgICAgICAgICAgJ2h0dHAtZXF1aXYnOiAnQ29udGVudC1TZWN1cml0eS1Qb2xpY3knLFxuICAgICAgICAgIH0sXG4gICAgICAgICAgdGFnOiAnbWV0YScsXG4gICAgICAgIH0sXG4gICAgICBdO1xuICAgIH0sXG4gICAgb3JkZXI6ICdwb3N0JyxcbiAgfSxcbn0pO1xuIiwgImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvVXNlcnMvanVsaWFuZ2xvdmVyL1NvZnR3YXJlL2RldGVybWluZWQvd2VidWkvcmVhY3RcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIi9Vc2Vycy9qdWxpYW5nbG92ZXIvU29mdHdhcmUvZGV0ZXJtaW5lZC93ZWJ1aS9yZWFjdC92aXRlLXBsdWdpbi1zdmctdG8tanN4LnRzXCI7Y29uc3QgX192aXRlX2luamVjdGVkX29yaWdpbmFsX2ltcG9ydF9tZXRhX3VybCA9IFwiZmlsZTovLy9Vc2Vycy9qdWxpYW5nbG92ZXIvU29mdHdhcmUvZGV0ZXJtaW5lZC93ZWJ1aS9yZWFjdC92aXRlLXBsdWdpbi1zdmctdG8tanN4LnRzXCI7aW1wb3J0IHsgcHJvbWlzZXMgYXMgZnMgfSBmcm9tICdmcyc7XG5cbmltcG9ydCB7IHRyYW5zZm9ybSBhcyBzd2NUcmFuc2Zvcm0gfSBmcm9tICdAc3djL2NvcmUnO1xuaW1wb3J0IHsganN4LCB0b0pzIH0gZnJvbSAnZXN0cmVlLXV0aWwtdG8tanMnO1xuaW1wb3J0IHsgZnJvbUh0bWwgfSBmcm9tICdoYXN0LXV0aWwtZnJvbS1odG1sJztcbmltcG9ydCB7IHRvRXN0cmVlIH0gZnJvbSAnaGFzdC11dGlsLXRvLWVzdHJlZSc7XG5pbXBvcnQgeyBvcHRpbWl6ZSwgQ29uZmlnIGFzIFN2Z29Db25maWcgfSBmcm9tICdzdmdvJztcbmltcG9ydCB7IFBsdWdpbiB9IGZyb20gJ3ZpdGUnO1xuXG5jb25zdCBwcm9wc0lkID0ge1xuICBuYW1lOiAncHJvcHMnLFxuICB0eXBlOiAnSWRlbnRpZmllcicsXG59IGFzIGNvbnN0O1xuY29uc3QgUmVhY3RDb21wb25lbnRJZCA9IHtcbiAgbmFtZTogJ1JlYWN0Q29tcG9uZW50JyxcbiAgdHlwZTogJ0lkZW50aWZpZXInLFxufSBhcyBjb25zdDtcbmV4cG9ydCBjb25zdCBzdmdUb1JlYWN0ID0gKGNvbmZpZzogU3Znb0NvbmZpZyk6IFBsdWdpbltdID0+IHtcbiAgcmV0dXJuIFtcbiAgICB7XG4gICAgICBlbmZvcmNlOiAncHJlJyxcbiAgICAgIGFzeW5jIGxvYWQoZnVsbFBhdGgpIHtcbiAgICAgICAgY29uc3QgW2ZpbGVQYXRoLCBxdWVyeV0gPSBmdWxsUGF0aC5zcGxpdCgnPycsIDIpO1xuICAgICAgICAvLyB0cmVhdCB0aGUgc3ZnIGFzIG5vcm1hbCBpZiB0aGVyZSdzIGEgcXVlcnlcbiAgICAgICAgaWYgKGZpbGVQYXRoLmVuZHNXaXRoKCcuc3ZnJykgJiYgIXF1ZXJ5KSB7XG4gICAgICAgICAgY29uc3Qgc3ZnQ29kZSA9IGF3YWl0IGZzLnJlYWRGaWxlKGZpbGVQYXRoLCB7IGVuY29kaW5nOiAndXRmOCcgfSk7XG4gICAgICAgICAgY29uc3Qgb3B0aW1pemVkU3ZnQ29kZSA9IG9wdGltaXplKHN2Z0NvZGUsIGNvbmZpZyk7XG4gICAgICAgICAgY29uc3QgaGFzdCA9IGZyb21IdG1sKG9wdGltaXplZFN2Z0NvZGUuZGF0YSwge1xuICAgICAgICAgICAgZnJhZ21lbnQ6IHRydWUsXG4gICAgICAgICAgICBzcGFjZTogJ3N2ZycsXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgLy8gZ2V0IHRoZSBmaXJzdCBjaGlsZCBvZiB0aGUgcm9vdCBub2RlIHNvIHdlIGRvbid0IGR1bXAgZXZlcnl0aGluZyBpbnRvIGEgZnJhZ21lbnRcbiAgICAgICAgICBjb25zdCBlc3RyZWUgPSB0b0VzdHJlZShoYXN0LmNoaWxkcmVuWzBdLCB7XG4gICAgICAgICAgICBzcGFjZTogJ3N2ZycsXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgY29uc3QgZXhwcmVzc2lvblN0YXRlbWVudCA9IGVzdHJlZS5ib2R5WzBdO1xuICAgICAgICAgIGlmIChleHByZXNzaW9uU3RhdGVtZW50LnR5cGUgIT09ICdFeHByZXNzaW9uU3RhdGVtZW50Jykge1xuICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKCdQYXJzZSBlcnJvciB3aGVuIGFkZGluZyBwcm9wcyB0byBqc3gnKTtcbiAgICAgICAgICB9XG4gICAgICAgICAgY29uc3QganN4RXhwcmVzc2lvbiA9IGV4cHJlc3Npb25TdGF0ZW1lbnQuZXhwcmVzc2lvbjtcbiAgICAgICAgICBpZiAoanN4RXhwcmVzc2lvbi50eXBlICE9PSAnSlNYRWxlbWVudCcpIHtcbiAgICAgICAgICAgIHRocm93IG5ldyBFcnJvcignUGFyc2UgZXJyb3Igd2hlbiBhZGRpbmcgcHJvcHMgdG8ganN4Jyk7XG4gICAgICAgICAgfVxuICAgICAgICAgIC8vIHNwcmVhZCBwcm9wcyBpbnRvIHRoZSBlbGVtZW50XG4gICAgICAgICAganN4RXhwcmVzc2lvbi5vcGVuaW5nRWxlbWVudC5hdHRyaWJ1dGVzLnB1c2goe1xuICAgICAgICAgICAgYXJndW1lbnQ6IHByb3BzSWQsXG4gICAgICAgICAgICB0eXBlOiAnSlNYU3ByZWFkQXR0cmlidXRlJyxcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBlc3RyZWUuYm9keVswXSA9IHtcbiAgICAgICAgICAgIGRlY2xhcmF0aW9uOiB7XG4gICAgICAgICAgICAgIGRlY2xhcmF0aW9uczogW1xuICAgICAgICAgICAgICAgIHtcbiAgICAgICAgICAgICAgICAgIGlkOiBSZWFjdENvbXBvbmVudElkLFxuICAgICAgICAgICAgICAgICAgaW5pdDoge1xuICAgICAgICAgICAgICAgICAgICBib2R5OiBqc3hFeHByZXNzaW9uLFxuICAgICAgICAgICAgICAgICAgICBleHByZXNzaW9uOiB0cnVlLFxuICAgICAgICAgICAgICAgICAgICBwYXJhbXM6IFtwcm9wc0lkXSxcbiAgICAgICAgICAgICAgICAgICAgdHlwZTogJ0Fycm93RnVuY3Rpb25FeHByZXNzaW9uJyxcbiAgICAgICAgICAgICAgICAgIH0sXG4gICAgICAgICAgICAgICAgICB0eXBlOiAnVmFyaWFibGVEZWNsYXJhdG9yJyxcbiAgICAgICAgICAgICAgICB9LFxuICAgICAgICAgICAgICBdLFxuICAgICAgICAgICAgICBraW5kOiAnY29uc3QnLFxuICAgICAgICAgICAgICB0eXBlOiAnVmFyaWFibGVEZWNsYXJhdGlvbicsXG4gICAgICAgICAgICB9LFxuICAgICAgICAgICAgc3BlY2lmaWVyczogW10sXG4gICAgICAgICAgICB0eXBlOiAnRXhwb3J0TmFtZWREZWNsYXJhdGlvbicsXG4gICAgICAgICAgfTtcbiAgICAgICAgICBlc3RyZWUuYm9keS5wdXNoKHtcbiAgICAgICAgICAgIGRlY2xhcmF0aW9uOiBSZWFjdENvbXBvbmVudElkLFxuICAgICAgICAgICAgdHlwZTogJ0V4cG9ydERlZmF1bHREZWNsYXJhdGlvbicsXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgY29uc3QgbmV3Q29kZSA9IHRvSnMoZXN0cmVlLCB7IGhhbmRsZXJzOiBqc3ggfSk7XG4gICAgICAgICAgcmV0dXJuIHN3Y1RyYW5zZm9ybShuZXdDb2RlLnZhbHVlLCB7XG4gICAgICAgICAgICBmaWxlbmFtZTogZmlsZVBhdGgsXG4gICAgICAgICAgICBqc2M6IHtcbiAgICAgICAgICAgICAgcGFyc2VyOiB7IGpzeDogdHJ1ZSwgc3ludGF4OiAnZWNtYXNjcmlwdCcgfSxcbiAgICAgICAgICAgICAgdGFyZ2V0OiAnZXMyMDIwJyxcbiAgICAgICAgICAgICAgdHJhbnNmb3JtOiB7XG4gICAgICAgICAgICAgICAgcmVhY3Q6IHtcbiAgICAgICAgICAgICAgICAgIHJ1bnRpbWU6ICdhdXRvbWF0aWMnLFxuICAgICAgICAgICAgICAgIH0sXG4gICAgICAgICAgICAgIH0sXG4gICAgICAgICAgICB9LFxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG4gICAgICB9LFxuICAgICAgbmFtZTogJ3N2Zy10by1yZWFjdDp0cmFuc2Zvcm0nLFxuICAgIH0sXG4gIF07XG59O1xuIl0sCiAgIm1hcHBpbmdzIjogIjtBQUE2VSxPQUFPQSxhQUFZO0FBQ2hXLE9BQU9DLFNBQVE7QUFDZixPQUFPLFVBQVU7QUFFakIsT0FBTyxXQUFXO0FBRWxCLE9BQU8sYUFBYTtBQUNwQixPQUFPLG1CQUFtQjtBQUMxQixTQUFTLGdCQUFnQixvQkFBb0I7OztBQ1JzUyxPQUFPLFlBQVk7QUFxQi9WLElBQU0sVUFBVSxDQUFDLEVBQUUsVUFBVSxjQUFjLENBQUMsRUFBRSxPQUFvQztBQUFBLEVBQ3ZGLE1BQU07QUFBQSxFQUNOLG9CQUFvQjtBQUFBLElBQ2xCLE1BQU0sUUFBUSxNQUFjO0FBQzFCLFlBQU0sZ0JBQStCO0FBQUEsUUFDbkMsWUFBWSxDQUFDLFFBQVE7QUFBQSxRQUNyQixHQUFHO0FBQUEsTUFDTDtBQUNBLFlBQU0sWUFBWSxPQUFPLFFBQVEsV0FBVztBQUM1QyxVQUFJLFVBQVUsUUFBUTtBQUNwQixjQUFNLFVBQVUsTUFBTSxPQUFPLGtHQUFTO0FBQ3RDLGNBQU0sSUFB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
