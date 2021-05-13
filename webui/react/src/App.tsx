import { Button, notification } from 'antd';
import React, { useEffect, useLayoutEffect, useState } from 'react';
import { HelmetProvider } from 'react-helmet-async';

import { setupAnalytics } from 'Analytics';
import Link from 'components/Link';
import Navigation from 'components/Navigation';
import Router from 'components/Router';
import StoreProvider, { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { useFetchInfo } from 'hooks/useFetch';
import useKeyTracker,{ KeyCode, keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import usePolling from 'hooks/usePolling';
import useResize from 'hooks/useResize';
import useRouteTracker from 'hooks/useRouteTracker';
import useTheme from 'hooks/useTheme';
import Omnibar from 'omnibar/Omnibar';
import appRoutes from 'routes';
import { correctViewportHeight, refreshPage } from 'utils/browser';

import css from './App.module.scss';
import { paths } from './routes/utils';

const AppView: React.FC = () => {
  const resize = useResize();
  const [ canceler ] = useState(new AbortController());
  const { info, ui } = useStore();
  const storeDispatch = useStoreDispatch();

  const fetchInfo = useFetchInfo(canceler);

  useKeyTracker();
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

  useEffect(() => {
    const keyDownListener = (e: KeyboardEvent) => {
      if (ui.omnibar.isShowing && e.code === KeyCode.Escape) {
        storeDispatch({ type: StoreAction.HideOmnibar });
      } else if (!ui.omnibar.isShowing && e.code === KeyCode.Space && e.ctrlKey) {
        storeDispatch({ type: StoreAction.ShowOmnibar });
      }
    };

    keyEmitter.on(KeyEvent.KeyDown, keyDownListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyDown, keyDownListener);
    };
  }, [ ui.omnibar.isShowing, storeDispatch ]);

  // Correct the viewport height size when window resize occurs.
  useLayoutEffect(() => correctViewportHeight(), [ resize ]);

  return (
    <div className={css.base}>
      <Navigation>
        <main>
          <Router routes={appRoutes} />
          {ui.omnibar.isShowing && <Omnibar />}
        </main>
      </Navigation>
    </div>
  );
};

const App: React.FC = () => {
  return (
    <HelmetProvider>
      <StoreProvider>
        <AppView />
      </StoreProvider>
    </HelmetProvider>
  );
};

export default App;
