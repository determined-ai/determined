import * as utils from './apiConfig';

describe('apiConfig', () => {
  describe('getUserIds', () => {
    it('should convert user id strings into user id numbers', () => {
      expect(utils.getUserIds(['123'])).toStrictEqual([123]);
      expect(utils.getUserIds(['456', '789'])).toStrictEqual([456, 789]);
    });

    it('should filter out non-numeric string user ids', () => {
      expect(utils.getUserIds(['abc', '123'])).toStrictEqual([123]);
    });

    it('should return `undefined` when there are no valid user ids', () => {
      expect(utils.getUserIds(['abc', 'def'])).toBeUndefined();
    });
  });
});
