import { findInsertionIndex, sumArrays } from './array';

describe('findInsertionIndex', () => {
  it('empty', () => {
    expect(findInsertionIndex([], 1))
      .toStrictEqual(0);
  });

  it('unique', () => {
    expect(findInsertionIndex([ 1, 1, 2, 3, 5, 8 ], 13))
      .toStrictEqual(6);
  });

  it('simple', () => {
    expect(findInsertionIndex([ 1, 1, 2, 3, 5, 8 ], 3))
      .toStrictEqual(3);
  });

  it('negative values', () => {
    expect(findInsertionIndex([ 1, 1, 2, 3, 5, 8 ], 4))
      .toStrictEqual(4);
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
