const wp = require('@cypress/webpack-preprocessor');

const experiments = require('../../utils/experiments');

// This function is called when a project is opened or re-opened (e.g. due to
// the project's config changing)

module.exports = on => {
  const options = {
    webpackOptions: require('../webpack.config'),
  };
  // `on` is used to hook into various events Cypress emits
  // `config` is the resolved Cypress config
  on('task', {
    createExperiment: experiments.create,
  });
  on('file:preprocessor', wp(options));
};
