import * as t from 'io-ts';
import { useObservable } from 'micro-observables';
import queryString from 'query-string';
import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { updateUserSetting } from 'services/api';
import { UpdateUserSettingParams } from 'services/types';
import { Primitive } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { useCurrentUser } from 'stores/users';
import { DetailedUser } from 'types';
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
  settings: { [K in keyof T]: SettingsConfigProp<T[K]> };
  storagePath: string;
}

interface UserSettingUpdate extends UpdateUserSettingParams {
  userId: number;
}

export type UpdateSettings = (updates: Settings, shouldPush?: boolean) => void;
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
  const loadableCurrentUser = useCurrentUser();
  const user = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });
  const {
    isLoading: isLoadingOb,
    querySettings,
    state: stateOb,
    clearQuerySettings,
  } = useContext(UserSettings);
  const isLoading = useObservable(isLoadingOb);
  const [derivedOb] = useState(stateOb.select((s) => s.get(config.storagePath)));
  const state = useObservable(derivedOb);
  const navigate = useNavigate();

  // parse navigation url to state
  useEffect(() => {
    if (!querySettings) return;

    const settings = queryToSettings<T>(config, querySettings);
    const stateSettings = state ?? {};

    if (isEqual(settings, stateSettings)) return;

    Object.keys(settings).forEach((setting) => {
      stateSettings[setting] = settings[setting];
    });
    stateOb.update((s) => s.set(config.storagePath, stateSettings));

    clearQuerySettings();
  }, [config, querySettings, state, clearQuerySettings, stateOb]);

  const settings: SettingsRecord<T> = useMemo(
    () =>
      ({
        ...(state ?? {}),
      } as SettingsRecord<T>),
    [state],
  );

  for (const key in config.settings) {
    const setting = config.settings[key];

    if (settings[setting.storageKey as keyof T] === undefined) {
      settings[setting.storageKey as keyof T] = setting.defaultValue;
    }
  }

  useEffect(() => {
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
    async (newSettings: Settings, user: DetailedUser) => {
      if (!settings) return;

      const dbUpdates = Object.keys(newSettings).reduce<UserSettingUpdate[]>((acc, setting) => {
        const newSetting = newSettings[setting];
        const stateSetting = settings[setting as keyof T];
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
    [config.storagePath, settings],
  );

  const resetSettings = useCallback(
    async (settingsArray?: string[]) => {
      if (!settings || !user) return;

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
      stateOb.update((s) => s.set(config.storagePath, newSettings));

      await updateDB(newSettings, user);

      navigate('', { replace: true });
    },
    [config, updateDB, navigate, settings, user, stateOb],
  );

  const updateSettings = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    (updates: Settings, shouldPush = false) => {
      stateOb.update((s) =>
        s.set(
          config.storagePath,
          s.get(config.storagePath) === updates
            ? s.get(config.storagePath) ?? {}
            : { ...s.get(config.storagePath), ...updates },
        ),
      );
    },
    [config, stateOb],
  );

  useEffect(() => {
    return derivedOb.subscribe(async (cur, prev) => {
      if (!cur || !user || cur === prev) return;

      await updateDB(cur, user);

      if (
        (Object.values(config.settings) as SettingsConfigProp<typeof config>[]).every(
          (setting) => !!setting.skipUrlEncoding,
        )
      ) {
        return;
      }
      const mappedSettings = settingsToQuery(config, cur);
      const url = `?${mappedSettings}`;
      navigate(url, { replace: true });
    });
  }, [derivedOb, user, navigate, config, updateDB]);

  return {
    activeSettings,
    isLoading,
    resetSettings,
    settings,
    updateSettings,
  };
};

export { SettingsProvider, useSettings };
