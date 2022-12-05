import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { globalStorage } from 'globalStorage';
import { getUsers } from 'services/api';
import { V1GetUsersRequestSortBy } from 'services/api-ts-sdk';
import { isEqual } from 'shared/utils/data';
import { Auth, DetailedUser } from 'types';
import { getCookie, setCookie } from 'utils/browser';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export type CurrentUser = Auth & { checked: boolean };
type UsersContext = {
  auth: CurrentUser;
  updateAuth: (fn: (users: CurrentUser) => CurrentUser) => void;
  updateUsers: (fn: (users: Loadable<DetailedUser[]>) => Loadable<DetailedUser[]>) => void;
  users: Loadable<DetailedUser[]>;
};

type FetchUsersConfig = {
  limit: number;
  offset: number;
  orderBy: 'ORDER_BY_DESC' | 'ORDER_BY_ASC';
  sortBy: V1GetUsersRequestSortBy;
};

type UseUsersReturn = {
  auth: CurrentUser;
  updateCurrentUser: (user: DetailedUser) => void;
  users: Loadable<DetailedUser[]>;
};

type UseAuthReturn = {
  auth: CurrentUser;
  resetAuth: () => void;
  resetAuthCheck: () => void;
  setAuth: (auth: Auth) => void;
  setAuthCheck: () => void;
};

export const AUTH_COOKIE_KEY = 'auth';

const UsersContext = createContext<UsersContext | null>(null);

export const UsersProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [users, setUsers] = useState<Loadable<DetailedUser[]>>(NotLoaded);
  const [auth, setAuth] = useState<CurrentUser>({ checked: false, isAuthenticated: false });

  return (
    <UsersContext.Provider
      value={{
        auth,
        updateAuth: setAuth,
        updateUsers: setUsers,
        users,
      }}>
      {children}
    </UsersContext.Provider>
  );
};

export const useFetchUsers = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchUsers outside of Users Context');
  }

  const { updateUsers } = context;

  return useCallback(
    async (cfg?: FetchUsersConfig): Promise<void> => {
      try {
        const config = cfg ?? {};
        const response = await getUsers(config, { signal: canceler.signal });

        updateUsers(() => Loaded(response.users));
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, updateUsers],
  );
};

export const useEnsureUsersFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchUsers outside of Users Context');
  }

  const { updateUsers, users } = context;

  return useCallback(
    async (cfg?: FetchUsersConfig): Promise<void> => {
      if (users !== NotLoaded) return;

      try {
        const config = cfg ?? {};
        const response = await getUsers(config, { signal: canceler.signal });

        updateUsers(() => Loaded(response.users));
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, updateUsers, users],
  );
};

export const useUsers = (): UseUsersReturn => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useUsers outside of Users Context');
  }
  const { users, auth, updateAuth: setCurrentUser } = context;

  const updateCurrentUser = useCallback(
    (user: DetailedUser) => {
      const usersArray = Loadable.getOrElse([], users);

      setCurrentUser((prevState) => {
        if (isEqual(prevState, user)) return prevState;

        const userIdx = usersArray.findIndex((user) => user.id === user.id);

        if (userIdx > -1) usersArray[userIdx] = { ...usersArray[userIdx], ...user };

        return { ...prevState, user: user };
      });
    },
    [setCurrentUser, users],
  );

  return {
    auth,
    updateCurrentUser,
    users,
  };
};

const clearAuthCookie = (): void => {
  document.cookie = `${AUTH_COOKIE_KEY}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
};

/**
 * set the auth cookie if it's not already set.
 * @param token auth token
 */
const ensureAuthCookieSet = (token: string): void => {
  if (!getCookie(AUTH_COOKIE_KEY)) setCookie(AUTH_COOKIE_KEY, token);
};

export const useAuth = (): UseAuthReturn => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useAuth outside of Users Context');
  }
  const { auth, updateAuth } = context;

  const resetAuth = useCallback(() => {
    clearAuthCookie();
    globalStorage.removeAuthToken();

    updateAuth((prevState) => ({ ...prevState, isAuthenticated: false }));
  }, [updateAuth]);

  const resetAuthCheck = useCallback(() => {
    updateAuth((prevState) => {
      if (!prevState.checked) return prevState;
      return { ...prevState, checked: false };
    });
  }, [updateAuth]);

  const setAuth = useCallback(
    (auth: Auth) => {
      if (auth.token) {
        /**
         * project Samuel provisioned auth doesn't set a cookie
         * like our other auth methods do.
         *
         */
        ensureAuthCookieSet(auth.token);
        globalStorage.authToken = auth.token;
      }

      updateAuth(() => ({ ...auth, checked: true }));
    },
    [updateAuth],
  );

  const setAuthCheck = useCallback(() => {
    updateAuth((prevState) => {
      if (prevState.checked) return prevState;

      return { ...prevState, checked: true };
    });
  }, [updateAuth]);

  return {
    auth,
    resetAuth,
    resetAuthCheck,
    setAuth,
    setAuthCheck,
  };
};
