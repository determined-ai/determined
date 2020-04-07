const reactEslint = require('../react/.eslintrc');

const UNSUPPORTED_RULES = ['^react*'];
const rules = Object.keys(reactEslint.rules).reduce((acc, cur) => {
  if (UNSUPPORTED_RULES.find(search => RegExp(search).test(cur))) return acc;
  acc[cur] = reactEslint.rules[cur];
  return acc;
}, {});

module.exports = {
  ...reactEslint,
  extends: [
    'plugin:cypress/recommended',
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',

  ],
  plugins: [
    'cypress', 
    'import'
  ],
  env: {
    ...reactEslint.env,
    'cypress/globals': true,
  },
  rules,
};
