module.exports = {
  env: {
    browser: true,
    commonjs: true,
    es6: true,
    node: true,
  },
  extends: [
    'eslint:recommended',
    'plugin:react/recommended',
    'plugin:jest/recommended',
    'plugin:jest/style',
    'plugin:@typescript-eslint/recommended',
    'plugin:react-hooks/recommended',
  ],
  globals: {
    Atomics: 'readonly',
    SharedArrayBuffer: 'readonly',
  },
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaFeatures: { jsx: true },
    ecmaVersion: 2018,
    sourceType: 'module',
  },
  plugins: [ 'import', 'react', 'react-hooks', 'sort-keys-fix' ],
  rules: {
    '@typescript-eslint/explicit-module-boundary-types': [ 'warn', {
      allowArgumentsExplicitlyTypedAsAny: true,
    } ],
    '@typescript-eslint/indent': [ 'error', 2 ],
    '@typescript-eslint/no-unused-vars': 'error',
    'array-bracket-spacing': [ 'error', 'always' ],
    'array-element-newline': [ 'error', 'consistent' ],
    'block-spacing': [ 'error', 'always' ],
    'brace-style': [ 'error', '1tbs', { allowSingleLine: true } ],
    'comma-dangle': [ 'error', 'always-multiline' ],
    'eol-last': [ 'error', 'always' ],
    'eqeqeq': [ 'error', 'smart' ],
    'import/order': [ 'error', {
      'alphabetize': { caseInsensitive: true, order: 'asc' },
      'groups': [ 'builtin', 'external', 'internal', 'parent', 'sibling', 'index' ],
      'newlines-between': 'always',
    } ],
    'indent': 'off',
    'max-len': [ 'error', 100, { tabWidth: 2 } ],
    'no-console': [ 'error', { allow: [ 'warn' ] } ],
    'no-multi-spaces': [ 'error', { ignoreEOLComments: true } ],
    'no-multiple-empty-lines': [ 'error', { max: 1, maxBOF: 0, maxEOF: 0 } ],
    'no-trailing-spaces': [ 'error', {} ],
    'object-curly-spacing': [ 'error', 'always' ],
    'object-property-newline': [ 'error', { allowAllPropertiesOnSameLine: true } ],
    'quote-props': [ 'error', 'consistent-as-needed' ],
    'quotes': [ 'error', 'single', { avoidEscape: true } ],
    'react/jsx-sort-props': [ 'error', {
      callbacksLast: true,
      ignoreCase: true,
    } ],
    'react/jsx-tag-spacing': [ 'error', {
      afterOpening: 'never',
      beforeClosing: 'never',
      beforeSelfClosing: 'always',
      closingSlash: 'never',
    } ],
    'react/prop-types': 'off',
    'react/self-closing-comp': [ 'error', {
      component: true,
      html: true,
    } ],
    'require-await': 'error',
    'semi': [ 'error', 'always' ],
    'sort-imports': [ 'error', {
      ignoreCase: true,
      ignoreDeclarationSort: true,
      ignoreMemberSort: false,
    } ],
    'sort-keys-fix/sort-keys-fix': [ 'error', 'asc', {
      caseSensitive: false,
      natural: true,
    } ],
  },
  settings: {
    'import/resolver': {
      typescript: {}, // This loads <rootdir>/tsconfig.json to eslint
    },
    'react': { version: 'detect' },
  },
};
