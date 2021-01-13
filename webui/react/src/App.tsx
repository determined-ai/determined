import { Button, notification } from 'antd';
import React, { useCallback, useEffect, useLayoutEffect } from 'react';

import { setupAnalytics } from 'Analytics';
import Link from 'components/Link';
import Navigation from 'components/Navigation';
import NavigationTabbar from 'components/NavigationTabbar';
import NavigationTopbar from 'components/NavigationTopbar';
import Router from 'components/Router';
import Spinner from 'components/Spinner';
import Compose from 'Compose';
import ActiveExperiments from 'contexts/ActiveExperiments';
import Agents from 'contexts/Agents';
import AppContexts from 'contexts/AppContexts';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Info from 'contexts/Info';
import UI from 'contexts/UI';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import useRestApi from 'hooks/useRestApi';
import useRouteTracker from 'hooks/useRouteTracker';
import useTheme from 'hooks/useTheme';
import appRoutes from 'routes';
import { getInfo } from 'services/api';
import { EmptyParams } from 'services/types';
import { DeterminedInfo, ResourceType } from 'types';
import { correctViewportHeight, refreshPage, updateFaviconType } from 'utils/browser';

import css from './App.module.scss';

const AppView: React.FC = () => {
  const resize = useResize();
  const { isAuthenticated } = Auth.useStateContext();
  const ui = UI.useStateContext();
  const cluster = ClusterOverview.useStateContext();
  const info = Info.useStateContext();
  const setInfo = Info.useActionContext();
  const setUI = UI.useActionContext();
  const [ infoResponse, triggerInfoRequest ] = useRestApi<EmptyParams, DeterminedInfo>(getInfo, {});
  const classes = [ css.base ];

  const fetchInfo = useCallback(() => triggerInfoRequest({}), [ triggerInfoRequest ]);

  if (!ui.showChrome || !isAuthenticated) classes.push(css.noChrome);

  updateFaviconType(cluster[ResourceType.GPU].allocation !== 0);

  useRouteTracker();
  useTheme();

  // Poll every 10 minutes
  usePolling(fetchInfo, { delay: 600000 });

  useEffect(() => {
    if (!infoResponse.data) return;
    setInfo({ type: Info.ActionType.Set, value: infoResponse.data });
  }, [ infoResponse, setInfo ]);

  useEffect(() => {
    setupAnalytics(info);

    // Check to make sure the WebUI version matches the platform version.
    if (info.version !== process.env.VERSION) {
      const btn = <Button type="primary" onClick={refreshPage}>Update Now</Button>;
      const message = 'New WebUI Version';
      const description = <div>
        WebUI version <b>v{info.version}</b> is available.
        Check out what&apos;s new in our <Link external path="/docs/release-notes.html">
          release notes
        </Link>.
      </div>;
      notification.warn({
        btn,
        description,
        duration: 0,
        key: 'version-mismatch',
        message,
        placement: 'bottomRight',
      });
    }
  }, [ info ]);

  useEffect(() => {
    setUI({ type: UI.ActionType.ShowSpinner });
  }, [ setUI ]);

  // Correct the viewport height size when window resize occurs.
  useLayoutEffect(() => correctViewportHeight(), [ resize ]);

  return (
    <div className={classes.join(' ')}>
      <Spinner spinning={ui.showSpinner}>
        {isAuthenticated && <AppContexts />}
        <div className={css.body}>
          <Navigation />
          <NavigationTopbar />
          <main><Router routes={appRoutes} /></main>
          <NavigationTabbar />
        </div>
      </Spinner>
    </div>
  );
};

const App: React.FC = () => {
  return (
    <Compose components={[
      Auth.Provider,
      Info.Provider,
      Users.Provider,
      Agents.Provider,
      ClusterOverview.Provider,
      ActiveExperiments.Provider,
      Commands.Provider,
      Notebooks.Provider,
      Shells.Provider,
      Tensorboards.Provider,
      UI.Provider,
    ]}>
      <AppView />
    </Compose>
  );
};

export default App;
