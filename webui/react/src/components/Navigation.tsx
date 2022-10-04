import React, { useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import {
  useFetchAgents,
  useFetchKnownRoles,
  useFetchMyRoles,
  useFetchPinnedWorkspaces,
  useFetchResourcePools,
  useFetchUserSettings,
} from 'hooks/useFetch';
import Spinner from 'shared/components/Spinner/Spinner';
import usePolling from 'shared/hooks/usePolling';

import css from './Navigation.module.scss';
import NavigationSideBar from './NavigationSideBar';
import NavigationTabbar from './NavigationTabbar';

interface Props {
  children?: React.ReactNode;
}

const Navigation: React.FC<Props> = ({ children }) => {
  const { ui } = useStore();
  const [canceler] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);
  const fetchResourcePools = useFetchResourcePools(canceler);
  const fetchPinnedWorkspaces = useFetchPinnedWorkspaces(canceler);
  const fetchUserSettings = useFetchUserSettings(canceler);
  const fetchKnownRoles = useFetchKnownRoles(canceler);
  const fetchMyRoles = useFetchMyRoles(canceler);

  usePolling(fetchAgents);
  usePolling(fetchPinnedWorkspaces);
  usePolling(fetchUserSettings, { interval: 60000 });

  useEffect(() => {
    fetchResourcePools();

    return () => canceler.abort();
  }, [canceler, fetchResourcePools]);

  const rbacEnabled = useFeature().isOn('rbac');
  useEffect(() => {
    if (rbacEnabled) {
      fetchMyRoles();
      fetchKnownRoles();
    }
    return () => canceler.abort();
  }, [canceler, fetchKnownRoles, fetchMyRoles, rbacEnabled]);

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
