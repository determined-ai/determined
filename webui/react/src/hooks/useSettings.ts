import queryString from 'query-string';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

import { useStore } from 'contexts/Store';
import { getUserSetting, updateUserSetting } from 'services/api';
import { V1UserWebSetting } from 'services/api-ts-sdk';
import { UpdateUserSettingParams } from 'services/types';
import usePrevious from 'shared/hooks/usePrevious';
import { Primitive, RecordKey } from 'shared/types';
import { clone, hasObjectKeys, isBoolean, isEqual, isNumber,
  isString } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { Storage } from 'shared/utils/storage';
import handleError from 'utils/error';

import useStorage from './useStorage';

export enum BaseType {
  Boolean = 'Boolean',
  Float = 'Float',
  Integer = 'Integer',
  String = 'String',
}

enum PathChangeType {
  None = 'none',
  Push = 'push',
  Replace = 'replace',
}

interface UserSettingUpdate extends UpdateUserSettingParams {
  userId: number;
}

type GenericSettingsType = Primitive | Primitive[] | undefined;
type GenericSettings = Record<string, GenericSettingsType>;
type PathChange<T> = { querySettings: Partial<T>, type: PathChangeType }

/*
 * defaultValue     - Optional default value. `undefined` as ultimate default.
 * skipUrlEncoding  - Avoid preserving setting in the URL query param.
 * storageKey       - If provided, save/load setting into/from storage.
 * type.baseType    - How to decode the string-based query param.
 * type.isArray     - List based query params can be non-array.
 */
export interface SettingsConfigProp {
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
  applicableRoutespace?: string;
  settings: SettingsConfigProp[];
  storagePath: string;
}

/*
 * Provide the ability to override hook options with
 * dynamic values during initialization.
 */
export interface SettingsHookOptions {
  storagePath?: string;
}

export type UpdateSettings<T> = (newSettings: Partial<T>, push?: boolean) => void;

export interface SettingsHook<T> {
  activeSettings: (keys?: string[]) => string[];
  resetSettings: (keys?: string[]) => void;
  settings: T;
  updateSettings: UpdateSettings<T>;
}

export const validateBaseType = (type: BaseType, value: unknown): boolean => {
  if (type === BaseType.Boolean && isBoolean(value)) return true;
  if (type === BaseType.Float && isNumber(value)) return true;
  if (type === BaseType.Integer && isNumber(value) &&
      Math.ceil(value) === Math.floor(value)) return true;
  if (type === BaseType.String && isString(value)) return true;
  return false;
};

export const validateSetting = (config: SettingsConfigProp, value: unknown): boolean => {
  if (value === undefined) return true;
  if (config.type.isArray) {
    if (!Array.isArray(value)) return false;
    return value.every((val) => validateBaseType(config.type.baseType, val));
  }
  return validateBaseType(config.type.baseType, value);
};

export const getDefaultSettings = <T>(config: SettingsConfig, storage: Storage): T => {
  return config.settings.reduce((acc, prop) => {
    let defaultValue = prop.defaultValue;
    if (prop.storageKey) {
      defaultValue = storage.getWithDefault(prop.storageKey, defaultValue);
    }
    acc[prop.key] = defaultValue;
    return acc;
  }, {} as GenericSettings) as unknown as T;
};

export const queryParamToType = (type: BaseType, param: string | null): Primitive | undefined => {
  if (param == null) return undefined;
  if (type === BaseType.Boolean) return param === 'true';
  if (type === BaseType.Float || type === BaseType.Integer) {
    const value = type === BaseType.Float ? parseFloat(param) : parseInt(param);
    return !isNaN(value) ? value : undefined;
  }
  if (type === BaseType.String) return param;
  return undefined;
};

export const queryToSettings = <T>(config: SettingsConfig, query: string): T => {
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
          .map((value) => queryParamToType(baseType, value))
          .filter((value): value is Primitive => value !== undefined)
        : queryParamToType(baseType, paramValue);

      /*
       * When expecting an array, convert valid non-array values into an array.
       * Example - 'PULLING' => [ 'PULLING' ]
       */
      const normalizedValue = prop.type.isArray && queryValue != null && !Array.isArray(queryValue)
        ? [ queryValue ] : queryValue;

      if (normalizedValue !== undefined) acc[prop.key] = normalizedValue;
    } catch (e) {
      handleError(e, { silent: true, type: ErrorType.Ui });
    }

    return acc;
  }, {} as GenericSettings) as unknown as T;
};

