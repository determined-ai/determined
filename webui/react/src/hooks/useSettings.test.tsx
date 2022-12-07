import { waitFor } from '@testing-library/react';
import { act, renderHook, RenderResult } from '@testing-library/react-hooks';
import { array, boolean, number, string, undefined as undefinedType, union } from 'io-ts';
import React from 'react';
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import StoreProvider from 'contexts/Store';
import history from 'shared/routes/history';
import { DetailedUser } from 'types';

import * as hook from './useSettings';
import { SettingsProvider } from './useSettingsProvider';

jest.mock('services/api', () => ({
  ...jest.requireActual('services/api'),
  getUserSetting: () => Promise.resolve({ settings: [] }),
}));
jest.mock('contexts/Store', () => ({
  __esModule: true,
  ...jest.requireActual('contexts/Store'),
  useStore: () => ({ auth: { user: { id: 1 } as DetailedUser } }),
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
  applicableRoutespace: 'settings/normal',
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
  storagePath: 'settings/normal',
};

const extraConfig: hook.SettingsConfig<ExtraSettings> = {
  applicableRoutespace: 'settings/extra',
  settings: {
    extra: {
      defaultValue: 'what',
      storageKey: 'extra',
      type: string,
    },
  },
  storagePath: 'settings/extra',
};

const setup = async (
  newSettings?: hook.SettingsConfig<Settings>,
  newExtraSettings?: hook.SettingsConfig<ExtraSettings>,
): Promise<{
  extraResult: ExtraHookReturn;
  result: HookReturn;
}> => {
  const RouterWrapper: React.FC<{ children: JSX.Element }> = ({ children }) => (
    <StoreProvider>
      <SettingsProvider>
        <HistoryRouter history={history}>{children}</HistoryRouter>
      </SettingsProvider>
    </StoreProvider>
  );
  const hookResult = await renderHook(() => hook.useSettings<Settings>(newSettings ?? config), {
    wrapper: RouterWrapper,
  });
  const extraHookResult = await renderHook(
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

  afterEach(() => jest.clearAllMocks());

  it('should have default settings', async () => {
    const { result } = await setup();
    Object.values(config.settings).forEach((configProp) => {
      const settingsKey = configProp.storageKey as keyof Settings;
      expect(result.container.current.settings[settingsKey]).toStrictEqual(configProp.defaultValue);
    });

    expect(history.location.search).toBe('');
  });

  it('should update settings', async () => {
    const { result } = await setup();
    await act(() => result.container.current.updateSettings(newSettings));

    Object.values(config.settings).forEach((configProp) => {
      const settingsKey = configProp.storageKey as keyof Settings;
      waitFor(() =>
        expect(result.container.current.settings[settingsKey]).toStrictEqual(
          newSettings[settingsKey],
        ),
      );
    });

    waitFor(() => {
      expect(history.location.search).toContain(
        [
          'boolean=false',
          'booleanArray=false&booleanArray=true',
          'number=3.14e-12',
          'numberArray=0&numberArray=100&numberArray=-5280',
          'string=Hello%20World',
          'stringArray=abc&stringArray=def&stringArray=ghi',
        ].join('&'),
      );
    });
  });

  it('should keep track of active settings', async () => {
    const { result } = await setup();
    await act(() => result.container.current.updateSettings(newSettings));

    waitFor(() =>
      expect(result.container.current.activeSettings()).toStrictEqual(Object.keys(newSettings)),
    );
  });

  it('should have default settings after reset', async () => {
    const { result } = await setup();
    await act(() => result.container.current.resetSettings());

    Object.values(config.settings).forEach(async (configProp) => {
      const settingsKey = configProp.storageKey as keyof Settings;
      await waitFor(() =>
        expect(result.container.current.settings[settingsKey]).toStrictEqual(
          configProp.defaultValue,
        ),
      );
    });
  });

  it('should be able to keep track of multiple settings', async () => {
    const { result, extraResult } = await setup();
    await act(() => {
      result.container.current.updateSettings(newSettings);
      extraResult.container.current.updateSettings(newExtraSettings);
    });

    Object.values(config.settings).forEach((configProp) => {
      const settingsKey = configProp.storageKey as keyof Settings;
      waitFor(() =>
        expect(result.container.current.settings[settingsKey]).toStrictEqual(
          newSettings[settingsKey],
        ),
      );
    });

    Object.values(config.settings).forEach((configProp) => {
      const settingsKey = configProp.storageKey as keyof ExtraSettings;
      waitFor(() =>
        expect(extraResult.container.current.settings[settingsKey]).toStrictEqual(
          newExtraSettings[settingsKey],
        ),
      );
    });
  });
});
