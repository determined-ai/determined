import React, { useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import { useFetchPinnedWorkspaces } from 'hooks/useFetch';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import usePolling from 'shared/hooks/usePolling';
import { useClusterOverview, useFetchAgents } from 'stores/agents';
import { useFetchResourcePools } from 'stores/resourcePools';
import { useFetchUserRolesAndAssignments } from 'stores/userRoles';
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
  const overview = useClusterOverview();

  const fetchAgents = useFetchAgents(canceler);
  const fetchResourcePools = useFetchResourcePools(canceler);
  const fetchPinnedWorkspaces = useFetchPinnedWorkspaces(canceler);
  const fetchMyRoles = useFetchUserRolesAndAssignments(canceler);

  usePolling(fetchAgents);
  usePolling(fetchPinnedWorkspaces);

  useEffect(() => {
    updateFaviconType(
      Loadable.quickMatch(overview, false, (o) => o[ResourceType.ALL].allocation !== 0),
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
