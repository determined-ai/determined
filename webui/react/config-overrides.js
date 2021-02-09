/* eslint-disable */
const {
  addLessLoader,
  addWebpackPlugin,
  adjustStyleLoaders,
  disableEsLint,
  override,
  fixBabelImports,
} = require('customize-cra');
const AntdDayjsWebpackPlugin = require('antd-dayjs-webpack-plugin');
const AntDesignThemePlugin = require('antd-theme-webpack-plugin');
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');
const path = require('path');
const webpack = require('webpack');
const jestConfig = require('./jest.config');

const IS_DEV = process.env.DET_NODE_ENV === 'development';

function customOverride(config, env) {
  let configPatch = {};
  if (IS_DEV) {
    configPatch = {
      mode: 'development',
      // remove webpack optimizations to lower build time.
      optimization: {},
    }
  }
  return {
    ...config,
    ...configPatch
  }
}

const webpackConfig = override(
  // Disable eslint for webpack config.
  disableEsLint(),

  // Support for import on demand for antd.
  fixBabelImports('import', {
    libraryName: 'antd',
    libraryDirectory: 'es',
    style: true,
  }),

  // Add LESS loader support for antd.
  addLessLoader({ lessOptions: { javascriptEnabled: true } }),

  // Add support for global SASS/SCSS file.
  adjustStyleLoaders(({ use: [ , css, postcss, resolve, processor ] }) => {
    if (processor && processor.loader.includes('sass-loader')) {
      processor.options = processor.options || {};
      processor.options.prependData = '@import \'global.scss\';';
      processor.options.sassOptions = processor.options.sassOptions || {};
      processor.options.sassOptions.includePaths = processor.options.sassOptions.includePaths || [];
      processor.options.sassOptions.includePaths.push(path.join(__dirname, './src/styles'));
    }
  }),

  // Replace momentjs to Day.js to reduce antd package size.
  addWebpackPlugin(new AntdDayjsWebpackPlugin({
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
  })),

  /*
   * Add theme override support for antd. For more options.
   * https://github.com/mzohaibqc/antd-theme-webpack-plugin
   */
  addWebpackPlugin(
    new AntDesignThemePlugin({
      antDir: path.join(__dirname, './node_modules/antd'),
      stylesDir: path.join(__dirname, './src/styles'),
      varFile: path.join(__dirname, './src/styles/variables.less'),
      mainLessFile: path.join(__dirname, './src/styles/index.less'),
      themeVariables: [
        '@primary-color',
        // TODO: Near future, add more colors to override in browser dynamically.
      ],
      indexFileName: 'index.html',
      publicPath: process.env.PUBLIC_URL,
    })
  ),

  // Webapp version is hardcoded but handled by `bumpversion`
  addWebpackPlugin(
    new webpack.DefinePlugin({
      'process.env.VERSION': '"0.14.1rc1"',
      'process.env.IS_DEV': JSON.stringify(IS_DEV),
      'process.env.SERVER_ADDRESS': JSON.stringify(process.env.SERVER_ADDRESS),
    })
  ),

  addWebpackPlugin(
    new MonacoWebpackPlugin({
      // available options are documented at https://github.com/Microsoft/monaco-editor-webpack-plugin#options
      languages: ['yaml'],
      features: [
        'codelens',
        'colorDetector',
        'find',
        'parameterHints',
        'quickOutline',
        'suggest',
        'wordHighlighter',
      ],
    })
  ),
);

module.exports = {
  webpack: (config, env) => {
    const customCraConfig = webpackConfig(config, env);
    return customOverride(customCraConfig);
  },
  jest: (config, env) => ({...config, ...jestConfig}),
  // devServer: (config, env) => config,
}
