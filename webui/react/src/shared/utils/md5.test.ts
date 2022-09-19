import md5 from './md5';

const LOREM_IPSUM = `
  Lorem ipsum dolor sit amet, consectetur adipiscing elit,
  sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
  Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris
  nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in
  reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla
  pariatur. Excepteur sint occaecat cupidatat non proident, sunt in
  culpa qui officia deserunt mollit anim id est laborum.
`
  .replace(/(?:\r\n|\r|\n)/g, '')
  .replace(/\s+/g, ' ')
  .trim();

describe('md5 Utility', () => {
  const tests = [
    { input: 'hello', output: '5d41402abc4b2a76b9719d911017c592' },
    { input: 'world', output: '7d793037a0760186574b0282f2f435e7' },
    { input: LOREM_IPSUM, output: 'db89bb5ceab87f9c0fcc2ab36c189c2c' },
  ];
  tests.forEach((test) => {
    it(`should hash "${test.input}" to "${test.output}"`, () => {
      expect(md5(test.input)).toBe(test.output);
    });
  });
});
