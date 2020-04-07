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
import Users from 'contexts/Users';
import useRouteTracker from 'hooks/useRouteTracker';
import { appRoutes } from 'routes';
import { getDeterminedInfo } from 'services/api';
import { lightTheme } from 'themes';

const AppView: React.FC = () => {
  const { isAuthenticated, user } = Auth.useStateContext();
  const username = user ? user.username : undefined;
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
  useRouteTracker();

  // TODO(hkang1): Rewrite this with the RestApiContext once it is available.
  const fetchDeterminedInfo = async (): Promise<void> => {
    try {
      const info = await getDeterminedInfo();
      window.analytics.identify(info.cluster_id);
    } catch (e) {
      window.analytics.track('Api Failed', e);
    }
  };

  useEffect(() => {
    fetchDeterminedInfo();
  }, [ fetchDeterminedInfo ]);

  return (
    <Compose components={[
      Auth.Provider,
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
