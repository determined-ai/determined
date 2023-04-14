import { App as AntdApp } from 'antd';
import { useObservable } from 'micro-observables';
import React, { useEffect, useLayoutEffect } from 'react';
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
import { StoreProvider } from 'stores';
import authStore from 'stores/auth';
import determinedStore from 'stores/determinedInfo';
import userStore from 'stores/users';
import { correctViewportHeight, refreshPage } from 'utils/browser';
import { notification } from 'utils/dialogApi';
import { Loadable } from 'utils/loadable';

import css from './App.module.scss';

import 'antd/dist/reset.css';
import '@glideapps/glide-data-grid/dist/index.css';

const AppView: React.FC = () => {
  const resize = useResize();

  const loadableAuth = useObservable(authStore.auth);
  const isAuthChecked = useObservable(authStore.isChecked);
  const isAuthenticated = useObservable(authStore.isAuthenticated);
  const loadableUser = useObservable(userStore.currentUser);
  const loadableInfo = useObservable(determinedStore.loadableInfo);
  const isServerReachable = useObservable(determinedStore.isServerReachable);
  const { updateTelemetry } = useTelemetry();
  const checkAuth = useAuthCheck();

  useEffect(() => {
    if (isServerReachable) checkAuth();
  }, [checkAuth, isServerReachable]);

  useKeyTracker();
  usePageVisibility();
  useRouteTracker();

  useEffect(() => (isAuthenticated ? userStore.fetchCurrentUser() : undefined), [isAuthenticated]);
  useEffect(() => (isAuthenticated ? userStore.fetchUsers() : undefined), [isAuthenticated]);
  useEffect(() => determinedStore.startPolling({ delay: 600_000 }), []);

  useEffect(() => {
    /*
     * Check to make sure the WebUI version matches the platform version.
     * Skip this check for development version.
     */
    Loadable.quickMatch(loadableInfo, undefined, (info) => {
      if (!process.env.IS_DEV && info.version !== process.env.VERSION) {
        const btn = (
          <Button type="primary" onClick={refreshPage}>
            Update Now
          </Button>
        );
        const message = 'New WebUI Version';
        const description = (
          <div>
            WebUI version <b>v{info.version}</b> is available. Check out what&apos;s new in
            our&nbsp;
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
    });
  }, [loadableInfo]);

  // Detect telemetry settings changes and update telemetry library.
  useEffect(() => {
    Loadable.quickMatch(
      Loadable.all([loadableAuth, loadableUser, loadableInfo]),
      undefined,
      ([auth, user, info]) => updateTelemetry(auth, user, info),
    );
  }, [loadableAuth, loadableInfo, loadableUser, updateTelemetry]);

  // Correct the viewport height size when window resize occurs.
  useLayoutEffect(() => correctViewportHeight(), [resize]);

  return Loadable.match(loadableInfo, {
    Loaded: () => (
      <div className={css.base}>
        {isAuthChecked ? (
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
