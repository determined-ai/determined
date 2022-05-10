import { RecordKey } from './types';

describe('Array.prototype utility', () => {
  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  const arrays: Record<RecordKey, Record<RecordKey, any[]>> = {
    boolean: {
      original: [ true, false, false, true ],
      sorted: [ false, false, true, true ],
    },
    empty: {
      original: [],
      sorted: [],
    },
    mixed: {
      original: [ -10, 0, null, undefined, 10, 200 ],
      sorted: [ undefined, null, -10, 0, 10, 200 ],
    },
    numbers: {
      original: [ 5280, 3.14, 2e10, -123, 5e-6, 0 ],
      sorted: [ -123, 0, 5e-6, 3.14, 5280, 2e10 ],
    },
    strings: {
      original: [ 'abcdef', 'ghij', 'xyz', 'XYZ' ],
      sorted: [ 'XYZ', 'abcdef', 'ghij', 'xyz' ],
    },
  };

  const dumbCompare = <T>(a: T, b: T): number => {
    if (a == null && b != null) return -1;
    if (a != null && b == null) return 1;
    if (a < b) return -1;
    if (a > b) return 1;
    return 0;
  };

  Object.keys(arrays).forEach(key => {
    const arr = arrays[key].original;
    const sortedArr = arrays[key].sorted;

    describe('first', () => {
      it(`should grab first element of ${key} array`, () => {
        expect(arr.first()).toBe(arr[0]);
      });
    });

    describe('last', () => {
      it(`should grab last element of ${key} array`, () => {
        expect(arr.last()).toBe(arr[arr.length - 1]);
      });
    });

    describe('random', () => {
      if (key === 'empty') {
        it('should return undefined when grabbing random from an empty array', () => {
          expect(arr.random()).toBeUndefined();
        });
      } else {
        it(`should grab random element of ${key} array`, () => {
          const random = arr.random();
          expect(arr).toContainEqual(random);
        });
      }
    });

    describe('sortAll', () => {
      const arrCopy = [ ...arr ];
      arrCopy.sortAll(dumbCompare);
      it('should sort different types of array', () => {
        expect(arrCopy).toEqual(sortedArr);
      });
    });
  });
});
