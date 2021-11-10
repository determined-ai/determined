import * as string from './string';

describe('Routing Utilities', () => {
  describe('capitalizeWord', () => {
    it('should make first character uppercase and the rest lower case', () => {
      const tests = [
        { input: 'hello world', output: 'Hello world' },
        { input: 'hello World', output: 'Hello world' },
        { input: '^@hello', output: '^@hello' },
      ];

      tests.forEach(test => {
        expect(string.capitalizeWord(test.input)).toBe(test.output);
      });
    });
  });

  describe('capitalize', () => {
    it('should capitalize all words in a string', () => {
      const tests = [
        { input: '123 abc Def ghi', output: '123 Abc Def Ghi' },
        { input: '^@hello', output: '^@hello' },
      ];

      tests.forEach(test => {
        expect(string.capitalize(test.input)).toBe(test.output);
      });
    });
  });

  describe('toHtmlId', () => {
    it('should replace spaces with -', () => {
      expect(string.toHtmlId('Hello World')).toBe('hello-world');
    });

    it('should remove everything but alphanumeric and -', () => {
      expect(string.toHtmlId('He$)%*#$%llo World)#$(%)')).toBe('hello-world');
    });

    it('should generate lowercase ids', () => {
      expect(string.toHtmlId('HellO')).toBe('hello');
    });
  });
});
