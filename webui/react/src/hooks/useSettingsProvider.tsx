import React, { createContext, useEffect, useRef, useState } from 'react';

import { useStore } from 'contexts/Store';
import { getUserSetting } from 'services/api';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

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
  update: (
    key: string,
    value: Settings,
    clearQuerySettings?: boolean,
    callback?: () => void,
  ) => void;
};

export const UserSettings = createContext<UserSettingsContext>({
  isLoading: false,
  querySettings: '',
  state: new Map(),
  update: () => undefined,
});

// TODO: check navigation and settings and changing map to state

export const SettingsProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const {
    auth: { user },
  } = useStore();
  const [canceler] = useState(new AbortController());
  const [isLoading, setIsLoading] = useState(false);
  const querySettings = useRef('');
  const settingsState = useRef(new Map<string, Settings>());

  useEffect(() => {
    if (!user?.id) return;

    setIsLoading(true);

    try {
      getUserSetting({}, { signal: canceler.signal }).then((response) => {
        setIsLoading(false);

        response.settings.forEach((setting) => {
          const value = setting.value ? JSON.parse(setting.value) : undefined;
          let key = setting.storagePath || setting.key; // falls back to the setting key due to storagePath being optional.

          if (key.includes('u:2/')) key = key.replace(/u:2\//g, '');

          const entry = settingsState.current.get(key);

          if (!entry) {
            settingsState.current.set(key, { [setting.key]: value });
          } else {
            settingsState.current.set(key, Object.assign(entry, { [setting.key]: value }));
          }
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
  }, [canceler, user?.id]);

  useEffect(() => {
    const url = window.location.search;

    querySettings.current = url;
  }, []);

  const update = (key: string, value: Settings, clearQuerySettings = false, callback) => {
    settingsState.current.set(key, value);

    if (clearQuerySettings) querySettings.current = '';

    callback?.();
  };

  return (
    <UserSettings.Provider
      value={{
        isLoading: isLoading,
        querySettings: querySettings.current,
        state: settingsState.current,
        update,
      }}>
      {children}
    </UserSettings.Provider>
  );
};
