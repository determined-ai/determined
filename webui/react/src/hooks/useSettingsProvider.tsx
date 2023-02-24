import { Map } from 'immutable';
import { observable, useObservable, WritableObservable } from 'micro-observables';
import React, { createContext, useEffect, useRef, useState } from 'react';

import { getUserSetting } from 'services/api';
import Spinner from 'shared/components/Spinner';
import { ErrorType } from 'shared/utils/error';
import { authChecked } from 'stores/auth';
import { useCurrentUser } from 'stores/users';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

/*
 * UserSettingsState contains all the settings for a user
 * across the application. Each key identifies a unique part
 * of the interface to store settings for.
 */
type UserSettingsState = Map<string, Settings>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type Settings = { [key: string]: any }; //TODO: find a way to use a better type here

type UserSettingsContext = {
  isLoading: WritableObservable<boolean>;
  querySettings: string;
  state: WritableObservable<UserSettingsState>;
  // update: (key: string, value: Settings, clearQuerySettings?: boolean) => void;
};

export const UserSettings = createContext<UserSettingsContext>({
  isLoading: observable(false),
  querySettings: '',
  state: observable(Map<string, Settings>()),
  // update: () => undefined,
});

export const SettingsProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const loadableCurrentUser = useCurrentUser();
  const user = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });
  const checked = useObservable(authChecked);
  const [canceler] = useState(new AbortController());
  const [isLoading] = useState(() => observable(true));
  const querySettings = useRef('');
  const [settingsState] = useState(() => observable(Map<string, Settings>()));

  useEffect(() => {
    if (!user?.id || !checked) return;

    try {
      getUserSetting({}, { signal: canceler.signal }).then((response) => {
        isLoading.set(false);
        settingsState.update((currentState) => {
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
      isLoading.set(false);
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

  // const update = (key: string, value: Settings, clearQuerySettings = false) => {
  //   settingsState.update((currentState) => currentState.set(key, value));

  //   if (clearQuerySettings) querySettings.current = '';
  // };

  // useEffect(() => {
  //   return settingsState.subscribe(async (cur, prev) => {
  //     // check the difference 
  //     const diff = Map<string, Settings>()
  //     cur.forEach()

  //     await updateDB(cur);

  //     if (
  //       (Object.values(config.settings) as SettingsConfigProp<typeof config>[]).every(
  //         (setting) => !!setting.skipUrlEncoding,
  //       )
  //     ) {
  //       return;
  //     }

  //     const mappedSettings = settingsToQuery(config, newSettings);
  //     const url = `?${mappedSettings}`;

  //     shouldPush ? navigate(url) : navigate(url, { replace: true });
  //   })
    
  // }, )

  return (
    <Spinner spinning={useObservable(isLoading) && !(checked && !user)} tip="Loading Page">
      <UserSettings.Provider
        value={{
          isLoading: isLoading,
          querySettings: querySettings.current,
          state: settingsState,
        }}>
        {children}
      </UserSettings.Provider>
    </Spinner>
  );
};
function updateDB(newSettings: any) {
  throw new Error('Function not implemented.');
}

function settingsToQuery(config: any, newSettings: any) {
  throw new Error('Function not implemented.');
}

function navigate(url: string) {
  throw new Error('Function not implemented.');
}

