import { findInsertionIndex, sumArrays } from './array';

const FIBO = [ 0, 1, 1, 2, 3, 5, 8 ];

describe('findInsertionIndex', () => {
  it('empty', () => {
    expect(findInsertionIndex([], 1)) .toStrictEqual(0);
  });

  it('beyond max', () => {
    expect(findInsertionIndex(FIBO, 13)).toStrictEqual(7);
  });

  it('existing value', () => {
    expect(findInsertionIndex(FIBO, 3)).toStrictEqual(4);
  });

  it('unexisting value', () => {
    expect(findInsertionIndex(FIBO, 4)).toStrictEqual(5);
  });
});

describe('sumArrays', () => {
  it('empty', () => {
    expect(sumArrays([]))
      .toStrictEqual([]);
  });

  it('unique', () => {
    expect(sumArrays([ 1, 2, 3 ]))
      .toStrictEqual([ 1, 2, 3 ]);
  });

  it('simple', () => {
    expect(sumArrays([ 1, 2, 3 ], [ 3, 2, 1 ]))
      .toStrictEqual([ 4, 4, 4 ]);
  });

  it('negative values', () => {
    expect(sumArrays([ -1, -2, -3 ], [ 3, 2, 1 ]))
      .toStrictEqual([ 2, 0, -2 ]);
  });
});
