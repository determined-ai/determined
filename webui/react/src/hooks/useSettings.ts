import queryString from 'query-string';
import { useCallback, useEffect, useState } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

import { Primitive } from 'types';
import { clone } from 'utils/data';

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
 * defaultValue   - Must be provided even when `undefined`.
 * storageKey     - If provide, save/load setting into/from storage.
 * type.baseType  - How to decode the string-based query param.
 * type.isArray   - List based query params can be non-array.
 */
export interface SettingsConfigProp {
  defaultValue: GenericSettingsType;
  key: string;
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
  settings: T;
  updateSettings: (newSettings: Partial<T>, push?: boolean) => void;
}

const queryParamToType = (type: BaseType, param: string | null): Primitive | undefined => {
  if (param === null) return undefined;
  if (type === BaseType.Boolean) return param === 'true';
  if (type === BaseType.Float || type === BaseType.Integer) {
    const value = type === BaseType.Float ? parseFloat(param) : parseInt(param);
    return !isNaN(value) ? value : undefined;
  }
  if (type === BaseType.String) return param;
  return undefined;
};

const queryToSettings = <T>(
  query: string,
  config: SettingsConfig,
  defaultSettings?: GenericSettings,
): T => {
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

      acc[prop.key] = normalizedValue;
    } catch (e) {
      if (defaultSettings && defaultSettings[prop.key] !== undefined) {
        acc[prop.key] = defaultSettings[prop.key];
      }
    }
    return acc;
  }, {} as GenericSettings) as unknown as T;
};

const useSettings = <T>(
  config: SettingsConfig,
  basePath: string,
): SettingsHook<T> => {
  const history = useHistory();
  const location = useLocation();
  const storage = useStorage(config.storagePath);
  const prevSearch = usePrevious(location.search, undefined);
  const [ settings, setSettings ] = useState<T>(() => {
    return config.settings.reduce((acc, config) => {
      let value = config.defaultValue;

      // Pull from storage if config storage key is defined.
      if (config.storageKey) {
        value = storage.getWithDefault(config.storageKey, value);
      }
      acc[config.key] = value;
      return acc;
    }, {} as GenericSettings) as unknown as T;
  });

  const updateSettings = useCallback((partialSettings: Partial<T>, push = false) => {
    const newSettings = { ...clone(settings), ...partialSettings };
    const path = `${basePath}?${queryString.stringify(newSettings)}`;
    push ? history.push(path) : history.replace(path);
  }, [ basePath, history, settings ]);

  useEffect(() => {
    if (!location.search || location.search === prevSearch) return;

    setSettings(prevSettings => {
      const querySettings = queryToSettings<T>(
        location.search,
        config,
        prevSettings as unknown as GenericSettings,
      );
      return { ...prevSettings, ...querySettings };
    });
  }, [ config, location.search, prevSearch ]);

  return { settings, updateSettings };
};

export default useSettings;