export const settingsToQuery = <T>(config: SettingsConfig, settings: T): string => {
  const fullSettings = config.settings.reduce((acc, prop) => {
    // Save settings into query if there is value defined and is not the default value.
    const value = settings[prop.key as keyof T];
    const isDefault = isEqual(prop.defaultValue, value);
    acc[prop.key as keyof T] = !prop.skipUrlEncoding && !isDefault ? value : undefined;
    return acc;
  }, {} as Partial<T>);

  return queryString.stringify(fullSettings);
};

export const getConfigKeyMap = (config: SettingsConfig): Record<RecordKey, boolean> => {
  return config.settings.reduce((acc, prop) => {
    acc[prop.key] = true;
    return acc;
  }, {} as Record<RecordKey, boolean>);
};

const getNewQueryPath = (
  config: SettingsConfig,
  basePath: string,
  currentQuery: string,
  newQuery: string,
): string => {
  // Strip out existing config settings from the current query.
  const keyMap = getConfigKeyMap(config);
  const params = queryString.parse(currentQuery);
  const cleanParams = {} as Record<RecordKey, unknown>;
  Object.keys(params).forEach((key) => {
    if (!keyMap[key] && params[key]) cleanParams[key] = params[key];
  });

  // Add new query to the clean query.
  const cleanQuery = queryString.stringify(cleanParams);
  const queries = [ cleanQuery, newQuery ].filter((query) => !!query).join('&');
  return `${basePath}?${queries}`;
};

const defaultPathChange = {
  querySettings: {},
  type: PathChangeType.None,
};

