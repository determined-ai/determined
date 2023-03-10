/* eslint-disable @typescript-eslint/no-var-requires */
const { DefinePlugin } = require('webpack');

const config = require('./src/shared/configs/craco.config');

const webpackEnvPlugin = new DefinePlugin({
  'process.env.IS_DEV': JSON.stringify(config.isDev),
  'process.env.SERVER_ADDRESS': JSON.stringify(process.env.SERVER_ADDRESS),
  'process.env.VERSION': '"0.20.1-dev0"',
});


// want to fallback in case of empty string, hence no ??
const webpackProxyUrl = process.env.DET_WEBPACK_PROXY_URL || 'http://localhost:8080'

module.exports = {
  ...config,
  devServer: {
    proxy:
    /**
     * ideally, we could proxy all {serverAddress}:3000/{api|proxy}
     * to {serverAddress}:8080{api|proxy}. devServer only intercepts
     * requests to the server itself though
     */
    {
      '/api': { target: webpackProxyUrl },
      '/proxy': { target: webpackProxyUrl },
    },
  },
  webpack: {
    ...config.webpack,
    plugins: [
      ...config.webpack.plugins,
      webpackEnvPlugin,
    ],
  },
};
