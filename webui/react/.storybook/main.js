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
  stories: ['../src/**/*.stories.@(ts|tsx)'],
};
