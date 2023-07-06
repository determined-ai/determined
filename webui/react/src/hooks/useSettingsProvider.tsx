import { Map } from 'immutable';
import React, { createContext, useEffect, useRef } from 'react';

import Spinner from 'components/Spinner';
import authStore from 'stores/auth';
import userStore from 'stores/users';
import userSettings from 'stores/userSettings';
import { Loadable, NotLoaded } from 'utils/loadable';
import { observable, useObservable, WritableObservable } from 'utils/observable';

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
  querySettings: URLSearchParams;
  state: WritableObservable<Loadable<UserSettingsState>>;
};

export const UserSettings = createContext<UserSettingsContext>({
  isLoading: true,
  querySettings: new URLSearchParams(''),
  state: observable(NotLoaded),
});

export const SettingsProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const isAuthChecked = useObservable(authStore.isChecked);
  const querySettings = useRef(new URLSearchParams(''));
  const isLoading = Loadable.isLoading(useObservable(userSettings._forUserSettingsOnly()));

  useEffect(() => {
    querySettings.current = new URLSearchParams(window.location.search);
  }, []);

  return (
    <Spinner spinning={isLoading && !(isAuthChecked && !currentUser)} tip="Loading Page">
      <UserSettings.Provider
        value={{
          isLoading,
          querySettings: querySettings.current,
          state: userSettings._forUserSettingsOnly(),
        }}>
        {children}
      </UserSettings.Provider>
    </Spinner>
  );
};
