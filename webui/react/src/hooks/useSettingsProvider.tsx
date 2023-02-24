import { Map } from 'immutable';
import { observable, useObservable, WritableObservable } from 'micro-observables';
import React, { createContext, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { updateUserSetting } from 'services/api';
import { getUserSetting } from 'services/api';
import { UpdateUserSettingParams } from 'services/types';
import Spinner from 'shared/components/Spinner';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { authChecked } from 'stores/auth';
import { useCurrentUser } from 'stores/users';
import { UserAssignment } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import { queryToSettings, SettingsConfig, SettingsConfigProp, settingsToQuery, UseSettingsReturn } from './useSettings';

/*
 * UserSettingsState contains all the settings for a user
 * across the application. Each key identifies a unique part
 * of the interface to store settings for.
 */
type UserSettingsState = Map<string, Settings>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type Settings = { [key: string]: any }; //TODO: find a way to use a better type here

type UserSettingsContext = {
  isLoading: boolean;
  querySettings: string;
  state: UserSettingsState;
  update: (key: string, value: Settings, clearQuerySettings?: boolean) => void;
};
type SettingsRecord<T> = { [K in keyof T]: T[K] };
export const UserSettings = createContext<UserSettingsContext>({
  isLoading: false,
  querySettings: '',
  state: Map<string, Settings>(),
  update: () => undefined,
});

interface UserSettingUpdate extends UpdateUserSettingParams {
  userId: number;
}

export class UserSettingsService {
  static #state: WritableObservable<Loadable<Map<string, Settings>>> = observable(NotLoaded);
  static #isLoading: WritableObservable<boolean> = observable(true)
  static #querySettings: WritableObservable<string> = observable((''));

  static async fetchUserSettings(canceler: AbortController): Promise<void> {
    // const user = Loadable.match(useCurrentUser(), {
    //   Loaded: (cUser) => cUser,
    //   NotLoaded: () => undefined,
    // });
    // const checked = useObservable(authChecked);

      const { settings } = await getUserSetting({}, { signal: canceler.signal });
      this.#isLoading.set(false);

      const preState = Loadable.getOrElse(Map<string, Settings>(), this.#state.get());
      const newState = preState.withMutations((state) => {
        settings.forEach((setting) => {
          // console.log(setting.storagePath)
          const value = setting.value ? JSON.parse(setting.value) : undefined;
          let key = setting.storagePath || setting.key; // falls back to the setting key due to storagePath being optional.

          if (key.includes('u:2/')) key = key.replace(/u:2\//g, '');

          const entry = this.#state.select((s) => Loadable.quickMatch(s, undefined, (m) => m.get(key)));

          if (!entry) {
            // this.#state.update(s => ({...s, key: { [setting.key]: value }}))
            state.set(key, { [setting.key]: value });
          } else {
            // this.#state.update(s => ({...s, key: Object.assign(entry, { [setting.key]: value })}))
            state.set(key, Object.assign(entry, { [setting.key]: value }));
          }
          console.log(state.size);
        });
      });
      console.log('set state', newState);
      this.#state.set(Loaded(newState));
  }

  static useSettings <T>(config: SettingsConfig<T>): UseSettingsReturn<T> {
    const navigate = useNavigate();

    const user = Loadable.match(useCurrentUser(), {
      Loaded: (cUser) => cUser,
      NotLoaded: () => undefined,
    });
    const settings: SettingsRecord<T> = useMemo(
      () => {
          // console.log(Loadable.getOrElse(Map<string, Settings>(), this.#state.get()).get(config.storagePath)?.get())
          return ({
          ...(Loadable.getOrElse(Map<string, Settings>(), this.#state.get()).get(config.storagePath) ?? {}),
        } as SettingsRecord<T>);
},
      [config],
    );
     // parse navigation url to state
  useEffect(() => {
    console.log(config);
    if (!this.#querySettings) return;

    const settings = queryToSettings<T>(config, this.#querySettings.get());
    const stateSettings = this.#state.select((s) => Loadable.quickMatch(s, undefined, (m) => m.get(config.storagePath))).get() ?? {};

    if (isEqual(settings, stateSettings)) return;

    Object.keys(settings).forEach((setting) => {
      stateSettings[setting] = settings[setting];
    });
    const state = Loadable.getOrElse(Map<string, Settings>(), this.#state.get()).withMutations((s) => (s.set(config.storagePath, stateSettings)));
    this.#state.set(Loaded(state));
    this.#querySettings.set('');
  }, [config]);

  // for (const key in config.settings) {
  //   const setting = config.settings[key];

  //   if (settings[setting.storageKey as keyof T] === undefined) {
  //     settings[setting.storageKey as keyof T] = setting.defaultValue;
  //   }
  // }

  useEffect(() => {
    const mappedSettings = settingsToQuery(config, settings as Settings);
    const url = `?${mappedSettings}`;

    if (mappedSettings && url !== window.location.search) navigate(url, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
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
      [user?.id, config.storagePath, settings],
    );
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
      const state = Loadable.getOrElse(Map<string, Settings>(), this.#state.get()).withMutations((s) => (s.set(config.storagePath, newSettings)));
      this.#state.set(Loaded(state));

      await updateDB(newSettings);

      navigate('', { replace: true });
    },
    [config, updateDB, navigate, settings],
  );

  const updateSettings = useCallback(
    async (updates: Settings, shouldPush = false) => {
      if (!settings) return;
      const newSettings = { ...settings, ...updates };

      if (isEqual(newSettings, settings)) return;
      const state = Loadable.getOrElse(Map<string, Settings>(), this.#state.get()).withMutations((s) => (s.set(config.storagePath, newSettings)));
      this.#state.set(Loaded(state));

      await updateDB(newSettings);

      if (
        (Object.values(config.settings) as SettingsConfigProp<typeof config>[]).every(
          (setting) => !!setting.skipUrlEncoding,
        )
      ) {
        return;
      }

      const mappedSettings = settingsToQuery(config, newSettings);
      const url = `?${mappedSettings}`;

      shouldPush ? navigate(url) : navigate(url, { replace: true });
    },
    [config, settings, navigate, updateDB],
  );

    return {
      activeSettings,
      isLoading: this.#isLoading.get(),
      resetSettings,
      settings,
      updateSettings,
    };
  }
}

export const SettingsProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const loadableCurrentUser = useCurrentUser();
  const user = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });
  const checked = useObservable(authChecked);
  const [canceler] = useState(new AbortController());
  const [isLoading, setIsLoading] = useState(true);
  const querySettings = useRef('');
  const [settingsState, setSettingsState] = useState(() => Map<string, Settings>());

  useEffect(() => {
    console.log({ checked, user });
    if (!user?.id || !checked) return;
    console.log('api');
    UserSettingsService.fetchUserSettings(canceler);
    return () => canceler.abort();
  }, [user?.id, checked, canceler]);

  useEffect(() => {

    if (!user?.id || !checked) return;

    // UserSettingsService.fetchUserSettings(canceler)()
    try {
      console.log(302, 'api');
      getUserSetting({}, { signal: canceler.signal }).then((response) => {
        setIsLoading(false);
        setSettingsState((currentState) => {
          return currentState.withMutations((state) => {
            response.settings.forEach((setting) => {
              const value = setting.value ? JSON.parse(setting.value) : undefined;
              let key = setting.storagePath || setting.key; // falls back to the setting key due to storagePath being optional.

              if (key.includes('u:2/')) key = key.replace(/u:2\//g, '');

              const entry = state.get(key);

              if (!entry) {
                state.set(key, { [setting.key]: value });
              } else {
                state.set(key, Object.assign(entry, { [setting.key]: value }));
              }
            });
          });
        });
      });
    } catch (error) {
      setIsLoading(false);
      handleError(error, {
        isUserTriggered: false,
        publicMessage: 'Unable to fetch user settings.',
        type: ErrorType.Api,
      });
    }

    return () => canceler.abort();
  }, [canceler, user?.id, checked, settingsState]);

  useEffect(() => {
    const url = window.location.search.substring(/^\?/.test(location.search) ? 1 : 0);

    querySettings.current = url;
  }, []);

  const update = (key: string, value: Settings, clearQuerySettings = false) => {
    console.log('update');
    setSettingsState((currentState) => currentState.set(key, value));

    if (clearQuerySettings) querySettings.current = '';
  };

  return (
    <Spinner spinning={isLoading && !(checked && !user)} tip="Loading Page">
      <UserSettings.Provider
        value={{
          isLoading: isLoading,
          querySettings: querySettings.current,
          state: settingsState,
          update,
        }}>
        {children}
      </UserSettings.Provider>
    </Spinner>
  );
};
