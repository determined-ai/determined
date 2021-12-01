/* eslint-disable @typescript-eslint/no-var-requires */
const { createJestConfig } = require('@craco/craco');

const cracoConfig = require('./craco.config');
const jestConfig = createJestConfig(cracoConfig);

const updatedConfig = {
  ...jestConfig,
  reporters: [
    'default',
    'jest-junit',
  ],
};

module.exports = updatedConfig;
