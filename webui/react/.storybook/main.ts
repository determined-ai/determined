import type { StorybookConfig } from '@storybook/react/types';
import { isString } from 'fp-ts/lib/string';
import type { Configuration, RuleSetRule } from 'webpack';

const config: StorybookConfig = {
  addons: [
    'storybook-preset-craco',
    {
      name: '@storybook/addon-docs',
      options: { configureJSX: true },
    },
    '@storybook/addon-links',
    '@storybook/addon-postcss',
    '@storybook/addon-essentials',
  ],
  features: {
    buildStoriesJson: true,
    modernInlineRender: true,
    storyStoreV7: true,
  },
  framework: '@storybook/react',
  staticDirs: [
    '../public',
    { from: '../src/shared/assets', to: '/assets' },
  ],
  stories: [ { directory: '../src' } ],
  webpackFinal: (config: Configuration) => {
    if (process.env.NODE_ENV !== 'production') return config;
    /*
     * Tweak `file-loader` to fix the css `url()` references for
     * `storybook:build` only. Without this adjustment, `build-storybook`
     * looks for the fonts in `static/css/static/media` instead of the
     * proper location of `static/media`.
     */
    const oneOfs = (config?.module?.rules
      ?.find((rule) => isString(rule) ? false : !!rule.oneOf) as RuleSetRule)?.oneOf;
    const fontMatcher = /\.woff2?$/;

    // Exclude fonts from default file-loader.
    const fileLoader = oneOfs
      ?.find((oneOf) => /file-loader/.test(oneOf?.loader?.toString() ?? '')) ?? {};
    if (Array.isArray(fileLoader.exclude)) fileLoader.exclude.push(fontMatcher);
    else if (fileLoader.exclude) fileLoader.exclude = [ fileLoader.exclude, fontMatcher ];
    else fileLoader.exclude = [ fontMatcher ];

    // Add a new file-loader to handle just fonts.
    const fileLoaderFont = {
      loader: fileLoader?.loader,
      options: {
        esModule: false,
        name: '[name].[ext]',
        outputPath: 'static/media',
        publicPath: '../media',
      },
      test: fontMatcher,
    };
    oneOfs?.push(fileLoaderFont);

    const maxAssetSize = 1024 * 1024;

    // split into more chunks
    config.optimization = {
      splitChunks: {
        chunks: 'all',
        // 30KB
        maxSize: maxAssetSize,
        minSize: 30 * 1024, // 1MB
      },
    };
    config.performance = { maxAssetSize: maxAssetSize };

    return config;
  },
};

module.exports = config;
