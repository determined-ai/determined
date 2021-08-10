import { RecordKey } from 'types';
import { MemoryStore, Storage } from 'utils/storage';

import * as hook from './useSettings';

interface Settings {
  'boolean': boolean;
  'boolean-array'?: boolean[];
  'number'?: number;
  'number-array': number[];
  'string'?: string;
  'string-array'?: string[];
}

const config: hook.SettingsConfig = {
  settings: [
    {
      defaultValue: true,
      key: 'boolean',
      type: { baseType: hook.BaseType.Boolean },
    },
    {
      key: 'boolean-array',
      storageKey: 'boolean-array',
      type: { baseType: hook.BaseType.Boolean, isArray: true },
    },
    {
      key: 'number',
      type: { baseType: hook.BaseType.Float },
    },
    {
      defaultValue: [ -5, 0, 1e10 ],
      key: 'number-array',
      type: { baseType: hook.BaseType.Integer, isArray: true },
    },
    {
      defaultValue: 'foo bar',
      key: 'string',
      storageKey: 'string',
      type: { baseType: hook.BaseType.String },
    },
    {
      key: 'string-array',
      storageKey: 'string-array',
      type: { baseType: hook.BaseType.String, isArray: true },
    },
  ],
  storagePath: 'storage-path',
};

describe('useSettings', () => {
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
      'boolean': true,
      'boolean-array': undefined,
      'number': undefined,
      'number-array': [ -5, 0, 1e10 ],
      'string': 'foo bar',
      'string-array': undefined,
    };

    it('should get settings from default values', () => {
      const defaultSettings = hook.getDefaultSettings<Settings>(config, testStorage);
      expect(defaultSettings).toStrictEqual(defaultResult);
    });

    it('should get settings from storage', () => {
      const storageStringArrayValue = [ 'hello', 'world' ];
      testStorage.set('string-array', storageStringArrayValue);

      const defaultSettings = hook.getDefaultSettings<Settings>(config, testStorage);
      const result = {
        ...defaultResult,
        'string-array': storageStringArrayValue,
      };
      expect(defaultSettings).toStrictEqual(result);
    });
  });
});
