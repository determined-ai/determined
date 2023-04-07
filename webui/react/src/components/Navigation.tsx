import React, { useEffect } from 'react';

import useFeature from 'hooks/useFeature';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import clusterStore from 'stores/cluster';
import determinedStore, { BrandingType } from 'stores/determinedInfo';
import permissionStore from 'stores/permissions';
import userStore from 'stores/users';
import workspaceStore from 'stores/workspaces';
import { ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';
import { useInitApi } from 'utils/dialogApi';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './Navigation.module.scss';
import NavigationSideBar from './NavigationSideBar';
import NavigationTabbar from './NavigationTabbar';

interface Props {
  children?: React.ReactNode;
}

const Navigation: React.FC<Props> = ({ children }) => {
  useInitApi();
  const { ui } = useUI();
  const info = useObservable(determinedStore.info);
  const loadableCurrentUser = useObservable(userStore.currentUser);
  const clusterOverview = useObservable(clusterStore.clusterOverview);

  useEffect(() => workspaceStore.startPolling(), []);

  useEffect(() => {
    updateFaviconType(
      Loadable.quickMatch(clusterOverview, false, (o) => o[ResourceType.ALL].allocation !== 0),
      info.branding || BrandingType.Determined,
    );
  }, [clusterOverview, info]);

  const rbacEnabled = useFeature().isOn('rbac'),
    mockAllPermission = useFeature().isOn('mock_permissions_all'),
    mockReadPermission = useFeature().isOn('mock_permissions_read');

  useEffect(() => {
    const shouldPoll =
      rbacEnabled &&
      !mockAllPermission &&
      !mockReadPermission &&
      Loadable.isLoaded(loadableCurrentUser);
    return permissionStore.startPolling({ condition: shouldPoll, delay: 120_000 });
  }, [loadableCurrentUser, mockAllPermission, mockReadPermission, rbacEnabled]);

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
