import fc from 'fast-check';
import { describe, test } from 'vitest';

import { Failed, Loadable, Loaded, NotLoaded } from './loadable';

describe('loadable', () => {
  // map
  test.concurrent('map should retain any Loaded values', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(Loaded(data).map((d) => d)).toEqual(Loaded(data));
      }),
    );
  });
  test.concurrent('map should modify any Loaded values', () => {
    expect(Loaded(1).map((d) => d + 1)).toEqual(Loaded(2));
  });
  test.concurrent('map should leave NotLoaded unchanged', () => {
    fc.assert(
      fc.property(fc.func(fc.anything()), (data) => {
        expect(NotLoaded.map(data)).toEqual(NotLoaded);
      }),
    );
  });
  test.concurrent('map should leave Failed unchanged', () => {
    fc.assert(
      fc.property(fc.func(fc.anything()), (data) => {
        expect(Failed(new Error('test')).map(data)).toEqual(Failed(new Error('test')));
      }),
    );
  });

  // flatMap
  test.concurrent('flatMap should retain any Loaded values', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(Loaded(data).flatMap((d) => Loaded(d))).toEqual(Loaded(data));
      }),
    );
  });
  test.concurrent('flatMap should convert Loaded to NotLoaded', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(Loaded(data).flatMap(() => NotLoaded)).toEqual(NotLoaded);
      }),
    );
  });
  test.concurrent('flatMap should convert Loaded to Failed', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(Loaded(data).flatMap(() => Failed(new Error('test')))).toEqual(
          Failed(new Error('test')),
        );
      }),
    );
  });
  test.concurrent('flatMap should modify any Loaded values', () => {
    expect(Loaded(1).flatMap((d) => Loaded(d + 1))).toEqual(Loaded(2));
  });
  test.concurrent('flatMap should leave NotLoaded unchanged', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(NotLoaded.flatMap(() => Loaded(data))).toEqual(NotLoaded);
      }),
    );
  });
  test.concurrent('flatMap should leave Failed unchanged', () => {
    fc.assert(
      fc.property(fc.func(fc.anything()), (data) => {
        expect(Failed(new Error('test')).flatMap(() => Loaded(data))).toEqual(
          Failed(new Error('test')),
        );
      }),
    );
  });

  // foreach
  test.concurrent('forEach should observe Loaded values', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        let i = null;
        Loaded(data).forEach((d) => {
          i = d;
        });
        expect(i).toEqual(data);
      }),
    );
  });
  test.concurrent('forEach should not observe NotLoaded values', () => {
    let i = null;
    NotLoaded.forEach(() => {
      i = 5;
    });
    expect(i).toEqual(null);
  });
  test.concurrent('forEach should not observe Failed values', () => {
    let i = null;
    Failed(new Error('test')).forEach(() => {
      i = 5;
    });
    expect(i).toEqual(null);
  });

  // getOrElse
  test.concurrent('getOrElse should return Loaded values', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(Loaded(data).getOrElse(null)).toBe(data);
      }),
    );
  });
  test.concurrent('getOrElse should return default for NotLoaded', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect((NotLoaded as Loadable<unknown>).getOrElse(data)).toEqual(data);
      }),
    );
  });
  test.concurrent('getOrElse should return default for Failed', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(Failed(new Error('test')).getOrElse(data)).toEqual(data);
      }),
    );
  });

  // match
  test.concurrent('match should call Loaded case for Loaded', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(
          Loaded(data).match({
            _: () => 1,
            Loaded: (data) => data,
          }),
        ).toBe(data);
      }),
    );
  });
  test.concurrent('match should call NotLoaded case for NotLoaded', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(
          NotLoaded.match({
            _: () => 1,
            NotLoaded: () => data,
          }),
        ).toBe(data);
      }),
    );
  });
  test.concurrent('match should call Failed case for Failed', () => {
    fc.assert(
      fc.property(fc.anything(), (data) => {
        expect(
          Failed(new Error('test')).match({
            Failed: () => data,
            Loaded: () => 1,
            NotLoaded: () => 2,
          }),
        ).toBe(data);
      }),
    );
  });

  // all
  test.concurrent('all should handle all Loadeds', () => {
    fc.assert(
      fc.property(fc.array(fc.anything()), (data) => {
        const loadeds = data.map((d) => Loaded(d));
        Loadable.all(loadeds).forEach((d) => {
          expect(d).toEqual(data);
        });
      }),
    );
  });
  test.concurrent('all should convert any NotLoaded into NotLoaded', () => {
    fc.assert(
      fc.property(fc.array(fc.anything()), (data) => {
        const loadeds = data.map((d) => Loaded(d));
        loadeds.push(NotLoaded);
        expect(Loadable.all(loadeds)).toBe(NotLoaded);

        const loadeds2 = [NotLoaded, ...data.map((d) => Loaded(d))];
        expect(Loadable.all(loadeds2)).toBe(NotLoaded);
      }),
    );
  });
  test.concurrent('all should convert any Failed into Failed', () => {
    fc.assert(
      fc.property(fc.array(fc.anything()), (data) => {
        const loadeds = data.map((d) => Loaded(d));
        loadeds.push(Failed(new Error('test')));
        expect(Loadable.all(loadeds)).toStrictEqual(Failed(new Error('test')));
      }),
    );
  });
  test.concurrent('all should prioritize Failed over NotLoaded', () => {
    fc.assert(
      fc.property(fc.array(fc.anything()), (data) => {
        const loadeds = data.map((d) => Loaded(d));
        loadeds.push(NotLoaded);
        loadeds.push(Failed(new Error('test')));
        expect(Loadable.all(loadeds)).toStrictEqual(Failed(new Error('test')));

        const loadeds2 = data.map((d) => Loaded(d));
        loadeds2.push(Failed(new Error('test')));
        loadeds2.push(NotLoaded);
        expect(Loadable.all(loadeds)).toStrictEqual(Failed(new Error('test')));
      }),
    );
  });
});
