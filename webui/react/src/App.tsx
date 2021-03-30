import { Button, notification } from 'antd';
import React, { useEffect, useLayoutEffect, useState } from 'react';

import { setupAnalytics } from 'Analytics';
import Link from 'components/Link';
import Navigation from 'components/Navigation';
import Router from 'components/Router';
import Compose from 'Compose';
import Agents from 'contexts/Agents';
import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Info, { useFetchInfo } from 'contexts/Info';
import UI from 'contexts/UI';
import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import useRouteTracker from 'hooks/useRouteTracker';
import useTheme from 'hooks/useTheme';
import appRoutes from 'routes';
import { correctViewportHeight, refreshPage } from 'utils/browser';

import css from './App.module.scss';
import { paths } from './routes/utils';

const AppView: React.FC = () => {
  const resize = useResize();
  const info = Info.useStateContext();
  const [ canceler ] = useState(new AbortController());

  const fetchInfo = useFetchInfo(canceler);

  useRouteTracker();
  useTheme();

  // Poll every 10 minutes
  usePolling(fetchInfo, { interval: 600000 });

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
    return () => canceler.abort();
  }, [ canceler ]);

  // Correct the viewport height size when window resize occurs.
  useLayoutEffect(() => correctViewportHeight(), [ resize ]);

  return (
    <div className={css.base}>
      <Navigation>
        <main><Router routes={appRoutes} /></main>
      </Navigation>
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
