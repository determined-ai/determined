import { Button, notification } from 'antd';
import queryString from 'query-string';
import React, { useEffect, useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';

import AuthToken from 'components/AuthToken';
import DeterminedAuth from 'components/DeterminedAuth';
import Logo, { Orientation } from 'components/Logo';
import Page from 'components/Page';
import PageMessage from 'components/PageMessage';
import { handleRelayState, samlUrl } from 'ee/SamlAuth';
import useAuthCheck from 'hooks/useAuthCheck';
import useFeature from 'hooks/useFeature';
import { defaultRoute, rbacDefaultRoute } from 'routes';
import { routeAll } from 'routes/utils';
import LogoGoogle from 'shared/assets/images/logo-sso-google-white.svg';
import LogoOkta from 'shared/assets/images/logo-sso-okta-white.svg';
import useUI from 'shared/contexts/stores/UI';
import usePolling from 'shared/hooks/usePolling';
import { RecordKey } from 'shared/types';
import { locationToPath, routeToReactUrl } from 'shared/utils/routes';
import { capitalize } from 'shared/utils/string';
import { useAuth } from 'stores/auth';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { BrandingType } from 'types';
import { Loadable } from 'utils/loadable';

import css from './SignIn.module.scss';

interface Queries {
  cli?: boolean;
  jwt?: string;
  redirect?: string;
}

const logoConfig: Record<RecordKey, string> = {
  google: LogoGoogle,
  okta: LogoOkta,
};

const SignIn: React.FC = () => {
  const { actions: uiActions } = useUI();
  const location = useLocation();
  const loadableAuth = useAuth();
  const authChecked = loadableAuth.authChecked;
  const isAuthenticated = Loadable.match(loadableAuth.auth, {
    Loaded: (auth) => auth.isAuthenticated,
    NotLoaded: () => false,
  });
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
  const [canceler] = useState(new AbortController());
  const rbacEnabled = useFeature().isOn('rbac');

  const queries: Queries = queryString.parse(location.search);
  const ssoQueries = handleRelayState(queries) as Record<string, boolean | string | undefined>;
  const ssoQueryString = queryString.stringify(ssoQueries);

  const externalAuthError = useMemo(() => {
    return authChecked && !isAuthenticated && !info.externalLoginUri && queries.jwt;
  }, [authChecked, isAuthenticated, info.externalLoginUri, queries.jwt]);

  /*
   * Check every so often to see if the user is authenticated.
   * For example, the user can authenticate in a different session,info
   * and this will pick up that auth and automatically redirect them into
   * their previous app. We don't run immediately because the router also
   * performs an auth check there as well upon the first page load.
   */
  usePolling(useAuthCheck(), { interval: 1000, runImmediately: false });

  /*
   * Check for when `isAuthenticated` becomes true and redirect
   * the user to the most recent requested page.
   */
  useEffect(() => {
    if (isAuthenticated) {
      // Stop the spinner, prepping for user redirect.
      uiActions.hideSpinner();

      // Show auth token via notification if requested via query parameters.
      if (queries.cli) notification.open({ description: <AuthToken />, duration: 0, message: '' });

      // Reroute the authenticated user to the app.
      if (!queries.redirect) {
        routeToReactUrl(
          locationToPath(location.state?.loginRedirect) ||
            (rbacEnabled ? rbacDefaultRoute.path : defaultRoute.path),
        );
      } else {
        routeAll(queries.redirect);
      }
    } else if (authChecked) {
      uiActions.hideSpinner();
    }
  }, [isAuthenticated, authChecked, info, location, queries, uiActions, rbacEnabled]);

  useEffect(() => {
    uiActions.hideChrome();
    return uiActions.showChrome;
  }, [uiActions]);

  // Stop the polling upon a dismount of this page.
  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  /*
   * Don't render sign in page if...
   *   1. jwt query param detected
   *   2. cluster has `externalLoginUri` defined
   *   3. authentication hasn't occurred yet
   * This will prevent the form from showing for a split second when
   * accessing a page from the browser when the user is already verified.
   */
  if (queries.jwt || info.externalLoginUri || !authChecked) return null;

  /*
   * An external auth error occurs when there are external auth urls,
   * auth fails with a jwt.
   */
  if (externalAuthError)
    return (
      <PageMessage title="Cluster Not Available">
        <p>Cluster is not ready. Please try again later.</p>
      </PageMessage>
    );

  return (
    <Page docTitle="Sign In" ignorePermissions>
      <div className={css.base}>
        <div className={css.content}>
          <Logo
            branding={info.branding || BrandingType.Determined}
            orientation={Orientation.Vertical}
          />
          <DeterminedAuth canceler={canceler} />
          {info.ssoProviders?.map((ssoProvider) => {
            const key = ssoProvider.name.toLowerCase();
            const logo = logoConfig[key] ? <img alt={key} src={logoConfig[key]} /> : '';
            return (
              <Button
                className={css.ssoButton}
                href={samlUrl(ssoProvider.ssoUrl, ssoQueryString)}
                key={key}
                size="large"
                type="primary">
                Sign in with {logo} {ssoProvider.name === key ? capitalize(key) : ssoProvider.name}
              </Button>
            );
          })}
        </div>
      </div>
    </Page>
  );
};

export default SignIn;
