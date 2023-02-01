import React, { useEffect, useState } from 'react';

import useFeature from 'hooks/useFeature';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import usePolling from 'shared/hooks/usePolling';
import { useClusterStore } from 'stores/cluster';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { UserRolesService } from 'stores/userRoles';
import { useCurrentUser } from 'stores/users';
import { useFetchWorkspaces } from 'stores/workspaces';
import { BrandingType, ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';
import { Loadable, NotLoaded } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './Navigation.module.scss';
import NavigationSideBar from './NavigationSideBar';
import NavigationTabbar from './NavigationTabbar';

interface Props {
  children?: React.ReactNode;
}

const Navigation: React.FC<Props> = ({ children }) => {
  const { ui } = useUI();
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
  const [canceler] = useState(new AbortController());

  const clusterOverview = useObservable(useClusterStore().clusterOverview);

  const fetchWorkspaces = useFetchWorkspaces(canceler);
  const currentUser = useCurrentUser();
  const fetchMyRoles = UserRolesService.fetchUserAssignmentsAndRoles(canceler);

  usePolling(fetchWorkspaces);

  useEffect(() => {
    updateFaviconType(
      Loadable.quickMatch(clusterOverview, false, (o) => o[ResourceType.ALL].allocation !== 0),
      info.branding || BrandingType.Determined,
    );
  }, [clusterOverview, info]);

  const rbacEnabled = useFeature().isOn('rbac'),
    mockAllPermission = useFeature().isOn('mock_permissions_all'),
    mockReadPermission = useFeature().isOn('mock_permissions_read');
  usePolling(
    () => {
      if (rbacEnabled && !mockAllPermission && !mockReadPermission && currentUser !== NotLoaded) {
        fetchMyRoles();
      }
    },
    { interval: 120000 },
  );

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
