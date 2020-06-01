import React, { useEffect } from 'react';
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
import useRestApi from 'hooks/useRestApi';
import useRouteTracker from 'hooks/useRouteTracker';
import useTheme from 'hooks/useTheme';
import { ioDeterminedInfo } from 'ioTypes';
import { appRoutes } from 'routes';
import { jsonToDeterminedInfo } from 'services/decoder';
import { DeterminedInfo } from 'types';
import { updateFaviconType } from 'utils/browser';

import css from './App.module.scss';

const AppView: React.FC = () => {
  const { isAuthenticated, user } = Auth.useStateContext();
  const cluster = ClusterOverview.useStateContext();
  const info = Info.useStateContext();
  const setInfo = Info.useActionContext();
  const showSpinner = FullPageSpinner.useStateContext();
  const setShowSpinner = FullPageSpinner.useActionContext();
  const username = user ? user.username : undefined;
  const [ infoResponse, requestInfo ] =
    useRestApi<DeterminedInfo>(ioDeterminedInfo, { mappers: jsonToDeterminedInfo });

  updateFaviconType(cluster.allocation !== 0);

  useRouteTracker();
  useTheme();

  useEffect(() => requestInfo({ url: '/info' }), [ requestInfo ]);

  useEffect(() => {
    if (!info.telemetry.enabled || !info.telemetry.segmentKey) return;
    window.analytics.load(info.telemetry.segmentKey);
    window.analytics.identify(info.clusterId);
    window.analytics.page();
  }, [ info ]);

  useEffect(() => {
    if (!infoResponse.data) return;
    setInfo({ type: Info.ActionType.Set, value: infoResponse.data });
  }, [ infoResponse, setInfo ]);

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
