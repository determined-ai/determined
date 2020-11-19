const reactEslint = require('../react/.eslintrc');

const UNSUPPORTED_RULES = [ '^react*' ];
const rules = Object.keys(reactEslint.rules).reduce((acc, cur) => {
  if (UNSUPPORTED_RULES.find(search => RegExp(search).test(cur))) return acc;
  acc[cur] = reactEslint.rules[cur];
  return acc;
}, {});
delete reactEslint.settings;

module.exports = {
  ...reactEslint,
  env: {
    ...reactEslint.env,
    'cypress/globals': true,
  },
  extends: [
    'plugin:cypress/recommended',
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',

  ],
  plugins: [
    'cypress',
    'import',
    'sort-keys-fix',
  ],
  rules: {
    ...rules,
    // disable until https://github.com/cypress-io/eslint-plugin-cypress/issues/43 is resolved.
    'cypress/no-unnecessary-waiting': 'off',
  },
};
