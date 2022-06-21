/* eslint-disable @typescript-eslint/no-var-requires */
const { DefinePlugin } = require('webpack');

const config = require('./src/shared/configs/craco.config');

const webpackEnvPlugin = new DefinePlugin({
  'process.env.IS_DEV': JSON.stringify(config.isDev),
  'process.env.SERVER_ADDRESS': JSON.stringify(process.env.SERVER_ADDRESS),
  'process.env.VERSION': '"0.18.3-dev0"',
});

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
      '/api': { target: 'http://localhost:8080' },
      '/proxy': { target: 'http://localhost:8080' },
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
