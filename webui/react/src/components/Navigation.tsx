import React, { useCallback, useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { useFetchAgents, useFetchResourcePools } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';

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
  const fetchResourcePools = useFetchResourcePools(canceler);

  const fetchAuthOnly = useCallback(async () => {
    if (auth.isAuthenticated) await fetchAgents();
  }, [ auth.isAuthenticated, fetchAgents ]);

  usePolling(fetchAuthOnly);

  useEffect(() => {
    fetchResourcePools();

    return () => canceler.abort();
  }, [ canceler, fetchResourcePools ]);

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
