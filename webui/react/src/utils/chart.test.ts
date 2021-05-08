import { defaultNumericRange,
  distance, getNumericRange, normalizeRange, updateRange } from './chart';

describe('defaultNumericRange', () => {
  it('Default', () => {
    expect(defaultNumericRange())
      .toStrictEqual([ Number.NEGATIVE_INFINITY, Number.POSITIVE_INFINITY ]);
  });
  it('Reverse', () => {
    expect(defaultNumericRange(true))
      .toStrictEqual([ Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY ]);
  });
});

describe('getNumericRange', () => {
  it('No values', () => {
    expect(getNumericRange([])).toBeUndefined();
  });
  it('Get range', () => {
    expect(getNumericRange([ 1,2,4 ])).toStrictEqual([ 1,4 ]);
  });
  it('Force range', () => {
    expect(getNumericRange([ 3.5 ], true)).toStrictEqual([ 3,4 ]);
  });
  it('Force range 2', () => {
    expect(getNumericRange([ 1,2,4 ], true)).toStrictEqual([ 1,4 ]);
  });
});

describe('updateRange', () => {
  it('No range', () => {
    expect(updateRange(undefined, 5)).toStrictEqual([ 5,5 ]);
  });
  it('Above range', () => {
    expect(updateRange([ 1,4 ], 5)).toStrictEqual([ 1,5 ]);
  });
  it('Below range', () => {
    expect(updateRange([ 1,4 ], 0)).toStrictEqual([ 0,4 ]);
  });
});

describe('normalizeRange', () => {
  it('No values, no range', () => {
    expect(normalizeRange([], [ 1,1 ])).toStrictEqual([]);
  });
  it('No range', () => {
    expect(normalizeRange([ 1,2,5 ], [ 1,1 ])).toStrictEqual([ 1,2,5 ]);
  });
  it('No values', () => {
    expect(normalizeRange([], [ 1,5 ])).toStrictEqual([]);
  });
  it('Values and range', () => {
    expect(normalizeRange([ 1,2,5 ], [ 1,5 ])).toStrictEqual([ 0,1/4,1 ]);
  });
});

describe('distance', () => {
  it('No difference', () => {
    expect(distance(0,0,0,0)).toStrictEqual(0);
  });
  it('Distance', () => {
    expect(distance(0,0,1,1)**2).toBeCloseTo(2);
  });
});
