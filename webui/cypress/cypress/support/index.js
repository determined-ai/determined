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
import { STORAGE_KEY_AUTH } from '../constants';

const _clear = Cypress.LocalStorage.clear;
Cypress.LocalStorage.clear = function(aKeys) {
  const keysToKeep = new Set([ STORAGE_KEY_AUTH ]);
  const keys = (aKeys.length ? aKeys : Object.keys(window.localStorage))
    .filter(key => !keysToKeep.has(key));
  return _clear(keys);
};
