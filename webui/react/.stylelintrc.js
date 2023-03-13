module.exports = {
  extends: ['stylelint-config-standard', 'stylelint-config-standard-scss'],
  plugins: ['stylelint-order', 'stylelint-scss'],
  rules: {
    'at-rule-no-unknown': null,
    'custom-property-empty-line-before': 'never',
    'declaration-empty-line-before': 'never',
    'declaration-property-value-no-unknown': true,
    'function-name-case': 'lower',
    'keyframes-name-pattern': null,
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
    'selector-not-notation': 'simple',
    'selector-pseudo-class-no-unknown': [true, { ignorePseudoClasses: ['global'] }],
    'value-keyword-case': null,
  },
};
