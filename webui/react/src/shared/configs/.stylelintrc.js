module.exports = {
  overrides: [
    {
      customSyntax: 'postcss-scss',
      extends: [
        'stylelint-config-standard',
        'stylelint-config-standard-scss',
        'stylelint-config-prettier', // stylelint-config-prettier should be the last
      ],
      files: ['**/*.scss'],
    },
    {
      customSyntax: 'postcss-less',
      extends: [
        'stylelint-config-standard',
        'stylelint-config-recommended-less',
        'stylelint-config-prettier', // stylelint-config-prettier should be the last
      ],
      files: ['**/*.less'],
    },
  ],
  plugins: ['stylelint-order', 'stylelint-scss', 'stylelint-less'],
  rules: {
    'at-rule-no-unknown': null,
    'at-rule-semicolon-space-before': 'never',
    'custom-property-empty-line-before': 'never',
    'declaration-block-semicolon-newline-after': 'always-multi-line',
    'declaration-block-semicolon-newline-before': 'never-multi-line',
    'declaration-block-semicolon-space-before': 'never',
    'declaration-block-trailing-semicolon': null,
    'declaration-empty-line-before': 'never',
    'function-name-case': 'lower',
    'keyframes-name-pattern': null,
    'no-eol-whitespace': [true, { ignore: ['empty-lines'] }],
    'no-extra-semicolons': true,
    'order/order': [
      'custom-properties',
      'dollar-variables',
      'at-variables',
      'declarations',
      'rules',
      'at-rules',
      'less-mixins',
    ],
    'order/properties-alphabetical-order': true,
    'property-no-vendor-prefix': null,
    'rule-empty-line-before': [
      'always',
      {
        except: ['after-rule', 'first-nested', 'inside-block-and-after-rule'],
      },
    ],
    'scss/at-rule-no-unknown': true,
    'selector-class-pattern': null,
    'selector-id-pattern': null,
    'selector-pseudo-class-no-unknown': [true, { ignorePseudoClasses: ['global'] }],
    'string-quotes': 'single',
    'value-keyword-case': null,
  },
};
