import { Button, notification } from 'antd';
import React, { useCallback, useEffect } from 'react';

import NavBar from 'components/NavBar';
import Router from 'components/Router';
import SideBar from 'components/SideBar';
import Spinner from 'components/Spinner';
import Compose from 'Compose';
import ActiveExperiments from 'contexts/ActiveExperiments';
import Agents from 'contexts/Agents';
import AppContexts from 'contexts/AppContexts';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Experiments from 'contexts/Experiments';
import Info from 'contexts/Info';
import UI from 'contexts/UI';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import useRouteTracker from 'hooks/useRouteTracker';
import useTheme from 'hooks/useTheme';
import { appRoutes } from 'routes';
import { getInfo } from 'services/api';
import { DeterminedInfo } from 'types';
import { updateFaviconType } from 'utils/browser';
import { parseUrl } from 'utils/routes';

import css from './App.module.scss';

const AppView: React.FC = () => {
  const { isAuthenticated, user } = Auth.useStateContext();
  const ui = UI.useStateContext();
  const cluster = ClusterOverview.useStateContext();
  const info = Info.useStateContext();
  const setInfo = Info.useActionContext();
  const setUI = UI.useActionContext();
  const username = user ? user.username : undefined;
  const [ infoResponse, requestInfo ] = useRestApiSimple<{}, DeterminedInfo>(getInfo, {});
  const classes = [ css.base ];

  const fetchInfo = useCallback(() => requestInfo({}), [ requestInfo ]);

  if (!ui.showChrome) classes.push(css.noChrome);

  updateFaviconType(cluster.allocation !== 0);

  useRouteTracker();
  useTheme();

  // Poll every 10 minutes
  usePolling(fetchInfo, { delay: 600000 });

  useEffect(() => {
    if (!infoResponse.data) return;
    setInfo({ type: Info.ActionType.Set, value: infoResponse.data });
  }, [ infoResponse, setInfo ]);

  useEffect(() => {
    /*
     * Check for analytics library and the validity of the segment key:
     * 32 characters composed of upper case letters, lower case letters and numbers 0-9.
     */
    if (window.analytics && info.telemetry.enabled &&
        info.telemetry.segmentKey && /^[a-z0-9]{32}$/i.test(info.telemetry.segmentKey)) {
      window.analytics.load(info.telemetry.segmentKey);
      window.analytics.identify(info.clusterId);
      window.analytics.page();
    }

    // Check to make sure the WebUI version matches the platform version.
    if (info.version !== process.env.VERSION) {
      /*
       * The method of cache busting here is to send a query string as most
       * modern browsers treat different URLs as different files, causing a
       * request of a fresh copy. The previous method of using `location.reload`
       * with a `forceReload` boolean has been deprecated and not reliable.
       */
      const handleRefresh = (): void => {
        const now = Date.now();
        const url = parseUrl(window.location.href);
        url.search = url.search ? `${url.search}&ts=${now}` : `ts=${now}`;
        window.location.href = url.toString();
      };
      const btn = <Button type="primary" onClick={handleRefresh}>Update Now</Button>;
      const message = 'New WebUI Version';
      const description = <div>
        WebUI version <b>v{info.version}</b> is available.
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
    setUI({ opaque: true, type: UI.ActionType.ShowSpinner });
  }, [ setUI ]);

  return (
    <div className={classes.join(' ')}>
      {isAuthenticated && ui.showChrome && <NavBar username={username} />}
      {isAuthenticated && <AppContexts />}
      <div className={css.body}>
        {isAuthenticated && ui.showChrome && <SideBar />}
        <Router routes={appRoutes} />
      </div>
      {ui.showSpinner && <Spinner fullPage opaque={ui.opaqueSpinner} />}
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
      Experiments.Provider,
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
