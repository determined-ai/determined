/* eslint-disable */
const {
  addLessLoader,
  addWebpackPlugin,
  disableEsLint,
  override,
  fixBabelImports,
} = require('customize-cra');
const AntdDayjsWebpackPlugin = require('antd-dayjs-webpack-plugin');
const AntDesignThemePlugin = require('antd-theme-webpack-plugin');
const path = require('path');
const webpack = require('webpack');

module.exports = override(
  // Disable eslint for webpack config.
  disableEsLint(),

  // Support for import on demand for antd.
  fixBabelImports('import', {
    libraryName: 'antd',
    libraryDirectory: 'es',
    style: true,
  }),

  // Add LESS loader support for antd.
  addLessLoader({ javascriptEnabled: true }),

  // Replace momentjs to Day.js to reduce antd package size.
  addWebpackPlugin(new AntdDayjsWebpackPlugin()),

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
    })
  ),

  // Webapp version is hardcoded but handled by `bumpversion`
  addWebpackPlugin(
    new webpack.DefinePlugin({
      'process.env.VERSION': '"0.12.4rc0"',
      'process.env.IS_DEV': JSON.stringify(process.env.NODE_ENV === 'development'),
    })
  )
);
