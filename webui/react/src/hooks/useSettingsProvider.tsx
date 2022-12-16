import { Map } from 'immutable';
import React, { createContext, useEffect, useRef, useState } from 'react';

import { getUserSetting } from 'services/api';
import Spinner from 'shared/components/Spinner';
import { ErrorType } from 'shared/utils/error';
import { useAuth } from 'stores/auth';
import { useCurrentUsers } from 'stores/users';
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
  isLoading: boolean;
  querySettings: string;
  state: UserSettingsState;
  update: (key: string, value: Settings, clearQuerySettings?: boolean) => void;
};

export const UserSettings = createContext<UserSettingsContext>({
  isLoading: false,
  querySettings: '',
  state: Map<string, Settings>(),
  update: () => undefined,
});

export const SettingsProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const loadableAuth = useAuth();
  const user = Loadable.match(useCurrentUsers().currentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });
  const checked = loadableAuth.authChecked;
  const [canceler] = useState(new AbortController());
  const [isLoading, setIsLoading] = useState(true);
  const querySettings = useRef('');
  const [settingsState, setSettingsState] = useState(() => Map<string, Settings>());

  useEffect(() => {
    if (!user?.id) return;

    try {
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
  }, [canceler, user?.id, settingsState]);

  useEffect(() => {
    const url = window.location.search.substr(/^\?/.test(location.search) ? 1 : 0);

    querySettings.current = url;
  }, []);

  const update = (key: string, value: Settings, clearQuerySettings = false) => {
    setSettingsState((currentState) => currentState.set(key, value));

    if (clearQuerySettings) querySettings.current = '';
  };

  if (isLoading && !(checked && !user)) return <Spinner spinning />;

  return (
    <UserSettings.Provider
      value={{
        isLoading: isLoading,
        querySettings: querySettings.current,
        state: settingsState,
        update,
      }}>
      {children}
    </UserSettings.Provider>
  );
};
