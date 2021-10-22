import { act, renderHook, RenderResult } from '@testing-library/react-hooks';
import queryString from 'query-string';
import React from 'react';
import { Router } from 'react-router-dom';

import StoreProvider from 'contexts/Store';
import history from 'routes/history';
import { RecordKey } from 'types';
import { MemoryStore, Storage } from 'utils/storage';

import useSettings, * as hook from './useSettings';

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

const config: hook.SettingsConfig = {
  settings: [
    {
      defaultValue: true,
      key: 'boolean',
      type: { baseType: hook.BaseType.Boolean },
    },
    {
      key: 'booleanArray',
      storageKey: 'booleanArray',
      type: { baseType: hook.BaseType.Boolean, isArray: true },
    },
    {
      key: 'number',
      type: { baseType: hook.BaseType.Float },
    },
    {
      defaultValue: [ -5, 0, 1e10 ],
      key: 'numberArray',
      type: { baseType: hook.BaseType.Integer, isArray: true },
    },
    {
      defaultValue: 'foo bar',
      key: 'string',
      storageKey: 'string',
      type: { baseType: hook.BaseType.String },
    },
    {
      key: 'stringArray',
      storageKey: 'stringArray',
      type: { baseType: hook.BaseType.String, isArray: true },
    },
  ],
  storagePath: 'settings/normal',
};

const extraConfig: hook.SettingsConfig = {
  settings: [
    {
      defaultValue: 'what',
      key: 'extra',
      storageKey: 'extra',
      type: { baseType: hook.BaseType.String },
    },
  ],
  storagePath: 'settings/extra',
};

describe('useSettings helper functions', () => {
  describe('validateBaseType', () => {
    it('should validate base types for settings', () => {
      const tests = [
        { type: hook.BaseType.Boolean, value: false },
        { type: hook.BaseType.Boolean, value: true },
        { type: hook.BaseType.Float, value: 3.14159 },
        { type: hook.BaseType.Float, value: 1.5e-10 },
        { type: hook.BaseType.Float, value: -52.80 },
        { type: hook.BaseType.Float, value: -0.00321 },
        { type: [ hook.BaseType.Float, hook.BaseType.Integer ], value: 0 },
        { type: [ hook.BaseType.Float, hook.BaseType.Integer ], value: 123 },
        { type: [ hook.BaseType.Float, hook.BaseType.Integer ], value: -123 },
        { type: [ hook.BaseType.Float, hook.BaseType.Integer ], value: 5e12 },
        { type: [ hook.BaseType.Float, hook.BaseType.Integer ], value: -5e12 },
        { type: hook.BaseType.String, value: 'hello' },
        { type: hook.BaseType.String, value: 'The quick fox jumped over the lazy dog.' },
      ];
      (Object.keys(hook.BaseType) as hook.BaseType[]).forEach(baseType => {
        tests.forEach(test => {
          const result = Array.isArray(test.type)
            ? test.type.includes(baseType)
            : test.type === baseType;
          expect(hook.validateBaseType(baseType, test.value)).toBe(result);
        });
      });
    });

    it('should validate settings', () => {
      const arraySuffix = 'Array';
      const configs = Object.keys(hook.BaseType).reduce((acc, type) => {
        acc[type] = { type: { baseType: type } };
        acc[type + arraySuffix] = { type: { baseType: type, isArray: true } };
        return acc;
      }, {} as Record<RecordKey, unknown>);
      const tests = [
        { config: configs[hook.BaseType.Boolean], value: true },
        { config: configs[hook.BaseType.Boolean], value: false },
        { config: configs[hook.BaseType.Boolean + arraySuffix], value: [ true, false, true ] },
        { config: configs[hook.BaseType.Float], value: 3.14159 },
        { config: configs[hook.BaseType.Float], value: -1.2e-52 },
        { config: configs[hook.BaseType.Float + arraySuffix], value: [ 3.14159, -1e-52, 0 ] },
        {
          config: [ configs[hook.BaseType.Float], configs[hook.BaseType.Integer] ],
          value: 0,
        },
        {
          config: [ configs[hook.BaseType.Float], configs[hook.BaseType.Integer] ],
          value: 1024,
        },
        {
          config: [ configs[hook.BaseType.Float], configs[hook.BaseType.Integer] ],
          value: -2048,
        },
        {
          config: [
            configs[hook.BaseType.Float + arraySuffix],
            configs[hook.BaseType.Integer + arraySuffix],
          ],
          value: [ 1024, 0, -2048 ],
        },
        { config: configs[hook.BaseType.String], value: 'Hello' },
        { config: configs[hook.BaseType.String + arraySuffix], value: [ 'Hello', 'Jumping Dog' ] },
      ];
      Object.keys(configs).forEach(key => {
        const config = configs[key];
        tests.forEach(test => {
          const result = Array.isArray(test.config)
            ? test.config.includes(config)
            : test.config === config;
          expect(hook.validateSetting(config as hook.SettingsConfigProp, test.value)).toBe(result);
        });
      });
    });
  });

  describe('getDefaultSettings', () => {
    const testStorage = new Storage({ basePath: config.storagePath, store: new MemoryStore() });
    const defaultResult = {
      boolean: true,
      booleanArray: undefined,
      number: undefined,
      numberArray: [ -5, 0, 1e10 ],
      string: 'foo bar',
      stringArray: undefined,
    };

    it('should get settings from default values', () => {
      const defaultSettings = hook.getDefaultSettings<Settings>(config, testStorage);
      expect(defaultSettings).toStrictEqual(defaultResult);
    });

    it('should get settings from storage', () => {
      const storageStringArrayValue = [ 'hello', 'world' ];
      testStorage.set('stringArray', storageStringArrayValue);

      const defaultSettings = hook.getDefaultSettings<Settings>(config, testStorage);
      const result = {
        ...defaultResult,
        stringArray: storageStringArrayValue,
      };
      expect(defaultSettings).toStrictEqual(result);
    });
  });
});

