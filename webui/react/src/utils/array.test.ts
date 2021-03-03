import { sumArrays } from './array';

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
