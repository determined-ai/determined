import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { globalStorage } from 'globalStorage';
import { Auth } from 'types';
import { getCookie, setCookie } from 'utils/browser';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export type CurrentUser = Auth;

type AuthContext = {
  auth: Loadable<CurrentUser>;
  authChecked: boolean;
  updateAuth: (fn: (users: Loadable<CurrentUser>) => Loadable<CurrentUser>) => void;
  updateAuthChecked: (fn: (aChecked: boolean) => boolean) => void;
};

type UseAuthReturn = {
  auth: Loadable<CurrentUser>;
  authChecked: boolean;
  resetAuth: () => void;
  setAuth: (auth: Auth) => void;
  setAuthCheck: () => void;
};

export const AUTH_COOKIE_KEY = 'auth';

const AuthContext = createContext<AuthContext | null>(null);

export const AuthProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [auth, setAuth] = useState<Loadable<CurrentUser>>(NotLoaded);
  const [authChecked, setAuthChecked] = useState(false);

  return (
    <AuthContext.Provider
      value={{
        auth,
        authChecked,
        updateAuth: setAuth,
        updateAuthChecked: setAuthChecked,
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
  const { auth, authChecked, updateAuth, updateAuthChecked } = context;

  const resetAuth = useCallback(() => {
    clearAuthCookie();
    globalStorage.removeAuthToken();

    updateAuth(() => NotLoaded);
    updateAuthChecked(() => false);
  }, [updateAuth, updateAuthChecked]);

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
    updateAuthChecked(() => {
      return true;
    });
  }, [updateAuthChecked]);

  return {
    auth,
    authChecked,
    resetAuth,
    setAuth,
    setAuthCheck,
  };
};
