import queryString from 'query-string';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

import { Primitive, RecordKey } from 'types';
import { clone, isBoolean, isEqual, isNumber, isString } from 'utils/data';
import { Storage } from 'utils/storage';

import usePrevious from './usePrevious';
import useStorage from './useStorage';

export enum BaseType {
  Boolean = 'Boolean',
  Float = 'Float',
  Integer = 'Integer',
  String = 'String',
}

type GenericSettingsType = Primitive | Primitive[] | undefined;

/*
 * defaultValue     - Optional default value. `undefined` as ultimate default.
 * skipUrlEncoding  - Avoid preserving setting in the URL query param.
 * storageKey       - If provided, save/load setting into/from storage.
 * type.baseType    - How to decode the string-based query param.
 * type.isArray     - List based query params can be non-array.
 */
interface SettingsConfigProp {
  defaultValue?: GenericSettingsType;
  key: string;
  skipUrlEncoding?: boolean;
  storageKey?: string;
  type: {
    baseType: BaseType;
    isArray?: boolean;
  };
}

export interface SettingsConfig {
  settings: SettingsConfigProp[];
  storagePath: string;
}

type GenericSettings = Record<string, GenericSettingsType>;

interface SettingsHook<T> {
  resetSettings: (keys?: string[]) => void;
  settings: T;
  settingsCount: (keys?: string[]) => number;
  updateSettings: (newSettings: Partial<T>, push?: boolean) => void;
}

const getDefaultSettings = <T>(config: SettingsConfig, storage: Storage): T => {
  return config.settings.reduce((acc, prop) => {
    let defaultValue = prop.defaultValue;
    if (prop.storageKey) {
      defaultValue = storage.getWithDefault(prop.storageKey, defaultValue);
    }
    acc[prop.key] = defaultValue;
    return acc;
  }, {} as GenericSettings) as unknown as T;
};

const queryParamToType = (type: BaseType, param: string | null): Primitive | undefined => {
  if (param == null) return undefined;
  if (type === BaseType.Boolean) return param === 'true';
  if (type === BaseType.Float || type === BaseType.Integer) {
    const value = type === BaseType.Float ? parseFloat(param) : parseInt(param);
    return !isNaN(value) ? value : undefined;
  }
  if (type === BaseType.String) return param;
  return undefined;
};

const queryToSettings = <T>(query: string, config: SettingsConfig): T => {
  const params = queryString.parse(query);
  return config.settings.reduce((acc, prop) => {
    /*
     * Attempt to decode the query parameter and if anything
     * goes wrong, set it to the default value.
     */
    try {
      const paramValue = params[prop.key];
      const baseType = prop.type.baseType;

      /*
       * Convert the string-based query params to primitives.
       * `undefined` values can happen if the query param values are invalid.
       *   string[] => Primitive[] | undefined
       *   string   => Primitive | undefined
       *   null     => undefined
       */
      const queryValue = Array.isArray(paramValue)
        ? paramValue
          .map(value => queryParamToType(baseType, value))
          .filter((value): value is Primitive => value !== undefined)
        : queryParamToType(baseType, paramValue);

      /*
       * When expecting an array, convert valid non-array values into an array.
       * Example - 'PULLING' => [ 'PULLING' ]
       */
      const normalizedValue = prop.type.isArray && queryValue != null && !Array.isArray(queryValue)
        ? [ queryValue ] : queryValue;

      if (normalizedValue !== undefined) acc[prop.key] = normalizedValue;
    } catch (e) {}

    return acc;
  }, {} as GenericSettings) as unknown as T;
};

const settingsToQuery = <T>(config: SettingsConfig, settings: T): string => {
  const fullSettings = config.settings.reduce((acc, prop) => {
    // Save settings into query if there is value defined and is not the default value.
    const value = settings[prop.key as keyof T];
    const isDefault = isEqual(prop.defaultValue, value);
    acc[prop.key as keyof T] = !prop.skipUrlEncoding && !isDefault ? value : undefined;
    return acc;
  }, {} as Partial<T>);

  return queryString.stringify(fullSettings);
};

const validateBaseType = (type: BaseType, value: unknown): boolean => {
  if (type === BaseType.Boolean && isBoolean(value)) return true;
  if (type === BaseType.Float && isNumber(value)) return true;
  if (type === BaseType.Integer && isNumber(value)) return true;
  if (type === BaseType.String && isString(value)) return true;
  return false;
};