const useSettings = <T>(config: SettingsConfig, options?: SettingsHookOptions): SettingsHook<T> => {
  const history = useHistory();
  const location = useLocation();
  const storage = useStorage(options?.storagePath || config.storagePath);
  const { auth: { user } } = useStore();
  const prevSearch = usePrevious(location.search, undefined);
  const prevUser = usePrevious(user, user);
  const [ settings, setSettings ] = useState<T>(() => getDefaultSettings<T>(config, storage));
  const [ pathChange, setPathChange ] = useState<PathChange<T>>(defaultPathChange);

  const configMap = useMemo(() => {
    return config.settings.reduce((acc, prop) => {
      acc[prop.key] = prop;
      return acc;
    }, {} as Record<RecordKey, SettingsConfigProp>);
  }, [ config.settings ]);

  /*
   * A setting is considered active if it is set to a value and the
   * value is not equivalent to a default value (if applicable).
   */
  const activeSettings = useCallback((keys?: string[]): string[] => {
    return config.settings.reduce((acc, prop) => {
      const key = prop.key as keyof T;
      const includesKey = !keys || keys.includes(prop.key);
      const isDefault = isEqual(settings[key], prop.defaultValue);
      if (includesKey && !isDefault) acc.push(prop.key);
      return acc;
    }, [] as string[]);
  }, [ config.settings, settings ]);

  const updateSettings = useCallback(async (partialSettings: Partial<T>, push = false) => {
    if (!location.pathname.includes(config.applicableRoutespace ?? '')) return;

    const changes = Object.keys(partialSettings) as (keyof T)[];
    const { internalSettings, querySettings, updates } = changes.reduce((acc, key) => {
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
        const persistedSetting: V1UserWebSetting = { key: config.storageKey };
        if (value === undefined || isDefault) {
          storage.remove(config.storageKey);
        } else {
          storage.set(config.storageKey, value);
          persistedSetting.value = JSON.stringify(value);
        }
        if (user?.id) {
          acc.updates.push({
            setting: persistedSetting,
            storagePath: storage.getStoragePath(),
            userId: user.id,
          });
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
      updates: [] as UserSettingUpdate[],
    });

    // Update user settings via API.
    if (updates.length !== 0) {
      try {
        // Persist storage to backend.
        await Promise.allSettled(updates.map((update) => updateUserSetting(update)));
      } catch (e) {
        handleError(e, {
          isUserTriggered: false,
          publicMessage: 'Unable to update user settings.',
          publicSubject: 'Some POST user settings failed.',
          silent: true,
          type: ErrorType.Api,
        });
      }
    }

    // Update internal settings state for when skipping url encoding of settings.
    setSettings((prev) => ({ ...clone(prev), ...internalSettings }));

    // Mark to trigger side effect of updating path.
    setPathChange({
      querySettings,
      type: push ? PathChangeType.Push : PathChangeType.Replace,
    });
  }, [ configMap, storage, user, config.applicableRoutespace, location.pathname ]);

  const resetSettings = useCallback(async (keys?: string[]) => {
    const newSettings = config.settings.reduce((acc, prop) => {
      const includesKey = !keys || keys.includes(prop.key);
      if (includesKey) acc[prop.key] = prop.defaultValue;
      return acc;
    }, {} as GenericSettings) as Partial<T>;

    await updateSettings(newSettings);
  }, [ config.settings, updateSettings ]);

  const fetchUserSetting = useCallback(async () => {
    if (!user || user === prevUser) return;
    try {
      const userSettingResponse = await getUserSetting({ userId: user.id });
      userSettingResponse.settings.forEach((setting) => {
        const {
          key,
          value,
          storagePath,
        } = setting;
        const jsonValue = JSON.parse(value || '');
        const config = configMap[key];
        if (!config) return;
        const isValid = validateSetting(config, jsonValue);
        const isDefault = isEqual(config.defaultValue, jsonValue);

        // Store or clear setting if `storageKey` is available.
        if (config.storageKey && isValid) {
          if (jsonValue === undefined || isDefault) {
            storage.remove(config.storageKey, storagePath);
          } else {
            storage.set(config.storageKey, jsonValue, storagePath);
          }
        }
      });
    } catch (e) {
      handleError(e, {
        isUserTriggered: false,
        publicMessage: 'Unable to fetch user settings.',
        publicSubject: 'GET user settings failed',
        silent: true,
        type: ErrorType.Api,
      });
    }

  }, [ configMap, prevUser, storage, user ]);

  useEffect(() => {
    fetchUserSetting();
  }, [ fetchUserSetting ]);

  useEffect(() => {
    if (location.search === prevSearch) return;

    // probably don't need this, we do need in updateSettings though
    if (!location.pathname.includes(config.applicableRoutespace ?? '')) return;

    /*
     * Set the initial query string if:
     * 1) current settings have set values
     * 2) there are no user specified query settings set
     *    (ignores defaults values since they are not user triggered)
     */
    const locationSearch = location.search.substr(/^\?/.test(location.search) ? 1 : 0);
    const currentQuery = settingsToQuery(config, settings);
    const searchSettings = queryToSettings(config, locationSearch);
    if (currentQuery && !hasObjectKeys(searchSettings)) {
      const newQueries = [ currentQuery ];
      if (locationSearch) newQueries.unshift(locationSearch);
      history.replace(`${location.pathname}?${newQueries.join('&')}`);
    } else {
      // Otherwise read settings from the query string.
      setSettings((prevSettings) => {
        const defaultSettings = getDefaultSettings<T>(config, storage);
        const querySettings = queryToSettings<Partial<T>>(config, locationSearch);
        return { ...prevSettings, ...defaultSettings, ...querySettings };
      });
    }
  }, [ config, history, location.pathname, location.search, prevSearch, settings, storage ]);

  useEffect(() => {
    if (pathChange.type === PathChangeType.None) return;

    // probably don't need this, we do need in updateSettings though
    if (!location.pathname.includes(config.applicableRoutespace ?? '')) return;

    // Update path with new and validated settings.
    const query = settingsToQuery(config, { ...clone(settings), ...pathChange.querySettings });
    const path = getNewQueryPath(config, location.pathname, location.search, query);
    pathChange.type === PathChangeType.Push ? history.push(path) : history.replace(path);

    // Reset path change.
    setPathChange(defaultPathChange);
  }, [ config, history, location.pathname, location.search, pathChange, settings ]);

  return { activeSettings, resetSettings, settings, updateSettings };
};

export default useSettings;