describe('useSettings', () => {
  const newSettings = {
    boolean: false,
    booleanArray: [ false, true ],
    number: 3.14e-12,
    numberArray: [ 0, 100, -5280 ],
    string: 'Hello World',
    stringArray: [ 'abc', 'def', 'ghi' ],
  };
  const newExtraSettings = { extra: 'fancy' };
  let result: RenderResult<hook.SettingsHook<Settings>>;
  let extraResult: RenderResult<hook.SettingsHook<ExtraSettings>>;

  beforeEach(() => {
    const RouterWrapper: React.FC = ({ children }) => (
      <StoreProvider>
        <Router history={history}>{children}</Router>
      </StoreProvider>
    );
    const hookResult = renderHook(
      () => useSettings<Settings>(config),
      { wrapper: RouterWrapper },
    );
    const extraHookResult = renderHook(
      () => useSettings<ExtraSettings>(extraConfig),
      { wrapper: RouterWrapper },
    );
    result = hookResult.result;
    extraResult = extraHookResult.result;
  });

  it('should have default settings', () => {
    config.settings.forEach(configProp => {
      const settingsKey = configProp.key as keyof Settings;
      expect(result.current.settings[settingsKey]).toStrictEqual(configProp.defaultValue);
    });

    expect(history.location.search).toBe('');
  });

  it('should update settings', () => {
    act(() => result.current.updateSettings(newSettings));

    config.settings.forEach(configProp => {
      const settingsKey = configProp.key as keyof Settings;
      expect(result.current.settings[settingsKey])
        .toStrictEqual(newSettings[settingsKey]);
    });

    expect(history.location.search).toContain([
      'boolean=false',
      'booleanArray=false&booleanArray=true',
      'number=3.14e-12',
      'numberArray=0&numberArray=100&numberArray=-5280',
      'string=Hello%20World',
      'stringArray=abc&stringArray=def&stringArray=ghi',
    ].join('&'));
  });

  it('should keep track of active settings', () => {
    expect(result.current.activeSettings()).toStrictEqual(Object.keys(newSettings));
  });

  it('should have default settings after reset', () => {
    act(() => result.current.resetSettings());

    config.settings.forEach(configProp => {
      const settingsKey = configProp.key as keyof Settings;
      expect(result.current.settings[settingsKey]).toStrictEqual(configProp.defaultValue);
    });
  });

  it('should be able to keep track of multiple settings', () => {
    act(() => {
      result.current.updateSettings(newSettings);
      extraResult.current.updateSettings(newExtraSettings);
    });

    config.settings.forEach(configProp => {
      const settingsKey = configProp.key as keyof Settings;
      expect(result.current.settings[settingsKey])
        .toStrictEqual(newSettings[settingsKey]);
    });

    extraConfig.settings.forEach(configProp => {
      const settingsKey = configProp.key as keyof ExtraSettings;
      expect(extraResult.current.settings[settingsKey])
        .toStrictEqual(newExtraSettings[settingsKey]);
    });

    expect(history.location.search).toContain([
      'boolean=false',
      'booleanArray=false&booleanArray=true',
      'number=3.14e-12',
      'numberArray=0&numberArray=100&numberArray=-5280',
      'string=Hello%20World',
      'stringArray=abc&stringArray=def&stringArray=ghi',
    ].join('&'));

    expect(history.location.search).toContain('extra=fancy');
  });

  it('should pick up query param changes and read new settings', () => {
    const newQueryParams = {
      boolean: true,
      extra: 'donut',
      number: 500,
    };
    const newQuery = queryString.stringify(newQueryParams);

    act(() => {
      result.current.resetSettings();
      history.replace(`${history.location.pathname}?${newQuery}`);
    });

    expect(history.location.search).toBe(`?${newQuery}`);
    expect(result.current.settings.boolean).toBe(newQueryParams.boolean);
    expect(result.current.settings.number).toBe(newQueryParams.number);
    expect(result.current.settings.string)
      .toBe(config.settings.find(setting => setting.key === 'string')?.defaultValue);
    expect(extraResult.current.settings.extra).toBe(newQueryParams.extra);
  });
});
