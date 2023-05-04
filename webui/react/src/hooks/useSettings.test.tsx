import { waitFor } from '@testing-library/react';
import { act, renderHook, RenderResult } from '@testing-library/react-hooks';
import { array, boolean, number, string, undefined as undefinedType, union } from 'io-ts';
import React, { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import authStore from 'stores/auth';
import userStore from 'stores/users';

import * as hook from './useSettings';
import { SettingsProvider } from './useSettingsProvider';

const CURRENT_USER = { id: 1, isActive: true, isAdmin: false, username: 'bunny' };

vi.mock('services/api', () => ({
  getUserSetting: () => Promise.resolve({ settings: [] }),
}));

interface Settings {
  boolean: boolean;
  booleanArray?: boolean[];
  number?: number;
  numberArray: number[];
  string?: string;
  stringArray?: string[];
}

interface ExtraSettings {
  extra: string;
}

type HookReturn = {
  container: RenderResult<hook.UseSettingsReturn<Settings>>;
  rerender: (
    props?:
      | {
          children: JSX.Element;
        }
      | undefined,
  ) => void;
};
type ExtraHookReturn = {
  container: RenderResult<hook.UseSettingsReturn<ExtraSettings>>;
  rerender: (
    props?:
      | {
          children: JSX.Element;
        }
      | undefined,
  ) => void;
};

const config: hook.SettingsConfig<Settings> = {
  settings: {
    boolean: {
      defaultValue: true,
      storageKey: 'boolean',
      type: boolean,
    },
    booleanArray: {
      defaultValue: undefined,
      storageKey: 'booleanArray',
      type: union([array(boolean), undefinedType]),
    },
    number: {
      defaultValue: undefined,
      storageKey: 'number',
      type: union([undefinedType, number]),
    },
    numberArray: {
      defaultValue: [-5, 0, 1e10],
      storageKey: 'numberArray',
      type: array(number),
    },
    string: {
      defaultValue: 'foo bar',
      storageKey: 'string',
      type: union([undefinedType, string]),
    },
    stringArray: {
      defaultValue: undefined,
      storageKey: 'stringArray',
      type: union([undefinedType, array(string)]),
    },
  },
  storagePath: 'settings-normal',
};

const extraConfig: hook.SettingsConfig<ExtraSettings> = {
  settings: {
    extra: {
      defaultValue: 'what',
      storageKey: 'extra',
      type: string,
    },
  },
  storagePath: 'settings-extra',
};

const Container: React.FC<{ children: JSX.Element }> = ({ children }) => {
  useEffect(() => {
    authStore.setAuth({ isAuthenticated: true });
    authStore.setAuthChecked();
    userStore.updateCurrentUser(CURRENT_USER);
  }, []);

  return (
    <SettingsProvider>
      <BrowserRouter>{children}</BrowserRouter>
    </SettingsProvider>
  );
};

const setup = (
  newSettings?: hook.SettingsConfig<Settings>,
  newExtraSettings?: hook.SettingsConfig<ExtraSettings>,
): {
  extraResult: ExtraHookReturn;
  result: HookReturn;
} => {
  const RouterWrapper: React.FC<{ children: JSX.Element }> = ({ children }) => (
    <UIProvider>
      <Container>{children}</Container>
    </UIProvider>
  );
  const hookResult = renderHook(() => hook.useSettings<Settings>(newSettings ?? config), {
    wrapper: RouterWrapper,
  });
  const extraHookResult = renderHook(
    () => hook.useSettings<ExtraSettings>(newExtraSettings ?? extraConfig),
    {
      wrapper: RouterWrapper,
    },
  );

  return {
    extraResult: { container: extraHookResult.result, rerender: extraHookResult.rerender },
    result: { container: hookResult.result, rerender: hookResult.rerender },
  };
};

describe('useSettings', () => {
  const newSettings = {
    boolean: false,
    booleanArray: [false, true],
    number: 3.14e-12,
    numberArray: [0, 100, -5280],
    string: 'Hello World',
    stringArray: ['abc', 'def', 'ghi'],
  };
  const newExtraSettings = { extra: 'fancy' };

  afterEach(() => vi.clearAllMocks());

  it('should have default settings', () => {
    const { result } = setup();
    Object.values(config.settings).forEach((configProp) => {
      const settingsKey = configProp.storageKey as keyof Settings;
      expect(result.container.current.settings[settingsKey]).toStrictEqual(configProp.defaultValue);
    });

    expect(window.location.search).toBe('');
  });

  it('should have default settings after reset', async () => {
    const { result } = setup();
    act(() => result.container.current.resetSettings());

    for (const configProp of Object.values(config.settings)) {
      const settingsKey = configProp.storageKey as keyof Settings;
      await waitFor(() =>
        expect(result.container.current.settings[settingsKey]).toStrictEqual(
          configProp.defaultValue,
        ),
      );
    }
  });

  it('should update settings', async () => {
    const { result } = setup();
    act(() => result.container.current.updateSettings(newSettings));

    for (const configProp of Object.values(config.settings)) {
      const settingsKey = configProp.storageKey as keyof Settings;
      await waitFor(() =>
        expect(result.container.current.settings[settingsKey]).toStrictEqual(
          newSettings[settingsKey],
        ),
      );
    }

    await waitFor(() => {
      expect(window.location.search).toContain(
        [
          'boolean=false',
          'booleanArray=false&booleanArray=true',
          'number=3.14e-12',
          'numberArray=0&numberArray=100&numberArray=-5280',
          'string=Hello+World',
          'stringArray=abc&stringArray=def&stringArray=ghi',
        ].join('&'),
      );
    });
  });

  it('should keep track of active settings', async () => {
    const { result } = setup();
    act(() => result.container.current.updateSettings(newSettings));

    await waitFor(() =>
      expect(result.container.current.activeSettings()).toStrictEqual(Object.keys(newSettings)),
    );
  });

  it('should be able to keep track of multiple settings', async () => {
    const { result, extraResult } = await setup();
    act(() => {
      result.container.current.updateSettings(newSettings);
      extraResult.container.current.updateSettings(newExtraSettings);
    });

    for (const configProp of Object.values(config.settings)) {
      const settingsKey = configProp.storageKey as keyof Settings & keyof ExtraSettings;
      await waitFor(() =>
        expect(result.container.current.settings[settingsKey]).toStrictEqual(
          newSettings[settingsKey],
        ),
      );
      await waitFor(() =>
        expect(extraResult.container.current.settings[settingsKey]).toStrictEqual(
          newExtraSettings[settingsKey],
        ),
      );
    }
  });
});
