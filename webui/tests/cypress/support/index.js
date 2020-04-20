// ***********************************************************
// This example support/index.js is processed and
// loaded automatically before your test files.
//
// This is a great place to put global configuration and
// behavior that modifies Cypress.
//
// You can change the location of this file or turn off
// automatically serving support files with the
// 'supportFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/configuration
// ***********************************************************

import './commands';

Cypress.Cookies.defaults({
  whitelist: /auth/,
});

Cypress.on('window:before:load', (window) => {
  Cypress.log({
    message: 'wrap on console.log',
    name: 'console.log',
  });

  // pass through cypress log so we can see log inside command execution order
  window.console.log = (...args) => {
    Cypress.log({
      message: args,
      name: 'console.log',
    });
  };

  // disable actions that would result in opening new tabs/windows
  // https://docs.cypress.io/guides/references/trade-offs.html#Permanent-trade-offs-1
  window.open = () => {};

}); // end of before:load

Cypress.on('log:added', (options) => {
  if (options.instrument === 'command') {
    // eslint-disable-next-line no-console
    console.log(
      `${(options.displayName || options.name || '').toUpperCase()} ${options.message}`,
    );
  }
});
