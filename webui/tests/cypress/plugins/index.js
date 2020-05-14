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
  on('task', { createExperiment: experiments.create });
  on('file:preprocessor', wp(options));

  require('cypress-log-to-output').install(on, (type, event) => {
    /*
     * return true or false from this plugin to control if the event
     * is logged `type` is either `console` or `browser`
     * if `type` is `browser`, `event` is an object of the type `LogEntry`:
     * https://chromedevtools.github.io/devtools-protocol/tot/Log#type-LogEntry
     * if `type` is `console`, `event` is an object of the type passed
     * to `Runtime.consoleAPICalled`:
     * https://chromedevtools.github.io/devtools-protocol/tot/Runtime#event-consoleAPICalled
     */

    if (process.env.DISCREET_LOGS && (event.level === 'error' || event.type === 'error')) {
      return true;
    }

    return false;
  });
};
