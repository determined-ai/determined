import React, { useEffect, useState } from 'react';

import useFeature from 'hooks/useFeature';
import {
  useFetchAgents,
  useFetchMyRoles,
  useFetchPinnedWorkspaces,
  useFetchResourcePools,
  useFetchUserSettings,
} from 'hooks/useFetch';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import usePolling from 'shared/hooks/usePolling';

import css from './Navigation.module.scss';
import NavigationSideBar from './NavigationSideBar';
import NavigationTabbar from './NavigationTabbar';

interface Props {
  children?: React.ReactNode;
}

const Navigation: React.FC<Props> = ({ children }) => {
  const { ui } = useUI();
  const [canceler] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);
  const fetchResourcePools = useFetchResourcePools(canceler);
  const fetchPinnedWorkspaces = useFetchPinnedWorkspaces(canceler);
  const fetchUserSettings = useFetchUserSettings(canceler);
  const fetchMyRoles = useFetchMyRoles(canceler);

  usePolling(fetchAgents);
  usePolling(fetchPinnedWorkspaces);
  usePolling(fetchUserSettings, { interval: 60000 });

  const rbacEnabled = useFeature().isOn('rbac');
  usePolling(
    () => {
      if (rbacEnabled) {
        fetchMyRoles();
      }
    },
    { interval: 120000 },
  );

  useEffect(() => {
    fetchResourcePools();

    return () => canceler.abort();
  }, [canceler, fetchResourcePools]);

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
