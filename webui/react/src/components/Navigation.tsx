import React, { useCallback, useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { useFetchAgents } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import { importApiStorageRemote, setRRMode } from 'recordReplay';

import css from './Navigation.module.scss';
import NavigationSideBar from './NavigationSideBar';
import NavigationTabbar from './NavigationTabbar';
import NavigationTopbar from './NavigationTopbar';
import Spinner from './Spinner';

interface Props {
  children?: React.ReactNode;
}

const Navigation: React.FC<Props> = ({ children }) => {
  const { auth, ui } = useStore();
  const [ canceler ] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);

  const fetchAuthOnly = useCallback(() => {
    if (auth.isAuthenticated) fetchAgents();
  }, [ auth.isAuthenticated, fetchAgents ]);

  usePolling(fetchAuthOnly);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <Spinner spinning={ui.showSpinner}>
      <div className={css.base}>
        <NavigationSideBar />
        <NavigationTopbar />
        {children}
        <NavigationTabbar />
      </div>
    </Spinner>
  );
};

export default Navigation;