const validateSetting = (config: SettingsConfigProp, value: unknown): boolean => {
  if (value === undefined) return true;
  if (config.type.isArray) {
    if (!Array.isArray(value)) return false;
    return value.every(val => validateBaseType(config.type.baseType, val));
  }
  return validateBaseType(config.type.baseType, value);
};

const useSettings = <T>(config: SettingsConfig, basePath: string): SettingsHook<T> => {
  const history = useHistory();
  const location = useLocation();
  const storage = useStorage(config.storagePath);
  const prevSearch = usePrevious(location.search, undefined);
  const [ settings, setSettings ] = useState<T>(() => getDefaultSettings<T>(config, storage));

  const configMap = useMemo(() => {
    return config.settings.reduce((acc, prop) => {
      acc[prop.key] = prop;
      return acc;
    }, {} as Record<RecordKey, SettingsConfigProp>);
  }, [ config.settings ]);

  const updateSettings = useCallback((partialSettings: Partial<T>, push = false) => {
    const changes = Object.keys(partialSettings) as (keyof T)[];
    const { internalSettings, querySettings } = changes.reduce((acc, key) => {
      // Check to make sure the settings key is defined in the config.
      const config = configMap[key];
      if (!config) return acc;

      // Set default settings to be undefined.
      acc.internalSettings[key] = undefined;
      acc.querySettings[key] = undefined;

      // If the settings value is invalid, set to undefined.
      const value = partialSettings[key];
      const isValid = validateSetting(config, value);
      const isDefault = isEqual(config.defaultValue, value);

      // Store or clear setting if `storageKey` is available.
      if (config.storageKey && isValid) {
        if (value === undefined || isDefault) {
          storage.remove(config.storageKey);
        } else {
          storage.set(config.storageKey, value);
        }
      }

      // Keep track of internal setting changes to update async from query settings.
      if (isValid) acc.internalSettings[key] = value;

      // Preserve the setting for updating query params unless `skipUrlEncoding` is set.
      if (!config.skipUrlEncoding && !isDefault && isValid) acc.querySettings[key] = value;

      return acc;
    }, {
      internalSettings: {} as Partial<T>,
      querySettings: {} as Partial<T>,
    });

    // Update internal settings state for when skipping url encoding of settings.
    setSettings({ ...clone(settings), ...internalSettings });

    // Update path with new and validated settings.
    const query = settingsToQuery(config, { ...clone(settings), ...querySettings });
    const path = `${basePath}?${query}`;
    push ? history.push(path) : history.replace(path);
  }, [ config, configMap, basePath, history, settings, storage ]);

  const resetSettings = useCallback((keys?: string[]) => {
    const newSettings = config.settings.reduce((acc, prop) => {
      const includesKey = !keys || keys.includes(prop.key);
      if (includesKey) acc[prop.key] = prop.defaultValue;
      return acc;
    }, {} as GenericSettings) as Partial<T>;

    updateSettings(newSettings);
  }, [ config.settings, updateSettings ]);

  const settingsCount = useCallback((keys?: string[]) => {
    return config.settings.reduce((acc, prop) => {
      const key = prop.key as keyof T;
      const includesKey = !keys || keys.includes(prop.key);
      const isDefault = isEqual(settings[key], prop.defaultValue);
      return acc + (includesKey && !isDefault ? 1 : 0);
    }, 0);
  }, [ config.settings, settings ]);

  useEffect(() => {
    if (location.search === prevSearch) return;

    /*
     * Set the initial query string if query settings are detected
     * but not found in the url query string.
     */
    const currentQuery = settingsToQuery(config, settings);
    if (location.search === '' && location.search !== currentQuery) {
      history.replace(`${basePath}?${currentQuery}`);
    } else {
      // Otherwise read settings from the query string.
      setSettings(prevSettings => {
        const querySettings = queryToSettings<Partial<T>>(location.search, config);
        const defaultSettings = getDefaultSettings<T>(config, storage);
        return { ...prevSettings, ...defaultSettings, ...querySettings };
      });
    }
  }, [ basePath, config, history, location.search, prevSearch, settings, storage ]);

  return { resetSettings, settings, settingsCount, updateSettings };
};

export default useSettings;
