import { assertIsDefined } from './assertion';

const setup = <T>(value: T): () => void => {
  return () => {
    assertIsDefined(value);
  };
};

describe('Assertion Utilities', () => {
  describe('assertIsDefined', () => {
    describe('string', () => {
      it('should be defined with empty string value', () => {
        const str: string | undefined = '';
        const func = setup(str);
        expect(typeof str === 'string').toBe(true);
        expect(func).not.toThrow(Error);
      });

      it('should be defined with string value', () => {
        const str: string | undefined = 'abc';
        const func = setup(str);
        expect(typeof str === 'string').toBe(true);
        expect(func).not.toThrow(Error);
      });

      it('should be undefined with undefined value', () => {
        const str: string | undefined = undefined;
        const func = setup(str);
        expect(typeof str === 'string').toBe(false);
        expect(typeof str === 'undefined').toBe(true);
        expect(func).toThrow(Error);
      });

      it('should be undefined with null value', () => {
        const str: string | null = null;
        const func = setup(str);
        expect(typeof str === 'string').toBe(false);
        expect(str).toBeNull();
        expect(func).toThrow(Error);
      });
    });
  });

  describe('number', () => {
    it('should be defined with 0 value', () => {
      const num: number | undefined = 0;
      const func = setup(num);
      expect(typeof num === 'number').toBe(true);
      expect(func).not.toThrow(Error);
    });

    it('should be defined with positive value', () => {
      const num: number | undefined = 123345;
      const func = setup(num);
      expect(typeof num === 'number').toBe(true);
      expect(func).not.toThrow(Error);
    });

    it('should be defined with negative value', () => {
      const num: number | undefined = -123345;
      const func = setup(num);
      expect(typeof num === 'number').toBe(true);
      expect(func).not.toThrow(Error);
    });

    it('should be undefined with undefined value', () => {
      const num: number | undefined = undefined;
      const func = setup(num);
      expect(typeof num === 'number').toBe(false);
      expect(typeof num === 'undefined').toBe(true);
      expect(func).toThrow(Error);
    });

    it('should be undefined with null value', () => {
      const num: number | null = null;
      const func = setup(num);
      expect(typeof num === 'number').toBe(false);
      expect(num).toBeNull();
      expect(func).toThrow(Error);
    });
  });
});
