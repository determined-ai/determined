import { numericSorter } from './sort';

describe('sort utility', () => {
  it('numericSorter', () => {
    const tests = [
      { input: [ 0, 1 ], output: -1 },
      { input: [ 1, 0 ], output: 1 },
      { input: [ 0, -0.123 ], output: 1 },
      { input: [ -0.123, 0 ], output: -1 },
      { input: [ 100, 10 ], output: 1 },
      { input: [ -100, -10 ], output: -1 },
      { input: [ 1, 10 ], output: -1 },
      { input: [ -1, -10 ], output: 1 },
      { input: [ 0.01, 0.1 ], output: -1 },
      { input: [ -0.01, -0.1 ], output: 1 },
      { input: [ 1.23e2, -123 ], output: 1 },
      { input: [ -0.999, 9e-3 ], output: -1 },
      { input: [ 0.123, undefined ], output: 1 },
      { input: [ 0, undefined ], output: 1 },
      { input: [ undefined, -0.123 ], output: -1 },
      { input: [ undefined, 0 ], output: -1 },
      { input: [ undefined, undefined ], output: 0 },
      { input: [ 0, 0 ], output: 0 },
      { input: [ 1e7, 1e7 ], output: 0 },
      { input: [ 1e-5, 1e-5 ], output: 0 },
      { input: [ 10, 10 ], output: 0 },
      { input: [ -0.123, -0.123 ], output: 0 },
    ];
    tests.forEach(test => {
      const result = numericSorter(test.input[0], test.input[1], false);
      const reverseResult = numericSorter(test.input[0], test.input[1], true);
      expect(result).toStrictEqual(test.output);
      expect(reverseResult).toStrictEqual(test.output === 0 ? 0 : test.output * -1);
    });
  });
});
