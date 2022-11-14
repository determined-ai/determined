/* eslint-disable @typescript-eslint/no-var-requires */

const path = require('path');

const { when } = require('@craco/craco');
const AntdDayjsWebpackPlugin = require('antd-dayjs-webpack-plugin');
const CracoLessPlugin = require('craco-less');
const CracoSassResoucesPlugin = require('craco-sass-resources-loader');
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

const IS_DEV = process.env.DET_NODE_ENV === 'development';

module.exports = {
  babel: {
    plugins: [
      [
        'import',
        {
          libraryDirectory: 'es',
          libraryName: 'antd',
          style: true,
        },
      ],
    ],
  },
  eslint: { enable: false },
  /** custom development flag */
  isDev: IS_DEV,
  plugins: [
    {
      options: { lessLoaderOptions: { lessOptions: { javascriptEnabled: true } } },
      plugin: CracoLessPlugin,
    },
    {
      options: { resources: path.join(__dirname, '../styles/global.scss') },
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
    plugins: [
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
        languages: ['markdown', 'yaml', 'python'],
      }),
    ],
  },
};
