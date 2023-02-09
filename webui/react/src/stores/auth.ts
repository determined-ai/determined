import { WritableObservable } from 'micro-observables';

import { globalStorage } from 'globalStorage';
import { Auth } from 'types';
import { getCookie, setCookie } from 'utils/browser';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export const AUTH_COOKIE_KEY = 'auth';

const clearAuthCookie = (): void => {
  document.cookie = `${AUTH_COOKIE_KEY}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
};

/**
 * set the auth cookie if it's not already set.
 *
 * @param token auth token
 */
const ensureAuthCookieSet = (token: string): void => {
  if (!getCookie(AUTH_COOKIE_KEY)) setCookie(AUTH_COOKIE_KEY, token);
};

const internalAuth = new WritableObservable<Loadable<Auth>>(NotLoaded);
const internalAuthChecked = new WritableObservable(false);

export const auth = internalAuth.readOnly();
export const authChecked = internalAuth.readOnly();

export const reset = (): void => {
  clearAuthCookie();
  globalStorage.removeAuthToken();
  WritableObservable.batch(() => {
    internalAuth.set(NotLoaded);
    internalAuthChecked.set(false);
  });
};
export const setAuth = (newAuth: Auth): void => {
  if (newAuth.token) {
    ensureAuthCookieSet(newAuth.token);
    globalStorage.authToken = newAuth.token;
  }
  internalAuth.set(Loaded(newAuth));
};
export const setAuthChecked = (): void => internalAuthChecked.set(true);
export const selectIsAuthenticated = auth.select((a) =>
  Loadable.match(a, {
    Loaded: (au) => au.isAuthenticated,
    NotLoaded: () => false,
  }),
);
