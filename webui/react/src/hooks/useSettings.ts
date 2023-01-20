import * as t from 'io-ts';
import queryString from 'query-string';
import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

import { updateUserSetting } from 'services/api';
import { UpdateUserSettingParams } from 'services/types';
import { Primitive } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { useCurrentUser } from 'stores/users';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

import { Settings, SettingsProvider, UserSettings } from './useSettingsProvider';

export interface SettingsConfigProp<A> {
  defaultValue: A;
  skipUrlEncoding?: boolean;
  storageKey: string;
  type: t.Type<A>;
}

export interface SettingsConfig<T> {
  applicableRoutespace?: string;
  settings: { [K in keyof T]: SettingsConfigProp<T[K]> };
  storagePath: string;
}

interface UserSettingUpdate extends UpdateUserSettingParams {
  userId: number;
}

export type UpdateSettings = (updates: Settings, shouldPush?: boolean) => void;
export type ResetSettings = (settings?: string[]) => void;
type SettingsRecord<T> = { [K in keyof T]: T[K] };
type SettingKeyType<T> = t.Type<SettingsConfig<T>, SettingsConfig<T>, unknown>;

export type UseSettingsReturn<T> = {
  activeSettings: (keys?: string[]) => string[];
  isLoading: boolean;
  resetSettings: ResetSettings;
  settings: T;
  updateSettings: UpdateSettings;
};

const settingsToQuery = <T>(config: SettingsConfig<T>, settings: Settings) => {
  const fullSettings = (Object.values(config.settings) as SettingsConfigProp<T>[]).reduce<Settings>(
    (acc, setting) => {
      // Save settings into query if there is value defined and is not the default value.
      const value = settings[setting.storageKey];
      const isDefault = isEqual(setting.defaultValue, value);

      acc[setting.storageKey] = !setting.skipUrlEncoding && !isDefault ? value : undefined;

      return acc;
    },
    {},
  );

  return queryString.stringify(fullSettings);
};

const queryParamToType = <T>(
  type: t.Type<SettingsConfig<T>, SettingsConfig<T>, unknown>, // type is refferent to each settign key
  param: string | null, // is refering to the value entry, which can be an index of a setting key if said setting is an "array of something"
): Primitive | undefined => {
  const validateLiteralType = <T>(
    type: SettingKeyType<T>,
    param: string,
  ): Primitive | undefined => {
    const typeName = type.name.replace(/(\(|\))/g, ''); // just in case it is a union type
    let parsedValue: Primitive | undefined;
    const checkTypes = (t: string) => {
      const possibleLiteralNumber = Number(t);

      if (t.includes('"')) { // check for the literal types
        if (param === t) parsedValue = param;
      } else if (!isNaN(possibleLiteralNumber)) { // we might have litreal numbers as type
        if (param === t) parsedValue = Number(param);
      } else { // union types can have regular types
        if (t.includes('{')) {
          parsedValue = JSON.parse(param);
        } else {
          parsedValue = queryParamToType({ name: t } as SettingKeyType<T>, param);
        }
      }
    };

    if (typeName.includes('|')) { // check for union types
      typeName.split(' | ').forEach((t) => checkTypes(t)); // parse each individual type
    } else {
      checkTypes(typeName);
    }

    return parsedValue;
  };

  if (param === null || param === undefined) return undefined;
  if (type.name === 'boolean') return param === 'true';
  if (type.name === 'number' || type.name === 'Array<number>') {
    const value = Number(param);
    return !isNaN(value) ? value : undefined;
  }
  if (type.name === 'string' || type.name === 'Array<string>')
  return param;
  if (type.is({})) return JSON.parse(param);
  if (type.name.includes('"')) {
    return validateLiteralType(type, param);
  }
  return undefined;
};

const queryToSettings = <T>(config: SettingsConfig<T>, query: string) => {
  const params = queryString.parse(query);

  return (Object.values(config.settings) as SettingsConfigProp<typeof config>[]).reduce<Settings>(
    (acc, setting) => {
      /*
       * Attempt to decode the query parameter and if anything
       * goes wrong, set it to the default value.
       */
      try {
        const paramValue = params[setting.storageKey];
        const baseType = setting.type;
        const isArray = baseType.is([]);

        if (paramValue !== null) {
          let queryValue: Primitive | Primitive[] | undefined = undefined;
          /*
           * Convert the string-based query params to primitives.
           * `undefined` values can happen if the query param values are invalid.
           *   string[] => Primitive[] | undefined
           *   string   => Primitive | undefined
           */
          if (Array.isArray(paramValue)) {
            queryValue = paramValue.reduce<Primitive[]>((acc, value) => {
              const parsedValue = queryParamToType<T>(baseType, value);

              if (parsedValue !== undefined) acc.push(parsedValue);

              return acc;
            }, []);
          } else {
            queryValue = queryParamToType<T>(baseType, paramValue);
          }

          if (queryValue !== undefined) {
            /*
             * When expecting an array, convert valid non-array values into an array.
             * Example - 'PULLING' => [ 'PULLING' ]
             */
            const normalizedValue = (() => {
              if (isArray && !Array.isArray(queryValue)) {
                return [queryValue];
              }
              return queryValue;
            })();

            acc[setting.storageKey] = normalizedValue;
          }
        }
      } catch (e) {
        handleError(e, { silent: true, type: ErrorType.Ui });
      }

      return acc;
    },
    {},
  );
};

