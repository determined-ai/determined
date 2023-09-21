import React, { useEffect } from 'react';

import Spinner from 'components/kit/Spinner';
import useUI from 'components/kit/Theme';
import { useInitApi } from 'components/kit/Toast';
import clusterStore from 'stores/cluster';
import determinedStore, { BrandingType } from 'stores/determinedInfo';
import permissionStore from 'stores/permissions';
import userStore from 'stores/users';
import { ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';
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

  useEffect(() => {
    updateFaviconType(
      Loadable.quickMatch(clusterOverview, false, (o) => o[ResourceType.ALL].allocation !== 0),
      info.branding || BrandingType.Determined,
    );
  }, [clusterOverview, info]);

  const { rbacEnabled } = useObservable(determinedStore.info);

  useEffect(() => {
    const shouldPoll = rbacEnabled && Loadable.isLoaded(loadableCurrentUser);
    return permissionStore.startPolling({ condition: shouldPoll, delay: 120_000 });
  }, [loadableCurrentUser, rbacEnabled]);

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
