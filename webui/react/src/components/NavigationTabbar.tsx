import React, { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import UI from 'contexts/UI';
import { handlePath } from 'routes/utils';
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
  const { isAuthenticated } = Auth.useStateContext();
  const ui = UI.useStateContext();
  const overview = ClusterOverview.useStateContext();
  const [ isShowingOverflow, setIsShowingOverflow ] = useState(false);

  const cluster = overview.allocation === 0 ? undefined : `${overview.allocation}%`;
  const showNavigation = isAuthenticated && ui.showChrome;

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
        <ToolbarItem icon="dashboard" label="Dashboard" path="/dashboard" />
        <ToolbarItem icon="experiment" label="Experiments" path="/experiments" />
        <ToolbarItem icon="tasks" label="Tasks" path="/tasks" />
        <ToolbarItem icon="cluster" label="Cluster" path="/cluster" status={cluster} />
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
            onClick: e => handlePathUpdate(e, '/logs'),
          },
          {
            external: true,
            icon: 'docs',
            label: 'Docs',
            path: '/docs',
            popout: true,
          },
          {
            external: true,
            icon: 'cloud',
            label: 'API (Beta)',
            path: '/docs/rest-api/',
            popout: true,
          },
        ]}
        show={isShowingOverflow}
        onCancel={handleActionSheetCancel} />
    </nav>
  );
};

export default NavigationTabbar;
