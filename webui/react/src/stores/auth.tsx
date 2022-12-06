import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { globalStorage } from 'globalStorage';
import { isEqual } from 'shared/utils/data';
import { Auth, DetailedUser } from 'types';
import { getCookie, setCookie } from 'utils/browser';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export type CurrentUser = Auth & { checked: boolean };

type AuthContext = {
  auth: Loadable<CurrentUser>;
  updateAuth: (fn: (users: Loadable<CurrentUser>) => Loadable<CurrentUser>) => void;
};

type UseAuthReturn = {
  auth: Loadable<CurrentUser>;
  resetAuth: () => void;
  setAuth: (auth: Auth) => void;
  setAuthCheck: () => void;
  updateCurrentUser: (user: DetailedUser, users: DetailedUser[]) => void;
};

export const AUTH_COOKIE_KEY = 'auth';

const AuthContext = createContext<AuthContext | null>(null);

export const AuthProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [auth, setAuth] = useState<Loadable<CurrentUser>>(NotLoaded);

  return (
    <AuthContext.Provider
      value={{
        auth,
        updateAuth: setAuth,
      }}>
      {children}
    </AuthContext.Provider>
  );
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
  const context = useContext(AuthContext);

  if (context === null) {
    throw new Error('Attempted to use useAuth outside of Auth Context');
  }
  const { auth, updateAuth } = context;

  const updateCurrentUser = useCallback(
    (user: DetailedUser, users: DetailedUser[]) => {
      updateAuth((prevState) => {
        const auth = Loadable.getOrElse({ checked: false, isAuthenticated: false }, prevState);
        if (isEqual(auth, user)) return prevState;

        const userIdx = users.findIndex((user) => user.id === user.id);

        if (userIdx > -1) users[userIdx] = { ...users[userIdx], ...user };

        return Loaded({ ...auth, user: user });
      });
    },
    [updateAuth],
  );

  const resetAuth = useCallback(() => {
    clearAuthCookie();
    globalStorage.removeAuthToken();

    updateAuth(() => NotLoaded);
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

      updateAuth(() => Loaded({ ...auth, checked: true }));
    },
    [updateAuth],
  );

  const setAuthCheck = useCallback(() => {
    updateAuth((prevState) => {
      const auth = Loadable.getOrElse({ checked: false, isAuthenticated: false }, prevState);
      if (auth.checked) return prevState;

      return Loaded({ ...auth, checked: true });
    });
  }, [updateAuth]);

  return {
    auth,
    resetAuth,
    setAuth,
    setAuthCheck,
    updateCurrentUser,
  };
};
