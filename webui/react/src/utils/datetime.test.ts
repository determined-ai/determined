import * as utils from './datetime';

describe('datetime utilities', () => {
  describe('formatDatetime', () => {
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
      it(`should format "${test.input}" into standard format`, () => {
        expect(utils.formatDatetime(test.input)).toBe(test.output);
      });
    });

    [
      { input: [ DATE[0], FORMAT[0] ], output: 'Nov 23, 2021 05:59am' },
      { input: [ DATE[1], FORMAT[1] ], output: 'December 31 (Wednesday)' },
    ].forEach(test => {
      it(`should format "${test.input[0]}" with format "${test.input[1]}"`, () => {
        expect(utils.formatDatetime(test.input[0], test.input[1])).toBe(test.output);
      });
    });

    [
      { input: [ DATE[2], FORMAT[0] ], output: 'Nov 10, 2021 11:01am' },
      { input: [ DATE[3], FORMAT[0] ], output: 'Nov 11, 2021 11:11am' },
    ].forEach(test => {
      it(`should format "${test.input[0]}" with format "${test.input[1]}" in local time`, () => {
        expect(utils.formatDatetime(test.input[0], test.input[1], false)).toBe(test.output);
      });
    });
  });
});
