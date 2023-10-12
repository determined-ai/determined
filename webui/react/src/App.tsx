import { App as AntdApp } from 'antd';
import { useObservable } from 'micro-observables';
import React, { useEffect, useLayoutEffect, useReducer, useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { HelmetProvider } from 'react-helmet-async';
import { useParams } from 'react-router-dom';
import { themeLightDetermined, themeDarkDetermined, ThemeHandler } from 'components/kit/Theme';
import JupyterLabGlobal from 'components/JupyterLabGlobal';
import Button from 'components/kit/Button';
import Spinner from 'components/kit/Spinner';
import useUI, { UIProvider } from 'components/kit/Theme';
import { notification } from 'components/kit/Toast';
import { ConfirmationProvider } from 'components/kit/useConfirm';
import { Loadable } from 'components/kit/utils/loadable';
import Link from 'components/Link';
import Navigation from 'components/Navigation';
import PageMessage from 'components/PageMessage';
import Router from 'components/Router';
import useAuthCheck from 'hooks/useAuthCheck';
import useKeyTracker from 'hooks/useKeyTracker';
import usePageVisibility from 'hooks/usePageVisibility';
import usePermissions from 'hooks/usePermissions';
import useResize from 'hooks/useResize';
import useRouteTracker from 'hooks/useRouteTracker';
import { useSettings } from 'hooks/useSettings';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import useTelemetry from 'hooks/useTelemetry';
import { config as themeConfig, Settings as themeSettings } from 'hooks/useTheme.settings';
import Omnibar from 'omnibar/Omnibar';
import appRoutes from 'routes';
import { paths, serverAddress } from 'routes/utils';
import authStore from 'stores/auth';
import clusterStore from 'stores/cluster';
import determinedStore from 'stores/determinedInfo';
import userStore from 'stores/users';
import userSettings from 'stores/userSettings';
import workspaceStore from 'stores/workspaces';
import { correctViewportHeight, refreshPage } from 'utils/browser';

import css from './App.module.scss';

import 'antd/dist/reset.css';
import '@hpe.com/glide-data-grid/dist/index.css';

const AppView: React.FC = () => {
  const resize = useResize();

  const loadableAuth = useObservable(authStore.auth);
  const isAuthChecked = useObservable(authStore.isChecked);
  const isAuthenticated = useObservable(authStore.isAuthenticated);
  const loadableUser = useObservable(userStore.currentUser);
  const loadableInfo = useObservable(determinedStore.loadableInfo);
  const isServerReachable = useObservable(determinedStore.isServerReachable);
  const info = useObservable(determinedStore.info);
  const { updateTelemetry } = useTelemetry();
  const checkAuth = useAuthCheck();
  const {
    settings,
    isLoading: isSettingsLoading,
    updateSettings,
  } = useSettings<themeSettings>(themeConfig);
  const [isSettingsReady, setIsSettingsReady] = useState(false);
  const { ui, actions: uiActions } = useUI();

  useEffect(() => {
    if (isServerReachable) checkAuth();
  }, [checkAuth, isServerReachable]);

  useKeyTracker();
  usePageVisibility();
  useRouteTracker();

  useEffect(() => (isAuthenticated ? userStore.fetchCurrentUser() : undefined), [isAuthenticated]);
  useEffect(() => (isAuthenticated ? clusterStore.startPolling() : undefined), [isAuthenticated]);
  useEffect(() => (isAuthenticated ? userSettings.startPolling() : undefined), [isAuthenticated]);
  useEffect(
    () => (isAuthenticated ? userStore.startPolling({ delay: 60_000 }) : undefined),
    [isAuthenticated],
  );
  useEffect(
    () => (isAuthenticated ? workspaceStore.startPolling({ delay: 60_000 }) : undefined),
    [isAuthenticated],
  );
  useEffect(() => determinedStore.startPolling({ delay: 600_000 }), []);

  useEffect(() => {
    /*
     * Check to make sure the WebUI version matches the platform version.
     * Skip this check for development version.
     */
    Loadable.quickMatch(loadableInfo, undefined, undefined, (info) => {
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
      undefined,
      ([auth, user, info]) => updateTelemetry(auth, user, info),
    );
  }, [loadableAuth, loadableInfo, loadableUser, updateTelemetry]);

  // Correct the viewport height size when window resize occurs.
  useLayoutEffect(() => correctViewportHeight(), [resize]);

  // Update setting mode when mode changes.
  useLayoutEffect(() => {
    if (isSettingsLoading) return;

    if (isSettingsReady) {
      // We have read from the settings, going forward any mode difference requires an update.
      if (settings.mode !== ui.mode) updateSettings({ mode: ui.mode });
    } else {
      // Initially set the mode from settings.
      uiActions.setMode(settings.mode);
      setIsSettingsReady(true);
    }
  }, [isSettingsReady, settings, uiActions, ui.mode, isSettingsLoading, updateSettings]);

  // Check permissions and params for JupyterLabGlobal.
  const { canCreateNSC, canCreateWorkspaceNSC } = usePermissions();
  const { workspaceId } = useParams<{
    workspaceId: string;
  }>();
  const loadableWorkspace = useObservable(workspaceStore.getWorkspace(Number(workspaceId ?? '')));
  const workspace = Loadable.getOrElse(undefined, loadableWorkspace);

  return Loadable.match(loadableInfo, {
    Failed: () => null, // TODO display any errors we receive
    Loaded: () => (
      <div className={css.base}>
        {isAuthChecked ? (
          <>
            {isServerReachable ? (
              <SettingsProvider>
                <UIProvider branding={info.branding} theme={themeLightDetermined}>
                  <AntdApp>
                    <ConfirmationProvider>
                      <Navigation>
                        <JupyterLabGlobal
                          enabled={
                            Loadable.isLoaded(loadableUser) &&
                            (workspace ? canCreateWorkspaceNSC({ workspace }) : canCreateNSC)
                          }
                          workspace={workspace ?? undefined}
                        />
                        <Omnibar />
                        <main>
                          <Router routes={appRoutes} />
                        </main>
                      </Navigation>
                    </ConfirmationProvider>
                  </AntdApp>
                </UIProvider>
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
          </>
        ) : (
          <Spinner center spinning />
        )}
      </div>
    ),
    NotLoaded: () => <Spinner center spinning />,
  });
};

const App: React.FC = () => {
  return (
    <HelmetProvider>
      <DndProvider backend={HTML5Backend}>
        <ThemeHandler>
          <AppView />
        </ThemeHandler>
      </DndProvider>
    </HelmetProvider>
  );
};

export default App;
