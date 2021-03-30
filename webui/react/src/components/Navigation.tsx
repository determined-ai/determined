import React, { useEffect, useState } from 'react';

import { useFetchAgents } from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import UI from 'contexts/UI';
import usePolling from 'hooks/usePolling';
import { ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';

import css from './Navigation.module.scss';
import NavigationSideBar from './NavigationSideBar';
import NavigationTabbar from './NavigationTabbar';
import NavigationTopbar from './NavigationTopbar';
import Spinner from './Spinner';

interface Props {
  children?: React.ReactNode;
}

const Navigation: React.FC<Props> = ({ children }) => {
  const cluster = ClusterOverview.useStateContext();
  const ui = UI.useStateContext();
  const [ canceler ] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);

  updateFaviconType(cluster[ResourceType.ALL].allocation !== 0);
  usePolling(fetchAgents);

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
