import * as utils from 'utils/string';

describe('String Utilities', () => {
  describe('snakeCaseToTitleCase', () => {
    it('should convert snake case to title case', () => {
      expect(utils.snakeCaseToTitleCase('')).toBe('');
      expect(utils.snakeCaseToTitleCase('hello')).toBe('Hello');
      expect(utils.snakeCaseToTitleCase('hello_world')).toBe('Hello World');
      expect(utils.snakeCaseToTitleCase('hello_new_world')).toBe('Hello New World');
      expect(utils.snakeCaseToTitleCase('Hello_New_World')).toBe('Hello New World');
      expect(utils.snakeCaseToTitleCase('hello_New_world')).toBe('Hello New World');
      expect(utils.snakeCaseToTitleCase('hello_NEW_world')).toBe('Hello New World');
      expect(utils.snakeCaseToTitleCase('HELLO_NEW_WORLD')).toBe('Hello New World');
      expect(utils.snakeCaseToTitleCase('hEllO_New_WoRLD')).toBe('Hello New World');
      expect(utils.snakeCaseToTitleCase('hello__New_world')).toBe('Hello  New World');
    });
  });

  describe('camelCaseToKebab', () => {
    it('should convert camel case to a kebab', () => {
      expect(utils.camelCaseToKebab('hello')).toBe('hello');
      expect(utils.camelCaseToKebab('camelCase')).toBe('camel-case');
      expect(utils.camelCaseToKebab(' carJumpStart ')).toBe('car-jump-start');
    });
  });

  describe('camelCaseToSentence', () => {
    it('should convert camel case to a sentence', () => {
      expect(utils.camelCaseToSentence('hello')).toBe('Hello');
      expect(utils.camelCaseToSentence('camelCase')).toBe('Camel Case');
      expect(utils.camelCaseToSentence(' carJumpStart ')).toBe('Car Jump Start');
    });
  });

  describe('camelCaseToSnake', () => {
    it('should convert camel case to a snake', () => {
      expect(utils.camelCaseToSnake('hello')).toBe('hello');
      expect(utils.camelCaseToSnake('camelCase')).toBe('camel_case');
      expect(utils.camelCaseToSnake(' carJumpStart ')).toBe('car_jump_start');
    });
  });

  describe('kebabToCamelCase', () => {
    it('should convert kebab to camel case', () => {
      expect(utils.kebabToCamelCase('Hello')).toBe('hello');
      expect(utils.kebabToCamelCase('camel-case')).toBe('camelCase');
      expect(utils.kebabToCamelCase(' car-jump-start ')).toBe('carJumpStart');
    });
  });

  describe('sentenceToCamelCase', () => {
    it('should convert camel case to a sentence', () => {
      expect(utils.sentenceToCamelCase('Hello')).toBe('hello');
      expect(utils.sentenceToCamelCase('Camel Case')).toBe('camelCase');
      expect(utils.sentenceToCamelCase(' Car Jump Start ')).toBe('carJumpStart');
    });
  });

  describe('capitalize', () => {
    it('should capitalize all words in a string', () => {
      const tests = [
        { input: '123 abc Def ghi', output: '123 Abc Def Ghi' },
        { input: '^@hello', output: '^@hello' },
      ];

      tests.forEach((test) => {
        expect(utils.capitalize(test.input)).toBe(test.output);
      });
    });
  });

  describe('capitalizeWord', () => {
    it('should make first character uppercase and the rest lower case', () => {
      const tests = [
        { input: 'hello world', output: 'Hello world' },
        { input: 'hello World', output: 'Hello world' },
        { input: '^@hello', output: '^@hello' },
      ];

      tests.forEach((test) => {
        expect(utils.capitalizeWord(test.input)).toBe(test.output);
      });
    });
  });

  describe('floatToPercent', () => {
    it('should convert float to percentage string', () => {
      expect(utils.floatToPercent(0)).toBe('0.00%');
      expect(utils.floatToPercent(0.5)).toBe('50.00%');
      expect(utils.floatToPercent(1)).toBe('100.00%');
    });

    it('should handle NaN, Infinity and -Infinity', () => {
      expect(utils.floatToPercent(NaN)).toBe('NaN');
      expect(utils.floatToPercent(Infinity)).toBe('Infinity');
      expect(utils.floatToPercent(-Infinity)).toBe('-Infinity');
    });

    it('should convert float to percentage string with various precisions', () => {
      expect(utils.floatToPercent(Math.PI, 0)).toBe('314%');
      expect(utils.floatToPercent(Math.PI, 1)).toBe('314.2%');
      expect(utils.floatToPercent(Math.PI, 2)).toBe('314.16%');
      expect(utils.floatToPercent(Math.PI, 3)).toBe('314.159%');
    });
  });

  describe('generateAlphaNumeric', () => {
    it('should generate a default sized alpha numeric string', () => {
      const alphaNumeric = utils.generateAlphaNumeric();
      const regex = new RegExp(`^[a-zA-Z0-9]{${utils.DEFAULT_ALPHA_NUMERIC_LENGTH}}$`, 'i');
      expect(regex.test(alphaNumeric)).toBe(true);
    });

    it('should generate various sized alpha numeric strings', () => {
      for (let i = 1; i < 10; i++) {
        const alphaNumeric = utils.generateAlphaNumeric(i);
        const regex = new RegExp(`^[a-zA-Z0-9]{${i}}$`, 'i');
        expect(regex.test(alphaNumeric)).toBe(true);
      }
    });

    it('should generate alhpa numeric strings from provided character set', () => {
      const CHARSET = 'AMZ';
      for (let i = 1; i < 10; i++) {
        const alphaNumeric = utils.generateAlphaNumeric(i, CHARSET);
        const regex = new RegExp(`^[AMZ]{${i}}$`, 'i');
        expect(regex.test(alphaNumeric)).toBe(true);
      }
    });
  });

  describe('generateUUID', () => {
    it('should generate UUIDs with the correct format', () => {
      const regex = /[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}/i;
      for (let i = 0; i < 10; i++) {
        const uuid = utils.generateUUID();
        expect(regex.test(uuid)).toBe(true);
      }
    });
  });

  describe('generateLetters', () => {
    it('should generate default sized letters only string', () => {
      const letters = utils.generateLetters();
      const regex = new RegExp(`^[a-zA-Z]{${utils.DEFAULT_ALPHA_NUMERIC_LENGTH}}$`, 'i');
      expect(regex.test(letters)).toBe(true);
    });

    it('should generate various sized letters only string', () => {
      for (let i = 1; i < 10; i++) {
        const letters = utils.generateLetters(i);
        const regex = new RegExp(`^[a-zA-Z]{${i}}$`, 'i');
        expect(regex.test(letters)).toBe(true);
      }
    });
  });

  describe('versionToString', () => {
    const cases = [
      { parsed: [1, 0, 0], str: '1.0.0' },
      { parsed: [0, 1, 0], str: '0.1.0' },
      { parsed: [23, 0, 1], str: '23.0.1' },
    ].map(({ str, parsed }) => ({
      parsed: {
        major: parsed[0],
        minor: parsed[1],
        patch: parsed[2],
      },
      str,
    }));

    it('parse string into semantic versions', () => {
      cases.forEach(({ str, parsed }) => {
        expect(utils.stringToVersion(str)).toStrictEqual(parsed);
      });
    });

    it('convert semantic version to string', () => {
      cases.forEach(({ str, parsed }) => {
        expect(utils.versionToString(parsed)).toStrictEqual(str);
      });
    });
  });

  describe('humanReadableBytes', () => {
    it('should convert bytes into readable bytes', () => {
      expect(utils.humanReadableBytes(0)).toBe('0 B');
      expect(utils.humanReadableBytes(1)).toBe('1 B');
      expect(utils.humanReadableBytes(1024)).toBe('1.02 kB');
      expect(utils.humanReadableBytes(2048)).toBe('2.05 kB');
      expect(utils.humanReadableBytes(1234567)).toBe('1.23 MB');
      expect(utils.humanReadableBytes(1234567890)).toBe('1.23 GB');
      expect(utils.humanReadableBytes(1234567890123)).toBe('1.23 TB');
      expect(utils.humanReadableBytes(1234567890123456)).toBe('1.23 PB');
    });
  });

  describe('listToStr', () => {
    it('should glue defined list items together', () => {
      expect(utils.listToStr(['a', 'b', 'c'])).toBe('a b c');
      expect(utils.listToStr(['a', undefined, 'b', undefined, 'c'])).toBe('a b c');
    });

    it('should glue defined list items together with custom glue', () => {
      expect(utils.listToStr(['a', undefined, 'b', undefined, 'c'], ', ')).toBe('a, b, c');
    });
  });

  describe('toHtmlId', () => {
    it('should replace spaces with -', () => {
      expect(utils.toHtmlId('Hello World')).toBe('hello-world');
    });

    it('should remove everything but alphanumeric and -', () => {
      expect(utils.toHtmlId('He$)%*#$%llo World)#$(%)')).toBe('hello-world');
    });

    it('should generate lowercase ids', () => {
      expect(utils.toHtmlId('HellO')).toBe('hello');
    });
  });

  describe('truncate', () => {
    it('should truncate strings at various sizes', () => {
      const VERY_LONG_STRING = 'very-very-very-very-very-long-string';
      expect(utils.truncate(VERY_LONG_STRING)).toBe('very-very-very-ve...');
      expect(utils.truncate(VERY_LONG_STRING, 1)).toBe('v...');
      expect(utils.truncate(VERY_LONG_STRING, 5)).toBe('ve...');
      expect(utils.truncate(VERY_LONG_STRING, 10)).toBe('very-ve...');
    });

    it('should skip truncating if string is within max length', () => {
      expect(utils.truncate('abc', 3)).toBe('abc');
    });

    it('should avoid changing short strings', () => {
      const s = 'abc';
      expect(utils.truncate(s, s.length + 1)).toBe(s);
    });

    it('should add a suffix when truncating', () => {
      const testStr = 'adoptacat';
      const suffix = '...';
      const size = 4;
      expect(utils.truncate(testStr, size, suffix)).toBe(
        testStr.substring(0, size - suffix.length) + suffix,
      );
    });

    it('should support skipping the suffix', () => {
      const testStr = 'adoptacat';
      const size = 4;
      expect(utils.truncate(testStr, size, '')).toBe(testStr.substring(0, size));
    });
  });

  describe('validateLength', () => {
    const sShort = 'Hello';
    const s80 = 'jfdjsakljfdsalkjflksadjflksajflkasjflksajflkdsjfkljdsafjsklfdsjkaljfdslasdfsdfdd';
    const space5 = '     ';

    it('should validate length with default params', () => {
      expect(utils.validateLength('')).toBeFalsy();
      expect(utils.validateLength('a')).toBeTruthy();
      expect(utils.validateLength(sShort)).toBeTruthy();
      expect(utils.validateLength(s80)).toBeTruthy();
      expect(utils.validateLength(s80 + 'a')).toBeFalsy();
      expect(utils.validateLength(s80 + '        ')).toBeTruthy();
    });

    it('should validate length with custom length', () => {
      expect(utils.validateLength('', 0)).toBeTruthy();
      expect(utils.validateLength(s80)).toBeTruthy();
      expect(utils.validateLength('1234567890', 1, 10)).toBeTruthy();
      expect(utils.validateLength('12345678901', 1, 10)).toBeFalsy();
      expect(utils.validateLength('12345678901', 10, 1)).toBeFalsy();
      expect(utils.validateLength('1', 1, 1)).toBeTruthy();
      expect(utils.validateLength('12', 1, 1)).toBeFalsy();
    });

    it('should validate length with custom trim', () => {
      expect(utils.validateLength(s80 + ' ', 1, 80)).toBeTruthy();
      expect(utils.validateLength(s80 + ' ', 1, 80, false)).toBeFalsy();
      expect(utils.validateLength(space5 + ' ', 1, 5, false)).toBeFalsy();
      expect(utils.validateLength(space5 + '  ', 1, 5, true)).toBeFalsy();
      expect(utils.validateLength('abcde' + space5, 1, 5)).toBeTruthy();
      expect(utils.validateLength('abcde' + space5, 1, 5, true)).toBeTruthy();
      expect(utils.validateLength(space5, 1, 5, false)).toBeTruthy();
    });
  });
  describe('pluralizer', () => {
    it("shouldn't pluralize if count === 1", () => {
      expect(utils.pluralizer(1, 'apple')).toBe('apple');
      expect(utils.pluralizer(1, 'octopus')).toBe('octopus');
    });

    it('should pluralize if count !== 1', () => {
      expect(utils.pluralizer(2, 'apple')).toBe('apples');
      expect(utils.pluralizer(-1, 'apple')).toBe('apples');
      expect(utils.pluralizer(0.5, 'apple')).toBe('apples');
    });

    it('should handle non-standard plurals', () => {
      expect(utils.pluralizer(2, 'octopus', 'octopi')).toBe('octopi');
    });
  });
});
