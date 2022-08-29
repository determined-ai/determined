/* eslint-disable @typescript-eslint/no-var-requires */
const path = require('path');

const AntDesignThemePlugin = require('antd-theme-webpack-plugin');
const { DefinePlugin } = require('webpack');

const config = require('./src/shared/configs/craco.config');

const webpackEnvPlugin = new DefinePlugin({
  'process.env.IS_DEV': JSON.stringify(config.isDev),
  'process.env.SERVER_ADDRESS': JSON.stringify(process.env.SERVER_ADDRESS),
  'process.env.VERSION': '"0.19.3-dev0"',
});

/**
 * Add theme override support for antd. For more options:
 * https://github.com/mzohaibqc/antd-theme-webpack-plugin
*/
const antdPlugin = new AntDesignThemePlugin({
  antDir: path.join(__dirname, './node_modules/antd'),
  indexFileName: 'index.html',
  mainLessFile: path.join(__dirname, './src/shared/styles/index.less'),
  publicPath: process.env.PUBLIC_URL,
  stylesDir: path.join(__dirname, './src/shared/styles'),
  themeVariables: [ '@primary-color' ],
  varFile: path.join(__dirname, './src/shared/styles/variables.less'),
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
      antdPlugin,
    ],
  },
};
