import { findReactRoute, reactHostAddress } from './utils';

const initPublicUrl = process.env.PUBLIC_URL;

describe('Routing Utilities', () => {
  afterEach(() => {
    process.env.PUBLIC_URL = initPublicUrl;
  });
  beforeEach(() => {
    process.env.PUBLIC_URL = initPublicUrl;
  });

  describe('findReactRoute', () => {
    it('should match PUBLIC_URL/experiments/1 and PUBLIC_URL/experiments/xyz', () => {
      const subdirectory = '/det';
      process.env.PUBLIC_URL = subdirectory;

      expect(findReactRoute(subdirectory + '/experiments/1')).toBeDefined();
      expect(findReactRoute(subdirectory + '/experiments/1/xyz')).toBeDefined();
    });

    it('should not match without PUBLIC_URL if PUBLIC_URL is set otherwise', () => {
      const subdirectory = '/det';
      process.env.PUBLIC_URL = subdirectory;

      // negative cases
      expect(findReactRoute('det/experiments/1')).toBeUndefined();
      expect(findReactRoute('xyz/experiments/1')).toBeUndefined();
      expect(findReactRoute('/xyz/experiments/1')).toBeUndefined();
      expect(findReactRoute('/experiments/1')).toBeUndefined();
    });

    it('should match full urls: HOST/PUBLIC_URL/experiments/1', () => {
      const subdirectory = '/det';
      process.env.PUBLIC_URL = subdirectory;
      const prefix = reactHostAddress() + subdirectory;

      expect(findReactRoute(prefix + '/experiments/1')).toBeDefined();
      expect(findReactRoute(prefix + '/experiments/1/xyz')).toBeDefined();
    });

    it('should not match full urls from other hosts', () => {
      const subdirectory = '/det';
      process.env.PUBLIC_URL = subdirectory;
      const prefix = 'http://letermined.com' + subdirectory;

      expect(findReactRoute(prefix + '/experiments/1')).toBeUndefined();
      expect(findReactRoute(prefix + '/experiments/1/xyz')).toBeUndefined();
    });

    it('should not match full urls missing PUBLIC_URL', () => {
      const subdirectory = '/det';
      process.env.PUBLIC_URL = subdirectory;

      expect(findReactRoute(reactHostAddress() + '/experiments/1')).toBeUndefined();
      expect(findReactRoute(reactHostAddress() + '/experiments/1/xyz')).toBeUndefined();
    });

  });

  describe('reactHostAddress', () => {
    it('should be independent of PUBLIC_URL', () => {
      const init = process.env.PUBLIC_URL;
      process.env.PUBLIC_URL = 'a';
      const a = reactHostAddress();
      process.env.PUBLIC_URL = 'b';
      const b = reactHostAddress();
      expect(a).toEqual(b);
      process.env.PUBLIC_URL = init;
    });
  });
});
