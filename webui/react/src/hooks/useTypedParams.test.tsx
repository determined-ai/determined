import { act, render } from '@testing-library/react';
import fc from 'fast-check';
import { NotLoaded } from 'hew/utils/loadable';
import * as t from 'io-ts';
import { isEqual, pick } from 'lodash';
import { observable } from 'micro-observables';
import { useLayoutEffect, useMemo } from 'react';
import { BrowserRouter } from 'react-router-dom';

import { Json, JsonObject } from 'types';

import { UserSettings } from './useSettingsProvider';
import { useTypedParams } from './useTypedParams';

const navSpy = vi.hoisted(() => vi.fn());
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useSearchParams: () => [new URLSearchParams(), navSpy],
}));

const codec = t.partial({
  array: t.array(t.number),
  boolean: t.boolean,
  deepObject: JsonObject,
  dictionary: t.record(t.string, t.string),
  nested: t.readonly(
    t.type({
      boolean: t.boolean,
      literal: t.literal('literal'),
    }),
  ),
  number: t.number,
  string: t.string,
  union: t.union([t.literal(true), t.literal(0)]),
});

const arbObj = {
  array: fc.array(fc.float(), { minLength: 1 }),
  boolean: fc.boolean(),
  // weird type mismatch between json types, idk
  deepObject: fc.dictionary(fc.string(), fc.jsonValue() as fc.Arbitrary<Json>),
  dictionary: fc.dictionary(fc.string(), fc.string()),
  nested: fc.record({
    boolean: fc.boolean(),
    literal: fc.constant('literal' as const),
  }),
  number: fc.float(),
  string: fc.string(),
  union: fc.constantFrom(true as const, 0 as const),
};
const arb = fc.record(arbObj);
const arbPartial = fc.shuffledSubarray(Object.entries(arbObj)).chain((entries) => {
  const partial = entries.reduce(
    (memo, [k, v]) => {
      memo[k] = v;
      return memo;
    },
    {} as Record<string, fc.Arbitrary<unknown>>,
  );
  return fc.record(partial);
});

const sameArbPartial = fc
  .tuple(arb, fc.shuffledSubarray(Object.keys(arbObj)))
  .map(([params, keys]) => {
    const partial = pick(params, keys);
    return [params, partial];
  });

const diffArbPartial = fc.tuple(arb, arbPartial).filter(([params, partial]) => {
  return !isEqual(partial, pick(params, Object.keys(partial)));
});

const setupTest = (params: t.TypeOf<typeof codec>) => {
  const outerRef: { current: null | ReturnType<typeof useTypedParams> } = { current: null };
  const Wrapper = ({ params }: { params: t.TypeOf<typeof codec> }) => {
    // set up queryparams for usersettingsprovider
    const querySettings = useMemo(() => {
      const pairs = Object.entries(params).reduce(
        (memo, [k, v]) => {
          if (Array.isArray(v)) {
            v.forEach((sv) => {
              if (sv !== null) {
                const stringValue = sv instanceof Object ? JSON.stringify(sv) : sv.toString();
                memo.push([k, stringValue]);
              }
            });
          } else if (v instanceof Object) {
            memo.push([k, JSON.stringify(v)]);
          } else {
            memo.push([k, `${v}`]);
          }
          return memo;
        },
        [] as [string, string][],
      );
      return new URLSearchParams(pairs);
    }, [params]);

    const Inner = () => {
      const result = useTypedParams(codec, {});
      useLayoutEffect(() => {
        outerRef.current = result;
      });
      return null;
    };

    return (
      <UserSettings.Provider
        value={{ isLoading: false, querySettings, state: observable(NotLoaded) }}>
        <BrowserRouter>
          <Inner />
        </BrowserRouter>
      </UserSettings.Provider>
    );
  };
  const renderResult = render(<Wrapper params={params} />);
  return {
    hookRef: outerRef,
    ...renderResult,
  };
};

describe('useTypedParams', () => {
  it.concurrent('parses the parameters properly', () => {
    fc.assert(
      fc.property(arb, (params) => {
        const { hookRef } = setupTest(params);
        // toEqual fails on -0 === +0: https://github.com/jestjs/jest/issues/12221
        expect(isEqual(hookRef.current?.params, params)).toBe(true);
      }),
    );
  });
  it('updates the state and url parameters on update if different', async () => {
    await fc.assert(
      fc
        .asyncProperty(diffArbPartial, async ([params, partial]) => {
          const { hookRef } = setupTest(params);
          await act(() => Promise.resolve(hookRef.current?.updateParams(partial)));
          const finalObject = { ...params, ...partial };
          expect(isEqual(hookRef.current?.params, finalObject)).toBe(true);
          expect(vi.mocked(navSpy)).toBeCalledTimes(1);
          const windowParams = navSpy.mock.lastCall[0](new URLSearchParams());
          Object.entries(finalObject).forEach(([k, v]) => {
            expect(windowParams.getAll(k)).toEqual(
              [v].flat().map((sv) => (typeof sv === 'string' ? sv.toString() : JSON.stringify(sv))),
            );
          });
        })
        .afterEach(() => navSpy.mockClear()),
    );
  });
  it('does not touch params that are not in the codec', async () => {
    await fc.assert(
      fc
        .asyncProperty(
          fc
            .tuple(fc.webQueryParameters(), diffArbPartial)
            .map(([query, [params, partial]]) => [query, params, partial] as const)
            .filter(([query, params]) => Object.keys(params).every((k) => !query.includes(k))),
          async ([query, params, partial]) => {
            const oldQuery = new URLSearchParams(query);
            const { hookRef } = setupTest(params);
            await act(() => Promise.resolve(hookRef.current?.updateParams(partial)));
            expect(vi.mocked(navSpy)).toBeCalledTimes(1);
            const windowParams = navSpy.mock.lastCall[0](oldQuery);
            [...oldQuery.entries()].forEach(([key, value]) => {
              expect(windowParams.get(key)).toBe(value);
            });
          },
        )
        .afterEach(() => navSpy.mockClear()),
    );
  });
  it('does not update if params are not changed', () => {
    fc.assert(
      fc
        .asyncProperty(sameArbPartial, async ([params, partial]) => {
          const { hookRef } = setupTest(params);
          await act(() => Promise.resolve(hookRef.current?.updateParams(partial)));
          expect(vi.mocked(navSpy)).not.toHaveBeenCalled();
        })
        .afterEach(() => navSpy.mockClear()),
    );
  });
});