const useSettings = <T>(config: SettingsConfig<T>): UseSettingsReturn<T> => {
  const loadableCurrentUser = useCurrentUser();
  const user = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });
  const { isLoading, querySettings, state, update } = useContext(UserSettings);
  const navigate = useNavigate();
  const location = useLocation();
  const pathname = location.pathname;
  const shouldSkipUpdates = useMemo(
    () => config.applicableRoutespace && !pathname.endsWith(config.applicableRoutespace),
    [config.applicableRoutespace, pathname],
  );
  const [shouldPush, setShouldPush] = useState(false); // internal state to manage navigation push property

  const settings: SettingsRecord<T> = useMemo(
    () =>
      ({
        ...(state.get(config.storagePath) ?? {}),
      } as SettingsRecord<T>),
    [config.storagePath, state],
  );
  const [returnedSettings, setReturnedSettings] = useState<SettingsRecord<T>>(settings);

  for (const key in config.settings) {
    const setting = config.settings[key];

    if (settings[setting.storageKey as keyof T] === undefined) {
      settings[setting.storageKey as keyof T] = setting.defaultValue;
    }
  }
  // parse navigation url to state
  useEffect(() => {
    if (!querySettings || shouldSkipUpdates) return;

    const settings = state.get(config.storagePath) ?? {};
    const settingsFromQuery = queryToSettings<T>(config, querySettings);

    if (isEqual(settingsFromQuery, settings)) return;

    Object.keys(settingsFromQuery).forEach((setting) => {
      settings[setting] = settingsFromQuery[setting];
    });

    update(config.storagePath, (stateSettings) => ({ ...stateSettings, ...settings }), true);
  }, [config, querySettings, state, update, shouldSkipUpdates]);

  /*
   * A setting is considered active if it is set to a value and the
   * value is not equivalent to a default value (if applicable).
   */
  const activeSettings = useCallback(
    (keys?: string[]): string[] => {
      return (Object.values(config.settings) as SettingsConfigProp<T>[]).reduce((acc, prop) => {
        if (!settings) return [];

        const key = prop.storageKey as keyof T;
        const includesKey = !keys || keys.includes(prop.storageKey);
        const isDefault = isEqual(settings[key], prop.defaultValue);

        if (includesKey && !isDefault) acc.push(prop.storageKey);

        return acc;
      }, [] as string[]);
    },
    [config.settings, settings],
  );

  const updateDB = useCallback(
    async (newSettings: Settings) => {
      if (!returnedSettings) return;

      const dbUpdates = Object.keys(newSettings).reduce<UserSettingUpdate[]>((acc, setting) => {
        const newSetting = newSettings[setting];
        const stateSetting = returnedSettings[setting as keyof T];

        if (user?.id && !isEqual(newSetting, stateSetting)) {
          acc.push({
            setting: {
              key: setting,
              storagePath: config.storagePath,
              value: JSON.stringify(newSettings[setting]),
            },
            storagePath: config.storagePath,
            userId: user.id,
          });
        }

        return acc;
      }, []);

      if (dbUpdates.length !== 0) {
        try {
          // Persist storage to backend.
          await Promise.allSettled(
            dbUpdates.map((update) => {
              updateUserSetting(update);
            }),
          );
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
    },
    [user?.id, config.storagePath, returnedSettings],
  );

  const resetSettings = useCallback(
    async (settingsArray?: string[]) => {
      if (!settings) return;

      const array = settingsArray ?? Object.keys(config.settings);
      const newSettings = { ...settings };

      array.forEach((setting) => {
        let defaultSetting: SettingsConfigProp<T[Extract<keyof T, string>]> | undefined = undefined;

        for (const key in config.settings) {
          const conf = config.settings[key];

          if (conf.storageKey === setting) {
            defaultSetting = conf;
            break;
          }
        }

        if (!defaultSetting) return;

        newSettings[setting as keyof T] = defaultSetting.defaultValue;
      });

      update(config.storagePath, () => newSettings);

      await updateDB(newSettings);

      navigate('', { replace: true });
    },
    [config, update, updateDB, navigate, settings],
  );

  const updateSettings = useCallback(
    (updates: Settings, shouldPushUpdate = false) => {
      if (shouldSkipUpdates) return;

      update(config.storagePath, (settings) => {
        if (!settings) return updates;
        return { ...settings, ...updates };
      });

      setShouldPush(shouldPushUpdate);
    },
    [config, update, shouldSkipUpdates],
  );

  useEffect(() => {
    // updates, if necessary, the returned settings
    if (!settings || isEqual(settings, returnedSettings)) return;

    setReturnedSettings(settings);

    updateDB(settings); // no need to await for DB changes as we optimistically update stuff based on the provided values
  }, [settings, returnedSettings, updateDB]);

  useEffect(() => {
    // updates the query settings
    if (shouldSkipUpdates) return;

    if (
      (Object.values(config.settings) as SettingsConfigProp<typeof config>[]).every(
        (setting) => !!setting.skipUrlEncoding,
      )
    ) {
      return;
    }

    const mappedSettings = settingsToQuery(config, returnedSettings);
    const url = `?${mappedSettings}`;

    if (mappedSettings && location.search !== url) {
      navigate(url, { replace: !shouldPush });
    }
  }, [shouldPush, location, returnedSettings, shouldSkipUpdates, navigate, config]);

  return {
    activeSettings,
    isLoading,
    resetSettings,
    settings: returnedSettings,
    updateSettings,
  };
};

export { SettingsProvider, useSettings };
