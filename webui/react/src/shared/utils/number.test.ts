import * as utils from './number';

describe('Number Utilities', () => {
  describe('clamp', () => {
    it('should clamp in-range value within a min/max range', () => {
      expect(utils.clamp(0, -1, 1)).toBe(0);
      expect(utils.clamp(1e4, 1e3, 1e5)).toBe(1e4);
    });

    it('should clamp out-of-range value within a min/max range', () => {
      expect(utils.clamp(-2, -1, 1)).toBe(-1);
      expect(utils.clamp(2, -1, 1)).toBe(1);
      expect(utils.clamp(1e2, 1e3, 1e5)).toBe(1e3);
      expect(utils.clamp(1e6, 1e3, 1e5)).toBe(1e5);
    });
  });

  describe('findFactorOfNumber', () => {
    it('should find factors of number', () => {
      expect(utils.findFactorOfNumber(-12)).toStrictEqual([ 1, 2, 3, 4, 6, 12 ]);
      expect(utils.findFactorOfNumber(-1)).toStrictEqual([ 1 ]);
      expect(utils.findFactorOfNumber(0)).toStrictEqual([ ]);
      expect(utils.findFactorOfNumber(1)).toStrictEqual([ 1 ]);
      expect(utils.findFactorOfNumber(12)).toStrictEqual([ 1, 2, 3, 4, 6, 12 ]);
      expect(utils.findFactorOfNumber(1093)).toStrictEqual([ 1, 1093 ]);
      expect(utils.findFactorOfNumber(12345)).toStrictEqual([
        1, 3, 5, 15, 823, 2469, 4115, 12345,
      ]);
    });
  });

  describe('isPercent', () => {
    it('should return true for percents', () => {
      expect(utils.isPercent('100%')).toBe(true);
      expect(utils.isPercent('0%')).toBe(true);
      expect(utils.isPercent('24.6%')).toBe(true);
      expect(utils.isPercent('.8269%')).toBe(true);
    });

    it('should return false for non-percents', () => {
      expect(utils.isPercent(100)).toBe(false);
      expect(utils.isPercent(0)).toBe(false);
      expect(utils.isPercent(null)).toBe(false);
      expect(utils.isPercent(undefined)).toBe(false);
      expect(utils.isPercent('hello')).toBe(false);
    });
  });

  describe('percent', () => {
    it('should convert a float to percent', () => {
      expect(utils.percent(0.523843984, 5)).toBe(52.38440);
      expect(utils.percent(0.523843984, 4)).toBe(52.3844);
      expect(utils.percent(0.523843984, 3)).toBe(52.384);
      expect(utils.percent(0.523843984, 2)).toBe(52.38);
      expect(utils.percent(0.523843984, 1)).toBe(52.4);
      expect(utils.percent(0.523843984, 0)).toBe(52);
    });

    it('should default to 1 decimal when unspecified', () => {
      expect(utils.percent(0.523843984)).toBe(52.4);
      expect(utils.percent(0.03495)).toBe(3.5);
    });

    it('should convert NaN, Infinity and -Infinity to percent', () => {
      expect(utils.percent(NaN)).toBe(0);
      expect(utils.percent(Infinity)).toBe(100);
      expect(utils.percent(-Infinity)).toBe(0);
    });
  });

  describe('percentToFloat', () => {
    const tests = [
      { input: '100%', output: 1 },
      { input: '0%', output: 0 },
      { input: '24.6%', output: 0.246 },
      { input: '.8269%', output: 0.008269 },
      { input: NaN, output: 1.0 },
    ];
    it('should convert a string percent to a floating point value', () => {
      tests.forEach(test => {
        const result = Math.abs(utils.percentToFloat(test.input) - test.output);
        expect(result).toBeLessThan(Number.EPSILON);
      });
    });
  });

  describe('roundToPrecision', () => {
    it('should round to specified precision', () => {
      expect(utils.roundToPrecision(0.523843984, 8)).toBe(0.52384398);
      expect(utils.roundToPrecision(0.523843984, 7)).toBe(0.5238440);
      expect(utils.roundToPrecision(0.523843984, 6)).toBe(0.523844);
      expect(utils.roundToPrecision(0.523843984, 5)).toBe(0.52384);
      expect(utils.roundToPrecision(0.523843984, 4)).toBe(0.5238);
      expect(utils.roundToPrecision(0.523843984, 3)).toBe(0.524);
      expect(utils.roundToPrecision(0.523843984, 2)).toBe(0.52);
      expect(utils.roundToPrecision(0.523843984, 1)).toBe(0.5);
      expect(utils.roundToPrecision(0.523843984, 0)).toBe(1);
    });

    it('should round to 6 precisions if unspecified', () => {
      expect(utils.roundToPrecision(0.523843984)).toBe(0.523844);
    });
  });
});
