import { Button, notification } from 'antd';
import React, { useCallback, useEffect, useLayoutEffect } from 'react';
import { GlobalHotKeys } from 'react-hotkeys';
import { Redirect, Route, Switch } from 'react-router-dom';

import { setupAnalytics } from 'Analytics';
import Link from 'components/Link';
import Navigation from 'components/Navigation';
import NavigationTabbar from 'components/NavigationTabbar';
import NavigationTopbar from 'components/NavigationTopbar';
import Router from 'components/Router';
import Spinner from 'components/Spinner';
import Compose from 'Compose';
import Agents from 'contexts/Agents';
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
import Omnibar, { keymap as omnibarKeymap } from 'omnibar/Component';
import OmnibarCtx from 'omnibar/Context';
import appRoutes from 'routes';
import { getInfo } from 'services/api';
import { EmptyParams } from 'services/types';
import { DeterminedInfo, ResourceType } from 'types';
import { correctViewportHeight, refreshPage, updateFaviconType } from 'utils/browser';

import css from './App.module.scss';
import { paths } from './routes/utils';

const globalKeymap = {
  HIDE_OMNIBAR: [ 'esc' ], // TODO scope it to the component
  SHOW_OMNIBAR: [ 'ctrl+space' ],
};

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

  updateFaviconType(cluster[ResourceType.ALL].allocation !== 0);

  const OmnibarState = OmnibarCtx.useStateContext();
  const setOmnibar = OmnibarCtx.useActionContext();
  const globalKeyHandler = {
    HIDE_OMNIBAR: (): void => setOmnibar({ type: OmnibarCtx.ActionType.Hide }),
    SHOW_OMNIBAR: (): void => setOmnibar({ type: OmnibarCtx.ActionType.Show }),
  };

  useRouteTracker();
  useTheme();

  // Poll every 10 minutes
  usePolling(fetchInfo, { interval: 600000 });

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
        Check out what&apos;s new in our <Link external path={paths.docs('/release-notes.html')}>
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
        <div className={css.body}>
          <Navigation />
          <NavigationTopbar />
          <main><Router routes={appRoutes} /></main>
          <NavigationTabbar />
        </div>
      </Spinner>
      {OmnibarState.isShowing && <Omnibar />}
      <GlobalHotKeys handlers={globalKeyHandler} keyMap={globalKeymap} />

    </div>
  );
};

// <div className={css.base}>
//   {isAuthenticated && <NavBar username={username} />}
//   <div className={css.body}>
//     {isAuthenticated && <SideBar />}
//     <Switch>
//       <Route exact path="/">
//         <Redirect to={defaultAppRoute.path} />
//       </Route>
//       <Router routes={appRoutes} />
//     </Switch>
//   </div>

const App: React.FC = () => {

  return (
    <Compose components={[
      Auth.Provider,
      Info.Provider,
      Users.Provider,
      Agents.Provider,
      ClusterOverview.Provider,
      Commands.Provider,
      Notebooks.Provider,
      Shells.Provider,
      Tensorboards.Provider,
      UI.Provider,
      OmnibarCtx.Provider,
    ]}>
      <AppView />
    </Compose>
  );
};

export default App;
