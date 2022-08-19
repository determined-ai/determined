import {
  DEFAULT_ERROR_MESSAGE,
  DetError,
  DetErrorOptions,
  ErrorLevel,
  ErrorType,
  isDetError,
  isError,
} from 'shared/utils/error';

const DEFAULT_DET_ERROR_OPTIONS: DetErrorOptions = {
  id: 'badbed',
  isUserTriggered: true,
  level: ErrorLevel.Fatal,
  payload: { abc: 1, def: true, ghi: 'what' },
  publicMessage: 'Public Message',
  publicSubject: 'Public Subject',
  silent: true,
  type: ErrorType.Ui,
};

describe('Error Handling Utilities', () => {
  describe('isError', () => {
    it('should report Error objects as errors', () => {
      const error = new Error('This is a crazy error!');
      expect(isError(error)).toBe(true);

      const detError = new DetError('This is a DET error.');
      expect(isError(detError)).toBe(true);
    });

    it('should not report non-Error items else as errors', () => {
      expect(isError(true)).toBe(false);
      expect(isError(123)).toBe(false);
      expect(isError('hello')).toBe(false);
      expect(isError(new Date())).toBe(false);
      expect(isError(new Set([ 1, 2, 3 ]))).toBe(false);
    });
  });

  describe('isDetError', () => {
    it('should detect DetError objects', () => {
      const detError = new DetError('This is a crazy error!');
      expect(isDetError(detError)).toBe(true);
    });

    it('should not report non-DetError items', () => {
      const error = new Error('This is a normal error.');
      expect(isDetError(error)).toBe(false);
      expect(isDetError(true)).toBe(false);
      expect(isDetError(123)).toBe(false);
      expect(isDetError('hello')).toBe(false);
      expect(isDetError(new Date())).toBe(false);
      expect(isDetError(new Set([ 1, 2, 3 ]))).toBe(false);
    });
  });

  describe('DetError', () => {
    it('should construct DetError from string error message', () => {
      const detError = new DetError(DEFAULT_ERROR_MESSAGE);

      expect(isDetError(detError)).toBe(true);
      expect(detError.message).toBe(DEFAULT_ERROR_MESSAGE);
    });

    it('should construct DetError from Error', () => {
      const error = new Error(DEFAULT_ERROR_MESSAGE);
      const detError = new DetError(error);

      expect(isError(detError)).toBe(true);
      expect(isDetError(detError)).toBe(true);
      expect(detError.message).toBe(DEFAULT_ERROR_MESSAGE);
    });

    it('should construct DetError from DetError', () => {
      const oldDetError = new DetError(DEFAULT_ERROR_MESSAGE);
      const newDetError = new DetError(oldDetError);

      expect(isError(newDetError)).toBe(true);
      expect(isDetError(newDetError)).toBe(true);
      expect(newDetError.message).toBe(DEFAULT_ERROR_MESSAGE);
    });

    it('should construct DetError with error options', () => {
      const oldDetError = new DetError(DEFAULT_ERROR_MESSAGE);
      const newDetError = new DetError(oldDetError, DEFAULT_DET_ERROR_OPTIONS);

      expect(isDetError(newDetError)).toBe(true);

      // Expect each error option value to be preserved in the new DetError.
      for (const [ key, value ] of Object.entries(DEFAULT_DET_ERROR_OPTIONS)) {
        expect(value).toStrictEqual(newDetError[key as keyof DetErrorOptions]);
      }
    });
  });
});
