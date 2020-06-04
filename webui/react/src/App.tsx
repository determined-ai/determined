import { Button, notification } from 'antd';
import React, { useCallback, useEffect } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

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
import FullPageSpinner from 'contexts/FullPageSpinner';
import Info from 'contexts/Info';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRouteTracker from 'hooks/useRouteTracker';
import useTheme from 'hooks/useTheme';
import { appRoutes } from 'routes';
import { updateFaviconType } from 'utils/browser';

import css from './App.module.scss';

const AppView: React.FC = () => {
  const { isAuthenticated, user } = Auth.useStateContext();
  const cluster = ClusterOverview.useStateContext();
  const info = Info.useStateContext();
  const showSpinner = FullPageSpinner.useStateContext();
  const setShowSpinner = FullPageSpinner.useActionContext();
  const username = user ? user.username : undefined;

  updateFaviconType(cluster.allocation !== 0);

  useRouteTracker();
  useTheme();

  useEffect(() => {
    if (info.telemetry.enabled && info.telemetry.segmentKey) {
      window.analytics.load(info.telemetry.segmentKey);
      window.analytics.identify(info.clusterId);
      window.analytics.page();
    }

    // Check to make sure the WebUI version matches the platform version.
    if (info.version !== process.env.VERSION) {
      const handleRefresh = (): void => window.location.reload(true);
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
    setShowSpinner({ opaque: true, type: FullPageSpinner.ActionType.Show });
  }, [ setShowSpinner ]);

  return (
    <div className={css.base}>
      {isAuthenticated && <NavBar username={username} />}
      {isAuthenticated && <AppContexts />}
      <div className={css.body}>
        {isAuthenticated && <SideBar />}
        <Switch>
          <Route exact path="/">
            <Redirect to="/det/dashboard" />
          </Route>
          <Router routes={appRoutes} />
        </Switch>
      </div>
      {showSpinner.isShowing && <Spinner fullPage opaque={showSpinner.isOpaque} />}
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
      FullPageSpinner.Provider,
    ]}>
      <AppView />
    </Compose>
  );
};

export default App;
