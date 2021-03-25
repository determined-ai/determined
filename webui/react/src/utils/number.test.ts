import { findFactorOfNumber, isPercent, percent, percentToFloat, roundToPrecision } from './number';

describe('number utility', () => {
  describe('findFactorOfNumber', () => {
    it('should find factors of number', () => {
      expect(findFactorOfNumber(-12)).toStrictEqual([ 1, 2, 3, 4, 6, 12 ]);
      expect(findFactorOfNumber(-1)).toStrictEqual([ 1 ]);
      expect(findFactorOfNumber(0)).toStrictEqual([ ]);
      expect(findFactorOfNumber(1)).toStrictEqual([ 1 ]);
      expect(findFactorOfNumber(12)).toStrictEqual([ 1, 2, 3, 4, 6, 12 ]);
      expect(findFactorOfNumber(1093)).toStrictEqual([ 1, 1093 ]);
      expect(findFactorOfNumber(12345)).toStrictEqual([ 1, 3, 5, 15, 823, 2469, 4115, 12345 ]);
    });
  });

  describe('isPercent', () => {
    it('should return true for percents', () => {
      expect(isPercent('100%')).toBe(true);
      expect(isPercent('0%')).toBe(true);
      expect(isPercent('24.6%')).toBe(true);
      expect(isPercent('.8269%')).toBe(true);
    });

    it('should return false for non-percents', () => {
      expect(isPercent(100)).toBe(false);
      expect(isPercent(0)).toBe(false);
      expect(isPercent(null)).toBe(false);
      expect(isPercent(undefined)).toBe(false);
      expect(isPercent('hello')).toBe(false);
    });
  });

  describe('percent', () => {
    it('should convert a float to percent', () => {
      expect(percent(0.523843984, 5)).toBe(52.38440);
      expect(percent(0.523843984, 4)).toBe(52.3844);
      expect(percent(0.523843984, 3)).toBe(52.384);
      expect(percent(0.523843984, 2)).toBe(52.38);
      expect(percent(0.523843984, 1)).toBe(52.4);
      expect(percent(0.523843984, 0)).toBe(52);
    });

    it('should default to 1 decimal when unspecified', () => {
      expect(percent(0.523843984)).toBe(52.4);
      expect(percent(0.03495)).toBe(3.5);
    });
  });

  describe('percentToFloat', () => {
    const tests = [
      { input: '100%', output: 1 },
      { input: '0%', output: 0 },
      { input: '24.6%', output: 0.246 },
      { input: '.8269%', output: 0.008269 },
    ];
    it('should convert a string percent to a floating point value', () => {
      tests.forEach(test => {
        expect(Math.abs(percentToFloat(test.input) - test.output)).toBeLessThan(Number.EPSILON);
      });
    });
  });

  describe('roundToPrecision', () => {
    it('should round to specified precision', () => {
      expect(roundToPrecision(0.523843984, 8)).toBe(0.52384398);
      expect(roundToPrecision(0.523843984, 7)).toBe(0.5238440);
      expect(roundToPrecision(0.523843984, 6)).toBe(0.523844);
      expect(roundToPrecision(0.523843984, 5)).toBe(0.52384);
      expect(roundToPrecision(0.523843984, 4)).toBe(0.5238);
      expect(roundToPrecision(0.523843984, 3)).toBe(0.524);
      expect(roundToPrecision(0.523843984, 2)).toBe(0.52);
      expect(roundToPrecision(0.523843984, 1)).toBe(0.5);
      expect(roundToPrecision(0.523843984, 0)).toBe(1);
    });

    it('should round to 6 precisions if unspecified', () => {
      expect(roundToPrecision(0.523843984)).toBe(0.523844);
    });
  });
});
