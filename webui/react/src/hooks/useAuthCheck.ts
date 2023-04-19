import { Observable, useObservable } from 'micro-observables';
import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';

import { globalStorage } from 'globalStorage';
import { routeAll } from 'routes/utils';
import { updateDetApi } from 'services/apiConfig';
import authStore, { AUTH_COOKIE_KEY } from 'stores/auth';
import determinedStore from 'stores/determinedInfo';
import { getCookie } from 'utils/browser';

const useAuthCheck = (): (() => void) => {
  const info = useObservable(determinedStore.info);
  const [searchParams] = useSearchParams();

  const updateBearerToken = useCallback((token: string) => {
    globalStorage.authToken = token;
    updateDetApi({ apiKey: `Bearer ${token}` });
  }, []);

  const redirectToExternalSignin = useCallback(() => {
    const redirect = encodeURIComponent(window.location.href);
    const authUrl = `${info.externalLoginUri}?redirect=${redirect}`;
    routeAll(authUrl);
  }, [info.externalLoginUri]);

  const checkAuth = useCallback((): void => {
    authStore.setAuthChecked(); // TODO if info.externalLoginUri, just do this - and redirectToExternalSignin() any time we get a 403
  }, [info.externalLoginUri, searchParams, redirectToExternalSignin, updateBearerToken]);

  return checkAuth;
};

export default useAuthCheck;
