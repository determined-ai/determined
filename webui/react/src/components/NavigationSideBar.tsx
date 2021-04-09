import { Button, Menu, Tooltip } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { CSSTransition } from 'react-transition-group';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import useStorage from 'hooks/useStorage';
import { paths } from 'routes/utils';
import { ResourceType } from 'types';
import { launchNotebook } from 'utils/task';

import Avatar from './Avatar';
import Dropdown, { Placement } from './Dropdown';
import Icon from './Icon';
import Link, { Props as LinkProps } from './Link';
import css from './NavigationSideBar.module.scss';

interface ItemProps extends LinkProps {
  badge?: number;
  icon: string;
  label: string;
  status?: string;
}

const NavigationItem: React.FC<ItemProps> = ({ path, status, ...props }: ItemProps) => {
  const { ui } = useStore();
  const location = useLocation();
  const [ isActive, setIsActive ] = useState(false);
  const classes = [ css.navItem ];

  if (isActive) classes.push(css.active);
  if (status) classes.push(css.hasStatus);

  useEffect(() => setIsActive(location.pathname === path), [ location.pathname, path ]);

  const link = (
    <Link className={classes.join(' ')} disabled={isActive} path={path} {...props}>
      <Icon name={props.icon} size="large" />
      <div className={css.label}>{props.label}</div>
      {status && <div className={css.status}>{status}</div>}
    </Link>
  );

  return ui.chromeCollapsed ? (
    <Tooltip placement="right" title={props.label}><div>{link}</div></Tooltip>
  ) : link;
};

const STORAGE_KEY = 'collapsed';

const NavigationSideBar: React.FC = () => {
  const { auth, cluster: overview, ui } = useStore();
  const storeDispatch = useStoreDispatch();
  const storage = useStorage('navigation');
  const [ isCollapsed, setIsCollapsed ] = useState(storage.getWithDefault(STORAGE_KEY, false));
  const [ isShowingCpu, setIsShowingCpu ] = useState(false);

  const showNavigation = auth.isAuthenticated && ui.showChrome;
  const version = process.env.VERSION || '';
  const shortVersion = version.split('.').slice(0, 3).join('.');
  const isVersionLong = (version.match(/\./g) || []).length > 2;
  const username = auth.user?.username || 'Anonymous';
  const cluster = overview[ResourceType.ALL].allocation === 0 ?
    undefined : `${overview[ResourceType.ALL].allocation}%`;

  const handleNotebookLaunch = useCallback(() => launchNotebook(1), []);
  const handleCpuNotebookLaunch = useCallback(() => launchNotebook(0), []);
  const handleVisibleChange = useCallback((visible: boolean) => setIsShowingCpu(visible), []);

  const handleCollapse = useCallback(() => {
    const newCollapsed = !isCollapsed;
    storage.set(STORAGE_KEY, newCollapsed);
    setIsCollapsed(newCollapsed);
  }, [ isCollapsed, storage ]);

  useEffect(() => {
    const type = isCollapsed ? StoreAction.CollapseUIChrome : StoreAction.ExpandUIChrome;
    storeDispatch({ type });
  }, [ isCollapsed, storeDispatch ]);

  if (!showNavigation) return null;

  return (
    <CSSTransition
      appear={true}
      classNames={{
        appear: css.collapsedAppear,
        appearActive: isCollapsed ? css.collapsedEnterActive : css.collapsedExitActive,
        appearDone: isCollapsed ? css.collapsedEnterDone : css.collapsedExitDone,
        enter: css.collapsedEnter,
        enterActive: css.collapsedEnterActive,
        enterDone: css.collapsedEnterDone,
        exit: css.collapsedExit,
        exitActive: css.collapsedExitActive,
        exitDone: css.collapsedExitDone,
      }}
      in={isCollapsed}
      timeout={200}>
      <nav className={css.base}>
        <header>
          <div className={css.logo}>
            <div className={css.logoIcon} />
            <div className={css.logoLabel} />
          </div>
          <div className={css.version}>
            {isVersionLong ? (
              <Tooltip placement="right" title={`Version ${version}`}>
                <span className={css.versionLabel}>{shortVersion}</span>
              </Tooltip>
            ) : (
              <span className={css.versionLabel}>{version}</span>
            )}
          </div>
        </header>
        <main>
          <section className={css.launch}>
            <div className={css.launchBlock}>
              <Button
                className={css.launchButton}
                onClick={handleNotebookLaunch}>Launch Notebook</Button>
              <Dropdown
                content={(
                  <Menu>
                    {isCollapsed && <Menu.Item onClick={handleNotebookLaunch}>
                      Launch Notebook
                    </Menu.Item>}
                    <Menu.Item onClick={handleCpuNotebookLaunch}>
                      Launch CPU-only Notebook
                    </Menu.Item>
                  </Menu>
                )}
                offset={isCollapsed ? { x: 8, y: 0 } : { x: 0, y: 8 }}
                placement={isCollapsed ? Placement.RightTop : Placement.BottomRight}
                onVisibleChange={handleVisibleChange}>
                <Button className={css.launchIcon}>
                  <Icon
                    name={isCollapsed ? 'add-small' : (isShowingCpu ? 'arrow-up': 'arrow-down')}
                    size="tiny" />
                </Button>
              </Dropdown>
            </div>
          </section>
          <section className={css.top}>
            <NavigationItem icon="dashboard" label="Dashboard" path={paths.dashboard()} />
            <NavigationItem icon="experiment" label="Experiments" path={paths.experimentList()} />
            <NavigationItem icon="tasks" label="Tasks" path={paths.taskList()} />
            <NavigationItem
              icon="cluster"
              label="Cluster"
              path={paths.cluster()}
              status={cluster} />
            <NavigationItem icon="logs" label="Master Logs" path={paths.masterLogs()} />
          </section>
          <section className={css.bottom}>
            <NavigationItem external icon="docs" label="Docs" path={paths.docs()} popout />
            <NavigationItem
              external
              icon="cloud"
              label="API (Beta)"
              path={paths.docs('/rest-api/')}
              popout />
            <NavigationItem
              icon={isCollapsed ? 'expand' : 'collapse'}
              label={isCollapsed ? 'Expand' : 'Collapse'}
              onClick={handleCollapse} />
          </section>
        </main>
        <footer>
          <Dropdown
            content={<Menu>
              <Menu.Item>
                <Link path={paths.logout()}>Sign Out</Link>
              </Menu.Item>
            </Menu>}
            offset={isCollapsed ? { x: -8, y: 0 } : { x: 16, y: -8 }}
            placement={isCollapsed ? Placement.Right : Placement.TopLeft}>
            <div className={css.user}>
              <Avatar hideTooltip name={username} />
              <span>{username}</span>
            </div>
          </Dropdown>
        </footer>
      </nav>
    </CSSTransition>
  );
};

export default NavigationSideBar;
