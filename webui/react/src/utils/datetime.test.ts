import * as utils from './datetime';

describe('Datetime Utilities', () => {
  describe('getDuration', () => {
    const DATES = ['2021-11-11 01:01:01Z', '2021-11-11 11:11:11Z', 'Nov 11, 2021 11:11:11Z'];

    [
      { input: { endTime: DATES[1], startTime: DATES[0] }, output: 36610000 },
      { input: { endTime: DATES[0], startTime: DATES[1] }, output: -36610000 },
      { input: { endTime: DATES[2], startTime: DATES[0] }, output: 36610000 },
    ].forEach((test) => {
      const { endTime: end, startTime: start } = test.input;
      it(`should get duration from start "${start}" and end "${end}"`, () => {
        expect(utils.getDuration(test.input)).toBe(test.output);
      });
    });

    it('should use now as default end time for duration', () => {
      const EPSILON = 5000;
      const now = new Date().toUTCString();
      const expected = utils.getDuration({ endTime: now, startTime: DATES[0] });
      const actual = utils.getDuration({ startTime: DATES[0] });
      expect(Math.abs(actual - expected)).toBeLessThan(EPSILON);
    });
  });

  describe('durationInEnglish', () => {
    const tests = [
      { input: 100, output: '100ms' },
      { input: 1300, output: '1s 300ms' },
      { input: 32000, output: '32s' },
      { input: 60000, output: '1m' },
      { input: 119000, output: '1m 59s' },
      { input: 10800000, output: '3h' },
      { input: 12600000, output: '3h 30m' },
      { input: 86400000, output: '1d' },
      { input: 129600000, output: '1d 12h' },
      { input: 604800000, output: '1w' },
      { input: 2592000000, output: '4w 2d' },
      { input: 3024000000, output: '1mo 1w' },
      { input: 31536000000, output: '1y' },
    ];
    tests.forEach((test) => {
      it(`should humanize duration of ${test.input} seconds to english "${test.output}"`, () => {
        expect(utils.durationInEnglish(test.input)).toBe(test.output);
      });
    });
  });

  describe('secondToHour', () => {
    const tests = [
      { input: 36, output: 0.01 },
      { input: 3600, output: 1 },
      { input: 87300, output: 24.25 },
    ];
    tests.forEach((test) => {
      it(`should convert ${test.input} second(s) to ${test.output} hour(s)`, () => {
        expect(utils.secondToHour(test.input)).toBe(test.output);
      });
    });
  });
});
