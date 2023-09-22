import { observable, WritableObservable } from 'micro-observables';

import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import { globalStorage } from 'globalStorage';
import { Auth } from 'types';
import { getCookie, setCookie } from 'utils/browser';

export const AUTH_COOKIE_KEY = 'auth';

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

interface AuthState {
  auth: Loadable<Auth>;
  isChecked: boolean;
}

const defaultState: AuthState = {
  auth: NotLoaded,
  isChecked: false,
};

class AuthStore {
  #state: WritableObservable<AuthState> = observable(defaultState);

  public readonly auth = this.#state.select((state) => state.auth);
  public readonly isChecked = this.#state.select((state) => state.isChecked);
  public readonly isAuthenticated = this.auth.select((loadableAuth) => {
    return Loadable.match(loadableAuth, {
      Loaded: (a) => a.isAuthenticated,
      NotLoaded: () => false,
    });
  });

  public setAuth(newAuth: Auth) {
    if (newAuth.token) {
      ensureAuthCookieSet(newAuth.token);
      globalStorage.authToken = newAuth.token;
    }
    this.#state.update((s) => ({ ...s, auth: Loaded(newAuth) }));
  }

  public setAuthChecked() {
    this.#state.update((s) => ({ ...s, isChecked: true }));
  }

  public reset() {
    clearAuthCookie();
    globalStorage.removeAuthToken();
    this.#state.set(defaultState);
  }
}

export default new AuthStore();
