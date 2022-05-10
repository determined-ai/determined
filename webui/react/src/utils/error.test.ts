import { DetError, isDetError, isError } from '../shared/utils/error';

describe('Error Handling Utilities', () => {
  describe('isError', () => {
    it('should report Error objects as errors', () => {
      const error = new Error('This is a crazy error!');
      expect(isError(error)).toStrictEqual(true);

      const detError = new DetError('This is a DET error.');
      expect(isError(detError)).toStrictEqual(true);
    });

    it('should not report non-Error items else as errors', () => {
      expect(isError(true)).toStrictEqual(false);
      expect(isError(123)).toStrictEqual(false);
      expect(isError('hello')).toStrictEqual(false);
      expect(isError(new Date())).toStrictEqual(false);
      expect(isError(new Set([ 1, 2, 3 ]))).toStrictEqual(false);
    });
  });

  describe('isDetError', () => {
    it('should detect DetError objects', () => {
      const error = new DetError('This is a crazy error!');
      expect(isDetError(error)).toStrictEqual(true);
    });

    it('should not report non-DetError items', () => {
      const error = new Error('This is a normal error.');
      expect(isDetError(error)).toStrictEqual(false);
      expect(isDetError(123)).toStrictEqual(false);
      expect(isDetError('hello')).toStrictEqual(false);
      expect(isDetError(new Date())).toStrictEqual(false);
      expect(isDetError(new Set([ 1, 2, 3 ]))).toStrictEqual(false);
    });
  });
});
