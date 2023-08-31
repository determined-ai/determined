import * as utils from 'utils/browser';

describe('Browser Utilities', () => {
  describe('cookies', () => {
    const cookieKeyValues = [
      { key: 'Closed', value: '2021-03-30T16:39:43.159Z' },
      { key: 'Id', value: 'NMRjx7JvgF' },
      { key: 'Datetime', value: 'Tue+Mar+30+2021+10%3A39%3A43+GMT-0600+(Mountain+Daylight+Time)' },
      { key: 'Version', value: '7.4.1' },
      { key: 'Hosts', value: '' },
      { key: 'Groups', value: 'C0003%3A1%2CC0004%3A1%2CC0002%3A1%2CC0001%3A1' },
    ];

    beforeAll(() => {
      // Stub out `document.cookie`.
      Object.defineProperty(document, 'cookie', {
        value: cookieKeyValues.map(({ key, value }) => `${key}=${value}`).join('; '),
        writable: true,
      });
    });

    cookieKeyValues.forEach((test) => {
      it(`should extract cookie key "${test.key}" value as "${test.value}"`, () => {
        expect(utils.getCookie(test.key)).toBe(test.value || null);
      });
    });

    cookieKeyValues.forEach((test) => {
      it(`should set cookie key "${test.key}" value as "${test.value}"`, () => {
        utils.setCookie(test.key, test.value);
        expect(utils.getCookie(test.key)).toBe(test.value || null);
      });
    });
  });

  describe('correctViewportHeight', () => {
    it('should correct viewport height', () => {
      // Stub out `window.innerHeight`.
      global.innerHeight = 1024;
      utils.correctViewportHeight();

      const vhProp = document.documentElement.style.getPropertyValue('--vh');
      expect(vhProp).toBe('10.24px');
    });
  });
});
