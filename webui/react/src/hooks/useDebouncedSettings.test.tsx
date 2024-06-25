import { act, render } from '@testing-library/react';
import { Loaded, NotLoaded } from 'hew/utils/loadable';
import * as t from 'io-ts';
import { useEffect, useLayoutEffect } from 'react';

import { getUserSetting, updateUserSetting } from 'services/api';
import userSettings from 'stores/userSettings';

import { useDebouncedSettings } from './useDebouncedSettings';

vi.mock('services/api', async () => ({
  ...(await vi.importActual('services/api')),
  getUserSetting: vi.fn(() => new Promise(() => {})),
  resetUserSetting: vi.fn(() => new Promise(() => {})),
  updateUserSetting: vi.fn(() => new Promise(() => {})),
}));

const setup = <T extends t.HasProps>(type: T, path: string, initialSettings: t.TypeOf<T>) => {
  vi.mocked(getUserSetting).mockImplementation(() => {
    return Promise.resolve({
      settings: Object.entries(initialSettings).map(([key, v]) => ({
        key,
        storagePath: path,
        value: JSON.stringify(v),
      })),
    });
  });
  const outerRef: { current: null | ReturnType<typeof useDebouncedSettings> } = { current: null };

  const Element = () => {
    const hookVal = useDebouncedSettings(type, path);
    useEffect(() => {
      return userSettings.startPolling({ delay: 1000 });
    }, []);
    useLayoutEffect(() => {
      outerRef.current = hookVal;
    });

    return null;
  };

  render(<Element />);

  return outerRef;
};

describe('useDebouncedSettings', () => {
  beforeEach(() => {
    userSettings.reset();
    vi.mocked(getUserSetting).mockClear();
    vi.mocked(updateUserSetting).mockClear();
  });
  beforeAll(() => {
    vi.useFakeTimers();
  });
  afterAll(() => {
    vi.useRealTimers();
  });

  it('reads server data', async () => {
    const type = t.type({ bar: t.number, foo: t.string });
    const expected: t.TypeOf<typeof type> = {
      bar: 100,
      foo: 'foo',
    };
    const ref = setup(type, 'test1', expected);
    expect(ref.current?.[0]).toEqual(NotLoaded);
    await act(() => vi.advanceTimersByTimeAsync(1000));
    expect(ref.current?.[0]).toEqual(Loaded(expected));
  });

  it("doesn't take local updates if server is unloaded", () => {
    const type = t.type({ bar: t.number, foo: t.string });
    const ref = setup(type, 'test1', {
      bar: 100,
      foo: 'foo',
    });
    const expected = {
      bar: 200,
    };

    act(() => {
      ref.current?.[1](expected);
    });
    expect(ref.current?.[0]).toEqual(NotLoaded);
  });

  it('updates local state after initial load', async () => {
    const type = t.type({ bar: t.number, foo: t.string });
    const ref = setup(type, 'test1', {
      bar: 100,
      foo: 'foo',
    });
    const expected = {
      bar: 200,
    };

    await act(() => vi.advanceTimersByTimeAsync(1000));
    act(() => {
      ref.current?.[1](expected);
    });
    expect(ref.current?.[0]).toEqual(Loaded({ bar: 200, foo: 'foo' }));
  });

  it('deep equals checks before returning a new object', async () => {
    const type = t.type({ bar: t.number, foo: t.string });
    const ref = setup(type, 'test1', {
      bar: 100,
      foo: 'foo',
    });
    const expected = {
      foo: 'foo',
    };

    await act(() => vi.advanceTimersByTimeAsync(1000));
    const initialValue = ref.current?.[0];
    expect(initialValue).toEqual(Loaded({ bar: 100, foo: 'foo' }));
    act(() => {
      ref.current?.[1](expected);
    });
    expect(ref.current?.[0]).toBe(initialValue);

    act(() => {
      ref.current?.[1]({ foo: 'bar' });
    });
    expect(ref.current?.[0]).not.toBe(initialValue);
  });
});
