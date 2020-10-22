import React, { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { CSSTransition } from 'react-transition-group';

import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import UI from 'contexts/UI';

import Icon from './Icon';
import Link, { Props as LinkProps } from './Link';
import css from './NavigationTabbar.module.scss';

interface ToolbarItemProps extends LinkProps {
  badge?: number;
  icon: string;
  label: string;
  status?: string;
}

interface OverflowItemProps extends LinkProps {
  icon: string;
  label: string;
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

const OverflowItem: React.FC<OverflowItemProps> = ({ path, ...props }: OverflowItemProps) => {
  return (
    <Link className={css.overflowItem} path={path} {...props}>
      <Icon name={props.icon} size="large" />
      <div className={css.label}>{props.label}</div>
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
  const handleOverflowClose = useCallback(() => setIsShowingOverflow(false), []);

  if (!showNavigation) return null;

  return (
    <nav className={css.base}>
      <div className={css.toolbar}>
        <ToolbarItem icon="dashboard" label="Dashboard" path="/det/dashboard" />
        <ToolbarItem icon="experiment" label="Experiments" path="/det/experiments" />
        <ToolbarItem icon="tasks" label="Tasks" path="/det/tasks" />
        <ToolbarItem icon="cluster" label="Cluster" path="/det/cluster" status={cluster} />
        <ToolbarItem icon="overflow-vertical" label="Overflow Menu" onClick={handleOverflowOpen} />
      </div>
      <CSSTransition
        classNames={{
          enter: css.overflowEnter,
          enterActive: css.overflowEnterActive,
          enterDone: css.overflowEnterDone,
          exit: css.overflowExit,
          exitActive: css.overflowExitActive,
          exitDone: css.overflowExitDone,
        }}
        in={isShowingOverflow}
        timeout={200}>
        <div className={css.overflow} onClick={handleOverflowClose}>
          <div className={css.overflowMenu}>
            <OverflowItem icon="logs" label="Master Logs" path="/det/logs" popout />
            <OverflowItem icon="docs" label="Docs" path="/docs" popout />
            <OverflowItem
              icon="cloud"
              label="API (Beta)"
              noProxy
              path="/docs/rest-api/"
              popout />
            <OverflowItem icon="error" label="Cancel" />
          </div>
        </div>
      </CSSTransition>
    </nav>
  );
};

export default NavigationTabbar;
