import * as utils from './error';

describe('Error Handling Utilities', () => {
  describe('isError', () => {
    it('should report Error objects as errors', () => {
      const error = new Error('This is a crazy error!');
      expect(utils.isError(error)).toStrictEqual(true);

      const detError = new utils.DetError('This is a DET error.');
      expect(utils.isError(detError)).toStrictEqual(true);
    });

    it('should not report non-Error items else as errors', () => {
      expect(utils.isError(true)).toStrictEqual(false);
      expect(utils.isError(123)).toStrictEqual(false);
      expect(utils.isError('hello')).toStrictEqual(false);
      expect(utils.isError(new Date())).toStrictEqual(false);
      expect(utils.isError(new Set([ 1, 2, 3 ]))).toStrictEqual(false);
    });
  });

  describe('isDetError', () => {
    it('should detect DetError objects', () => {
      const error = new utils.DetError('This is a crazy error!');
      expect(utils.isDetError(error)).toStrictEqual(true);
    });

    it('should not report non-DetError items', () => {
      const error = new Error('This is a normal error.');
      expect(utils.isDetError(error)).toStrictEqual(false);
      expect(utils.isDetError(123)).toStrictEqual(false);
      expect(utils.isDetError('hello')).toStrictEqual(false);
      expect(utils.isDetError(new Date())).toStrictEqual(false);
      expect(utils.isDetError(new Set([ 1, 2, 3 ]))).toStrictEqual(false);
    });
  });
});
