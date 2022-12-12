import * as t from 'io-ts';
import queryString from 'query-string';
import { useCallback, useContext, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';

import { updateUserSetting } from 'services/api';
import { UpdateUserSettingParams } from 'services/types';
import { Primitive } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { useAuth } from 'stores/auth';
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
  applicableRoutespace: string;
  settings: { [K in keyof T]: SettingsConfigProp<T[K]> };
  storagePath: string;
}

interface UserSettingUpdate extends UpdateUserSettingParams {
  userId: number;
}

export type UpdateSettings = (updates: Settings, shouldPush?: boolean) => Promise<void>;
export type ResetSettings = (settings?: string[]) => void;
type SettingsRecord<T> = { [K in keyof T]: T[K] };

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
  type: t.Type<SettingsConfig<T>, SettingsConfig<T>, unknown>,
  param: string | null,
): Primitive | undefined => {
  if (param === null || param === undefined) return undefined;
  if (type.is(false)) return param === 'true';
  if (type.is(0)) {
    const value = Number(param);
    return !isNaN(value) ? value : undefined;
  }
  if (type.is({})) return JSON.parse(param);
  if (type.is('')) return param;
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
  const loadableAuth = useAuth();
  const user = Loadable.match(loadableAuth.auth, {
    Loaded: (auth) => auth.user,
    NotLoaded: () => undefined,
  });
  const { isLoading, querySettings, state, update } = useContext(UserSettings);
  const navigate = useNavigate();
  const pathname = window.location.pathname;
  const shouldSkipUpdates = useMemo(
    () =>
      config.applicableRoutespace.includes('/') && !pathname.endsWith(config.applicableRoutespace),
    [config.applicableRoutespace, pathname],
  );

  // parse navigation url to state
  useEffect(() => {
    if (!querySettings || shouldSkipUpdates) return;

    const settings = queryToSettings<T>(config, querySettings);
    const stateSettings = state.get(config.applicableRoutespace) ?? {};

    if (isEqual(settings, stateSettings)) return;

    Object.keys(settings).forEach((setting) => {
      stateSettings[setting] = settings[setting];
    });

    update(config.applicableRoutespace, stateSettings, true);
  }, [config, querySettings, state, update, shouldSkipUpdates]);

  const settings: SettingsRecord<T> = useMemo(
    () =>
      ({
        ...(state.get(config.applicableRoutespace) ?? {}),
      } as SettingsRecord<T>),
    [config, state],
  );

  for (const key in config.settings) {
    const setting = config.settings[key];

    if (settings[setting.storageKey as keyof T] === undefined) {
      settings[setting.storageKey as keyof T] = setting.defaultValue;
    }
  }

  useEffect(() => {
    if (shouldSkipUpdates) return;

    const mappedSettings = settingsToQuery(config, settings as Settings);
    const url = `?${mappedSettings}`;

    if (mappedSettings && url !== window.location.search) navigate(url, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
      if (!settings) return;

      const dbUpdates = Object.keys(newSettings).reduce<UserSettingUpdate[]>((acc, setting) => {
        const newSetting = newSettings[setting];
        const stateSetting = settings[setting as keyof T];

        if (user?.id && !isEqual(newSetting, stateSetting)) {
          acc.push({
            setting: {
              key: setting,
              storagePath: config.applicableRoutespace,
              value: JSON.stringify(newSettings[setting]),
            },
            storagePath: config.applicableRoutespace,
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
    [user?.id, config.applicableRoutespace, settings],
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

      update(config.applicableRoutespace, newSettings);

      await updateDB(newSettings);

      navigate('', { replace: true });
    },
    [config, update, updateDB, navigate, settings],
  );

  const updateSettings = useCallback(
    async (updates: Settings, shouldPush = false) => {
      if (!settings || shouldSkipUpdates) return;

      const newSettings = { ...settings, ...updates };

      if (isEqual(newSettings, settings)) return;

      update(config.applicableRoutespace, newSettings);

      await updateDB(newSettings);

      const mappedSettings = settingsToQuery(config, newSettings);
      const url = `?${mappedSettings}`;

      shouldPush ? navigate(url) : navigate(url, { replace: true });
    },
    [config, settings, navigate, update, updateDB, shouldSkipUpdates],
  );

  return {
    activeSettings,
    isLoading,
    resetSettings,
    settings,
    updateSettings,
  };
};

export { SettingsProvider, useSettings };
