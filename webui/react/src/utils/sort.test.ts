import { stringToVersion } from 'utils/string';

import * as sorters from './sort';

interface SortTest {
  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  input: any[];
  output: number;
}

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
type GenericSorter = (a: any, b: any) => number;

describe('sort utility', () => {
  const runSortTests = (tests: SortTest[], sorter: GenericSorter) => {
    tests.forEach((test) => {
      it(`compare "${test.input[0]}" with "${test.input[1]}"`, () => {
        const result = sorter(test.input[0], test.input[1]);
        expect(result).toStrictEqual(test.output);
      });
    });
  };

  describe('alphaNumericSorter', () => {
    const tests = [
      { input: ['Jumping', 'elephant'], output: 1 },
      { input: [52, 'elephant'], output: -1 },
      { input: ['elephant', -12], output: 1 },
      { input: [52, -12], output: 1 },
      { input: [-12, 52], output: -1 },
      { input: ['abc', 'abc'], output: 0 },
      { input: ['ABC', 'abc'], output: 1 },
      { input: ['abc', 'ABC'], output: -1 },
      { input: [-12, -12], output: 0 },
      { input: [52, 52], output: 0 },
      { input: [0.123, undefined], output: -1 },
      { input: [0, undefined], output: -1 },
      { input: [undefined, -0.123], output: 1 },
      { input: [0.123, null], output: -1 },
      { input: [0, null], output: -1 },
      { input: [null, -0.123], output: 1 },
      { input: ['ABC', undefined], output: -1 },
      { input: ['abc', undefined], output: -1 },
      { input: [undefined, 'ABC'], output: 1 },
      { input: [undefined, 'abc'], output: 1 },
      { input: ['ABC', null], output: -1 },
      { input: ['abc', null], output: -1 },
      { input: [null, 'ABC'], output: 1 },
      { input: [null, 'abc'], output: 1 },
    ];
    runSortTests(tests, sorters.alphaNumericSorter);
  });

  describe('booleanSorter', () => {
    const tests = [
      { input: [true, true], output: 0 },
      { input: [false, false], output: 0 },
      { input: [true, false], output: -1 },
      { input: [false, true], output: 1 },
      { input: [true, undefined], output: -1 },
      { input: [false, undefined], output: -1 },
      { input: [true, null], output: -1 },
      { input: [false, null], output: -1 },
      { input: [undefined, true], output: 1 },
      { input: [undefined, false], output: 1 },
      { input: [null, true], output: 1 },
      { input: [null, false], output: 1 },
    ];
    runSortTests(tests, sorters.booleanSorter);
  });

  describe('dateTimeStringSorter', () => {
    const dateString = ['2021-01-01T12:59:59Z', '2021-01-01T13:00:00Z', '2021-12-31T00:00:00Z'];
    const tests = [
      { input: [dateString[0], dateString[0]], output: 0 },
      { input: [dateString[1], dateString[1]], output: 0 },
      { input: [dateString[0], dateString[1]], output: -1 },
      { input: [dateString[0], dateString[2]], output: -1 },
      { input: [dateString[2], dateString[1]], output: 1 },
      { input: [dateString[2], dateString[0]], output: 1 },
      { input: [dateString[0], undefined], output: -1 },
      { input: [dateString[0], null], output: -1 },
      { input: [undefined, dateString[1]], output: 1 },
      { input: [null, dateString[1]], output: 1 },
    ];
    runSortTests(tests, sorters.dateTimeStringSorter);
  });

  describe('numericSorter', () => {
    const tests = [
      { input: [0, 1], output: -1 },
      { input: [1, 0], output: 1 },
      { input: [0, -0.123], output: 1 },
      { input: [-0.123, 0], output: -1 },
      { input: [100, 10], output: 1 },
      { input: [-100, -10], output: -1 },
      { input: [1, 10], output: -1 },
      { input: [-1, -10], output: 1 },
      { input: [0.01, 0.1], output: -1 },
      { input: [-0.01, -0.1], output: 1 },
      { input: [1.23e2, -123], output: 1 },
      { input: [-0.999, 9e-3], output: -1 },
      { input: [0.123, undefined], output: -1 },
      { input: [0, undefined], output: -1 },
      { input: [undefined, -0.123], output: 1 },
      { input: [undefined, 0], output: 1 },
      { input: [0.123, null], output: -1 },
      { input: [0, null], output: -1 },
      { input: [null, -0.123], output: 1 },
      { input: [null, 0], output: 1 },
      { input: [0, 0], output: 0 },
      { input: [1e7, 1e7], output: 0 },
      { input: [1e-5, 1e-5], output: 0 },
      { input: [10, 10], output: 0 },
      { input: [-0.123, -0.123], output: 0 },
    ];
    runSortTests(tests, sorters.numericSorter);
  });

  describe('nullSorter', () => {
    const tests = [
      { input: [null, null], output: 0 },
      { input: [null, undefined], output: 0 },
      { input: [undefined, null], output: 0 },
      { input: [undefined, undefined], output: 0 },
      { input: ['abc', null], output: -1 },
      { input: ['abc', undefined], output: -1 },
      { input: [null, 'abc'], output: 1 },
      { input: [undefined, 'abc'], output: 1 },
      { input: [-0.123, null], output: -1 },
      { input: [-0.123, undefined], output: -1 },
      { input: [null, -0.123], output: 1 },
      { input: [undefined, -0.123], output: 1 },
      { input: ['abc', -0.123], output: 0 },
    ];
    runSortTests(tests, sorters.nullSorter);
  });

  describe('sortVersions', () => {
    const tests = [{ input: ['1.2.3', '1.2.4'], output: ['1.2.4', '1.2.3'] }];
    tests.forEach((t) => {
      it('should sort latest first', () => {
        expect(sorters.sortVersions(t.input.map(stringToVersion))).toStrictEqual(
          t.output.map(stringToVersion),
        );
      });
    });
  });
  describe('primitiveSorter', () => {
    const tests = [
      { input: ['abc', 'ABC'], output: -1 },
      { input: ['giraffe', 'elephant'], output: 1 },
      { input: [-1.5, 5e10], output: -1 },
      { input: [1e7, 1e-7], output: 1 },
      { input: [true, false], output: -1 },
      { input: [false, true], output: 1 },
      { input: ['abc', true], output: 0 },
      { input: [1e7, 'elephant'], output: 0 },
      { input: [false, -5], output: 0 },
    ];
    runSortTests(tests, sorters.primitiveSorter);
  });
});
