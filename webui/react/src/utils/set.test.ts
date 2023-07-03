import * as utils from './set';

describe('Set Utilities', () => {
  const one = new Set([1, 2, 3]);
  const two = new Set([1, 2, 3, 4]);
  const three = new Set([2, 3, 4]);

  describe('isSuperset', () => {
    it('should return true if the first argument is a superset of the second', () => {
      expect(utils.isSuperset(two, one)).toBe(true);
    });
    it('should return false if the first argument is not a superset of the second', () => {
      expect(utils.isSuperset(one, two)).not.toBe(true);
    });
  });
  describe('union', () => {
    it('should find the union of two sets', () => {
      expect(utils.union(one, three)).toStrictEqual(two);
    });
  });
  describe('symmetricDifference', () => {
    it('should find the symettric difference of two sets', () => {
      expect(utils.symmetricDifference(one, three)).toStrictEqual(new Set([1, 4]));
    });
  });
  describe('difference', () => {
    it('should find the difference of two sets', () => {
      expect(utils.difference(one, three)).toStrictEqual(new Set([1]));
    });
  });
});
