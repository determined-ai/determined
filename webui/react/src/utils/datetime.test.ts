import dayjs from 'dayjs';

import * as utils from './datetime';

describe('Datetime Utilities', () => {
  describe('formatDatetime', () => {
    const SYSTEM_UTC_OFFSET = dayjs().utcOffset();
    const DATE = [
      '2021-11-23T05:59:59.500Z',
      'December 31, 1980 23:59:59.999Z',
      '2021-11-11T01:01:01.000+07:00',
      '2021-11-11T11:11:11.000-07:00',
    ];
    const FORMAT = [
      'MMM DD, YYYY HH:mma',
      'MMMM DD (dddd)',
    ];

    [
      { input: DATE[0], output: '2021-11-23, 05:59:59' },
      { input: DATE[1], output: '1980-12-31, 23:59:59' },
    ].forEach(test => {
      it(`should format "${test.input}" as default format`, () => {
        expect(utils.formatDatetime(test.input)).toBe(test.output);
      });
    });

    [
      {
        input: { date: DATE[0], options: { format: FORMAT[0] } },
        output: 'Nov 23, 2021 05:59am',
      },
      {
        input: { date: DATE[1], options: { format: FORMAT[1] } },
        output: 'December 31 (Wednesday)',
      },
    ].forEach(test => {
      const { date, options } = test.input;
      it(`should format "${date}" as "${options.format}" in UTC`, () => {
        expect(utils.formatDatetime(date, options)).toBe(test.output);
      });
    });

    [
      {
        input: { date: DATE[2], options: { format: FORMAT[0], outputUTC: false } },
        output: dayjs.utc(DATE[2]).add(SYSTEM_UTC_OFFSET, 'minute').format(FORMAT[0]),
      },
      {
        input: { date: DATE[3], options: { format: FORMAT[1], outputUTC: false } },
        output: dayjs.utc(DATE[3]).add(SYSTEM_UTC_OFFSET, 'minute').format(FORMAT[1]),
      },
    ].forEach(test => {
      const { date, options } = test.input;
      it(`should format "${date}" as "${options.format}" in local time`, () => {
        expect(utils.formatDatetime(date, options)).toBe(test.output);
      });
    });

    [
      {
        input: { date: DATE[2], options: { inputUTC: true } },
        output: dayjs
          .utc(utils.stripTimezone(DATE[2]))
          .format(utils.DEFAULT_DATETIME_FORMAT),
      },
      {
        input: { date: DATE[3], options: { inputUTC: true, outputUTC: false } },
        output: dayjs
          .utc(utils.stripTimezone(DATE[3]))
          .add(SYSTEM_UTC_OFFSET, 'minute')
          .format(utils.DEFAULT_DATETIME_FORMAT),
      },
    ].forEach(test => {
      const { date, options } = test.input;
      const resultFormat = options.outputUTC ? 'UTC' : 'local time';
      it(`should read "${date}" as UTC and format as ${resultFormat}`, () => {
        expect(utils.formatDatetime(date, options)).toBe(test.output);
      });
    });
  });

  describe('getDuration', () => {
    const DATES = [
      '2021-11-11 01:01:01Z',
      '2021-11-11 11:11:11Z',
      'Nov 11, 2021 11:11:11Z',
    ];

    [
      { input: { endTime: DATES[1], startTime: DATES[0] }, output: 36610000 },
      { input: { endTime: DATES[0], startTime: DATES[1] }, output: -36610000 },
      { input: { endTime: DATES[2], startTime: DATES[0] }, output: 36610000 },
    ].forEach(test => {
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
    tests.forEach(test => {
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
    tests.forEach(test => {
      it(`should convert ${test.input} second(s) to ${test.output} hour(s)`, () => {
        expect(utils.secondToHour(test.input)).toBe(test.output);
      });
    });
  });

  describe('stripTimezone', () => {
    it('should strip timezone from datetime strings', () => {
      const tests = [
        { input: '2021-11-11T00:00:00', output: '2021-11-11T00:00:00' },
        { input: '2021-11-11T00:00:00Z', output: '2021-11-11T00:00:00' },
        { input: '2021-11-11T00:00:00+11:11', output: '2021-11-11T00:00:00' },
        { input: '2021-11-11T00:00:00-05:05', output: '2021-11-11T00:00:00' },
      ];
      tests.forEach(test => {
        expect(utils.stripTimezone(test.input)).toBe(test.output);
      });
    });
  });
});
