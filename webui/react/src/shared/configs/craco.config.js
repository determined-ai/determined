/* eslint-disable @typescript-eslint/no-var-requires */

const path = require('path');

const { when } = require('@craco/craco');
const CracoSassResoucesPlugin = require('craco-sass-resources-loader');
const CspHtmlWebpackPlugin = require('csp-html-webpack-plugin');
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

const IS_DEV = process.env.DET_NODE_ENV === 'development';

module.exports = {
  eslint: { enable: false },
  /** custom development flag */
  isDev: IS_DEV,
  plugins: [
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
      new CspHtmlWebpackPlugin(
        {
          'frame-src': "'self' netlify.determined.ai",
          'object-src': "'none'",
          'script-src': "'self' cdn.segment.com",
          'style-src': "'self' 'unsafe-inline'",
        },
        {
          enabled: true,
          hashEnabled: {
            'script-src': true,
            'style-src': false,
          },
          nonceEnabled: {
            'script-src': false,
            'style-src': false,
          },
        },
      ),
    ],
  },
};
