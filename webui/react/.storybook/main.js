module.exports = {
  addons: [
    {
      name: 'storybook-preset-craco',
      options: {
        cracoConfigFile: '../../craco.config.js',
      },
    },
    {
      name: '@storybook/addon-docs',
      options: {
        configureJSX: true,
      },
    },
    '@storybook/addon-actions',
    '@storybook/addon-backgrounds',
    '@storybook/addon-links',
    '@storybook/addon-knobs',
    '@storybook/addon-postcss',
    '@storybook/addon-viewport',
  ],
  staticDirs: [
    '../public',
    { from: '../src/shared/assets', to: '/assets' }
  ],
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  webpackFinal: config => {
    if (process.env.NODE_ENV !== 'production') return config;
    /*
     * Tweak `file-loader` to fix the css `url()` references for
     * `storybook:build` only. Without this adjustment, `build-storybook`
     * looks for the fonts in `static/css/static/media` instead of the
     * proper location of `static/media`.
     */
    const oneOfs = config.module.rules.find(rule => !!rule.oneOf).oneOf;
    const fontMatcher = /\.woff2?$/;

    // Exclude fonts from default file-loader.
    const fileLoader = oneOfs.find(oneOf => /file-loader/.test(oneOf.loader));
    fileLoader.exclude.push(fontMatcher);

    // Add a new file-loader to handle just fonts.
    const fileLoaderFont = {
      loader: fileLoader.loader,
      options: {
        publicPath: '../media',
        outputPath: 'static/media',
        name: '[name].[ext]',
        esModule: false
      },
      test: fontMatcher,
    };
    oneOfs.push(fileLoaderFont);

    return config;
  },
};
