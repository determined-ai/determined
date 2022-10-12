import React, { useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import {
  useFetchMyRoles,
  useFetchPinnedWorkspaces,
  useFetchResourcePools,
  useFetchUserSettings,
} from 'hooks/useFetch';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import usePolling from 'shared/hooks/usePolling';
import { initClusterOverview, useClusterOverview, useFetchAgents } from 'stores/agents';
import { BrandingType, ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';
import { Loadable } from 'utils/loadable';

import css from './Navigation.module.scss';
import NavigationSideBar from './NavigationSideBar';
import NavigationTabbar from './NavigationTabbar';

interface Props {
  children?: React.ReactNode;
}

const Navigation: React.FC<Props> = ({ children }) => {
  const { ui } = useUI();
  const { info } = useStore();
  const [canceler] = useState(new AbortController());
  const overview = Loadable.getOrElse(initClusterOverview, useClusterOverview());

  const fetchAgents = useFetchAgents(canceler);
  const fetchResourcePools = useFetchResourcePools(canceler);
  const fetchPinnedWorkspaces = useFetchPinnedWorkspaces(canceler);
  const fetchUserSettings = useFetchUserSettings(canceler);
  const fetchMyRoles = useFetchMyRoles(canceler);

  usePolling(fetchAgents);
  usePolling(fetchPinnedWorkspaces);
  usePolling(fetchUserSettings, { interval: 60000 });

  useEffect(() => {
    updateFaviconType(
      overview[ResourceType.ALL].allocation !== 0,
      info.branding || BrandingType.Determined,
    );
  }, [overview, info]);

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
