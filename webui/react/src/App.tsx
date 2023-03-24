import { App as AntdApp } from 'antd';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';

import Button from 'components/kit/Button';
import Link from 'components/Link';
import Navigation from 'components/Navigation';
import PageMessage from 'components/PageMessage';
import Router from 'components/Router';
import { ThemeProvider } from 'components/ThemeProvider';
import useAuthCheck from 'hooks/useAuthCheck';
import useKeyTracker from 'hooks/useKeyTracker';
import usePageVisibility from 'hooks/usePageVisibility';
import useResize from 'hooks/useResize';
import useRouteTracker from 'hooks/useRouteTracker';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import useTelemetry from 'hooks/useTelemetry';
import Omnibar from 'omnibar/Omnibar';
import appRoutes from 'routes';
import { paths, serverAddress } from 'routes/utils';
import Spinner from 'shared/components/Spinner/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { StoreProvider } from 'stores';
import {
  auth as authObservable,
  authChecked as observeAuthChecked,
  selectIsAuthenticated,
} from 'stores/auth';
import { fetchDeterminedInfo, initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import usersStore from 'stores/users';
import { correctViewportHeight, refreshPage } from 'utils/browser';
import { notification } from 'utils/dialogApi';
import { Loadable } from 'utils/loadable';

import css from './App.module.scss';

import 'antd/dist/reset.css';

const AppView: React.FC = () => {
  const resize = useResize();

  const isAuthenticated = useObservable(selectIsAuthenticated);
  const auth = useObservable(authObservable);
  const loadableUser = useObservable(usersStore.getCurrentUser());
  const infoLoadable = useDeterminedInfo();
  const authChecked = useObservable(observeAuthChecked);
  const info = Loadable.getOrElse(initInfo, infoLoadable);
  const [canceler] = useState(new AbortController());
  const { updateTelemetry } = useTelemetry();
  const checkAuth = useAuthCheck();

  const isServerReachable = useMemo(() => {
    return Loadable.match(infoLoadable, {
      Loaded: (info) => !!info.clusterId,
      NotLoaded: () => undefined,
    });
  }, [infoLoadable]);

  const fetchUsers = useCallback(() => usersStore.ensureUsersFetched(canceler), [canceler]);
  const fetchCurrentUser = useCallback(
    () => usersStore.ensureCurrentUserFetched(canceler),
    [canceler],
  );

  useEffect(() => {
    if (isServerReachable) checkAuth();
  }, [checkAuth, isServerReachable]);

  useKeyTracker();
  usePageVisibility();
  useRouteTracker();

  // Poll every 10 minutes
  usePolling(() => fetchDeterminedInfo(canceler), { interval: 600000 });

  useEffect(() => {
    if (isAuthenticated) {
      fetchUsers();
      fetchCurrentUser();
    }
  }, [isAuthenticated, fetchCurrentUser, fetchUsers]);

  useEffect(() => {
    /*
     * Check to make sure the WebUI version matches the platform version.
     * Skip this check for development version.
     */

    if (!process.env.IS_DEV && info.version !== process.env.VERSION) {
      const btn = (
        <Button type="primary" onClick={refreshPage}>
          Update Now
        </Button>
      );
      const message = 'New WebUI Version';
      const description = (
        <div>
          WebUI version <b>v{info.version}</b> is available. Check out what&apos;s new in our&nbsp;
          <Link external path={paths.docs('/release-notes.html')}>
            release notes
          </Link>
          .
        </div>
      );
      setTimeout(() => {
        notification.warning({
          btn,
          description,
          duration: 0,
          key: 'version-mismatch',
          message,
          placement: 'bottomRight',
        });
      }, 10);
    }
  }, [info]);

  // Detect telemetry settings changes and update telemetry library.
  useEffect(() => {
    updateTelemetry(auth, loadableUser, info);
  }, [auth, loadableUser, info, updateTelemetry]);

  // Abort cancel signal when app unmounts.
  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  // Correct the viewport height size when window resize occurs.
  useLayoutEffect(() => correctViewportHeight(), [resize]);

  return Loadable.match(infoLoadable, {
    Loaded: () => (
      <div className={css.base}>
        {authChecked ? (
          <>
            {isServerReachable ? (
              <SettingsProvider>
                <ThemeProvider>
                  <AntdApp>
                    <Navigation>
                      <main>
                        <Router routes={appRoutes} />
                      </main>
                    </Navigation>
                  </AntdApp>
                </ThemeProvider>
              </SettingsProvider>
            ) : (
              <PageMessage title="Server is Unreachable">
                <p>
                  Unable to communicate with the server at &quot;{serverAddress()}&quot;. Please
                  check the firewall and cluster settings.
                </p>
                <Button onClick={refreshPage}>Try Again</Button>
              </PageMessage>
            )}
            <Omnibar />
          </>
        ) : (
          <Spinner center />
        )}
      </div>
    ),
    NotLoaded: () => <Spinner center />,
  });
};

const App: React.FC = () => {
  return (
    <HelmetProvider>
      <StoreProvider>
        <DndProvider backend={HTML5Backend}>
          <AppView />
        </DndProvider>
      </StoreProvider>
    </HelmetProvider>
  );
};

export default App;
