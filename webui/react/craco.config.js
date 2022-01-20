/* eslint-disable @typescript-eslint/no-var-requires */
const path = require('path');

const { when } = require('@craco/craco');
const AntdDayjsWebpackPlugin = require('antd-dayjs-webpack-plugin');
const AntDesignThemePlugin = require('antd-theme-webpack-plugin');
const CracoLessPlugin = require('craco-less');
const CracoSassResoucesPlugin = require('craco-sass-resources-loader');
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');
const { DefinePlugin } = require('webpack');

const IS_DEV = process.env.DET_NODE_ENV === 'development';

module.exports = {
  babel: {
    plugins: [
      [
        'import', {
          libraryDirectory: 'es',
          libraryName: 'antd',
          style: true,
        },
      ],
    ],
  },
  eslint: { enable: false },
  plugins: [
    {
      options: { lessLoaderOptions: { lessOptions: { javascriptEnabled: true } } },
      plugin: CracoLessPlugin,
    },
    {
      options: { resources: './src/styles/global.scss' },
      plugin: CracoSassResoucesPlugin,
    },
  ],
  webpack: {
    // Skip webpack v4 optimizations when development environment to lower build time.
    ...when(
      IS_DEV,
      () => ({
        configure: {
          optimization: {
            concatenateModules: false,
            flagIncludedChunks: false,
            mergeDuplicateChunks: false,
            minimize: false,
            namedChunks: false,
            namedModules: false,
            occurrenceOrder: false,
            providedExports: false,
            removeAvailableModules: false,
            removeEmptyChunks: false,
            sideEffects: false,
            usedExports: false,
          },
        },
      }),
      {},
    ),
    configure: {
      module: {
        rules: [
          /*
           * Plotly needs browserify transformation applied when building production files.
           * https://github.com/plotly/plotly.js/blob/master/BUILDING.md#webpack
           */
          {
            loader: 'ify-loader',
            test: /\.js$/,
          },
        ],
      },
    },
    plugins: [
      new DefinePlugin({
        'process.env.IS_DEV': JSON.stringify(IS_DEV),
        'process.env.SERVER_ADDRESS': JSON.stringify(process.env.SERVER_ADDRESS),
        'process.env.VERSION': '"0.17.6"',
      }),
      /*
       * Add theme override support for antd. For more options:
       * https://github.com/mzohaibqc/antd-theme-webpack-plugin
       */
      new AntDesignThemePlugin({
        antDir: path.join(__dirname, './node_modules/antd'),
        indexFileName: 'index.html',
        mainLessFile: path.join(__dirname, './src/styles/index.less'),
        publicPath: process.env.PUBLIC_URL,
        stylesDir: path.join(__dirname, './src/styles'),
        themeVariables: [ '@primary-color' ],
        varFile: path.join(__dirname, './src/styles/variables.less'),
      }),
      /*
       * Replace momentjs with dayjs to reduce antd package size.
       * https://github.com/ant-design/antd-dayjs-webpack-plugin
       */
      new AntdDayjsWebpackPlugin({
        plugins: [
          'isSameOrBefore',
          'isSameOrAfter',
          'advancedFormat',
          'customParseFormat',
          'weekday',
          'weekYear',
          'weekOfYear',
          'isMoment',
          'localeData',
          'localizedFormat',
        ],
        replaceMoment: true,
      }),
      /* Available MonacoWebpackPlugin options are documented at:
       * https://github.com/Microsoft/monaco-editor-webpack-plugin#options
       */
      new MonacoWebpackPlugin({
        features: [
          'codelens',
          'colorDetector',
          'find',
          'parameterHints',
          'quickOutline',
          'suggest',
          'wordHighlighter',
        ],
        languages: [ 'markdown', 'yaml' ],
      }),
    ],
  },
};
