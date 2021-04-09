import React, { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import { useStore } from 'contexts/Store';
import { handlePath, paths } from 'routes/utils';
import { ResourceType } from 'types';
import { launchNotebook } from 'utils/task';

import ActionSheet from './ActionSheet';
import Icon from './Icon';
import Link, { Props as LinkProps } from './Link';
import css from './NavigationTabbar.module.scss';

interface ToolbarItemProps extends LinkProps {
  badge?: number;
  icon: string;
  label: string;
  status?: string;
}

const ToolbarItem: React.FC<ToolbarItemProps> = ({ path, status, ...props }: ToolbarItemProps) => {
  const location = useLocation();
  const classes = [ css.toolbarItem ];
  const [ isActive, setIsActive ] = useState(false);

  if (isActive) classes.push(css.active);

  useEffect(() => setIsActive(location.pathname === path), [ location.pathname, path ]);

  return (
    <Link className={classes.join(' ')} path={path} {...props}>
      <Icon name={props.icon} size="large" />
      {status && <div className={css.status}>{status}</div>}
    </Link>
  );
};

const NavigationTabbar: React.FC = () => {
  const { auth, cluster: overview, ui } = useStore();
  const [ isShowingOverflow, setIsShowingOverflow ] = useState(false);

  const cluster = overview[ResourceType.ALL].allocation === 0 ?
    undefined : `${overview[ResourceType.ALL].allocation}%`;
  const showNavigation = auth.isAuthenticated && ui.showChrome;

  const handleOverflowOpen = useCallback(() => setIsShowingOverflow(true), []);
  const handleActionSheetCancel = useCallback(() => setIsShowingOverflow(false), []);
  const handleLaunchNotebook = useCallback((cpuOnly = false) => {
    launchNotebook(cpuOnly ? 0 : 1);
    setIsShowingOverflow(false);
  }, []);

  const handlePathUpdate = useCallback((e, path) => {
    handlePath(e, { path });
    setIsShowingOverflow(false);
  }, []);

  if (!showNavigation) return null;

  return (
    <nav className={css.base}>
      <div className={css.toolbar}>
        <ToolbarItem icon="dashboard" label="Dashboard" path={paths.dashboard()} />
        <ToolbarItem icon="experiment" label="Experiments" path={paths.experimentList()} />
        <ToolbarItem icon="tasks" label="Tasks" path={paths.taskList()} />
        <ToolbarItem icon="cluster" label="Cluster" path={paths.cluster()} status={cluster} />
        <ToolbarItem icon="overflow-vertical" label="Overflow Menu" onClick={handleOverflowOpen} />
      </div>
      <ActionSheet
        actions={[
          {
            icon: 'notebook',
            label: 'Launch Notebook',
            onClick: () => handleLaunchNotebook(),
          },
          {
            icon: 'notebook',
            label: 'Launch CPU-only Notebook',
            onClick: () => handleLaunchNotebook(true),
          },
          {
            icon: 'logs',
            label: 'Master Logs',
            onClick: e => handlePathUpdate(e, paths.masterLogs()),
          },
          {
            external: true,
            icon: 'docs',
            label: 'Docs',
            path: paths.docs(),
            popout: true,
          },
          {
            external: true,
            icon: 'cloud',
            label: 'API (Beta)',
            path: paths.docs('/rest-api/'),
            popout: true,
          },
        ]}
        show={isShowingOverflow}
        onCancel={handleActionSheetCancel} />
    </nav>
  );
};

export default NavigationTabbar;
