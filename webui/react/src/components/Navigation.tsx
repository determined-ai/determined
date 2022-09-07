import React, { useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { useFetchAgents, useFetchPinnedWorkspaces,
  useFetchResourcePools, useFetchUserSettings } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import Spinner from 'shared/components/Spinner/Spinner';

import css from './Navigation.module.scss';
import NavigationSideBar from './NavigationSideBar';
import NavigationTabbar from './NavigationTabbar';

interface Props {
  children?: React.ReactNode;
}

const Navigation: React.FC<Props> = ({ children }) => {
  const { ui } = useStore();
  const [ canceler ] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);
  const fetchResourcePools = useFetchResourcePools(canceler);
  const fetchPinnedWorkspaces = useFetchPinnedWorkspaces(canceler);
  const fetchUserSettings = useFetchUserSettings(canceler);

  usePolling(fetchAgents);
  usePolling(fetchPinnedWorkspaces);
  usePolling(fetchUserSettings);

  useEffect(() => {
    fetchResourcePools();

    return () => canceler.abort();
  }, [ canceler, fetchResourcePools ]);

  return (
    <Spinner spinning={ui.showSpinner}>
      <div className={css.base}>
        <NavigationSideBar />
        {children}
        <NavigationTabbar />
      </div>
    </Spinner>
  );
};

export default Navigation;
