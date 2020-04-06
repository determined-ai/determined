import React, { useEffect } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import styled, { ThemeProvider } from 'styled-components';

import NavBar from 'components/NavBar';
import Router from 'components/Router';
import Compose from 'Compose';
import ActiveExperiments from 'contexts/ActiveExperiments';
import Agents from 'contexts/Agents';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Info from 'contexts/Info';
import Users from 'contexts/Users';
import useRestApi from 'hooks/useRestApi';
import useRouteTracker from 'hooks/useRouteTracker';
import { ioDeterminedInfo } from 'ioTypes';
import { appRoutes } from 'routes';
import { jsonToDeterminedInfo } from 'services/decoder';
import { lightTheme } from 'themes';
import { DeterminedInfo } from 'types';

const AppView: React.FC = () => {
  const { isAuthenticated, user } = Auth.useStateContext();
  const info = Info.useStateContext();
  const setInfo = Info.useActionContext();
  const username = user ? user.username : undefined;
  const [ infoResponse, requestInfo ] =
    useRestApi<DeterminedInfo>(ioDeterminedInfo, { mappers: jsonToDeterminedInfo });

  useRouteTracker();

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

  return (
    <Base>
      {isAuthenticated && <NavBar username={username} />}
      <Switch>
        <Route exact path="/">
          <Redirect to="/det" />
        </Route>
        <Router routes={appRoutes} />
      </Switch>
    </Base>
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
    ]}>
      <ThemeProvider theme={lightTheme}>
        <AppView />
      </ThemeProvider>
    </Compose>
  );
};

const Base = styled.div`
  background-color: white;
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  > *:last-child { flex-grow: 1; }
`;

export default App;
