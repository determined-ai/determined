import { Button, Menu, Tooltip } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { CSSTransition } from 'react-transition-group';

import { useStore } from 'contexts/Store';
import useModalUserSettings from 'hooks/useModal/useModalUserSettings';
import useSettings, { BaseType, SettingsConfig } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { ResourceType } from 'types';
import { percent } from 'utils/number';

import Avatar from './Avatar';
import Dropdown, { Placement } from './Dropdown';
import Icon from './Icon';
import JupyterLabModal from './JupyterLabModal';
import Link, { Props as LinkProps } from './Link';
import css from './NavigationSideBar.module.scss';

interface ItemProps extends LinkProps {
  badge?: number;
  icon: string;
  label: string;
  status?: string;
  tooltip?: boolean;
}

interface Settings {
  navbarCollapsed: boolean;
}

const settingsConfig: SettingsConfig = {
  settings: [
    {
      defaultValue: false,
      key: 'navbarCollapsed',
      skipUrlEncoding: true,
      storageKey: 'navbarCollapsed',
      type: { baseType: BaseType.Boolean },
    },
  ],
  storagePath: 'navigation',
};

const menuConfig = {
  bottom: [
    { external: true, icon: 'docs', label: 'Docs', path: paths.docs(), popout: true },
    {
      external: true,
      icon: 'cloud',
      label: 'API (Beta)',
      path: paths.docs('/rest-api/'),
      popout: true,
    },
  ],
  top: [
    { icon: 'dashboard', label: 'Dashboard', path: paths.dashboard() },
    { icon: 'experiment', label: 'Experiments', path: paths.experimentList() },
    { icon: 'model', label: 'Model Registry', path: paths.modelList() },
    { icon: 'tasks', label: 'Tasks', path: paths.taskList() },
    { icon: 'cluster', label: 'Cluster', path: paths.cluster() },
    { icon: 'queue', label: 'Job Queue', path: paths.jobs() },
    { icon: 'logs', label: 'Cluster Logs', path: paths.clusterLogs() },
  ],
};

const NavigationItem: React.FC<ItemProps> = ({ path, status, ...props }: ItemProps) => {
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

  return props.tooltip ? (
    <Tooltip placement="right" title={props.label}><div>{link}</div></Tooltip>
  ) : link;
};

const NavigationSideBar: React.FC = () => {
  // `nodeRef` padding is required for CSSTransition to work with React.StrictMode.
  const nodeRef = useRef(null);
  const { auth, cluster: overview, ui, resourcePools } = useStore();
  const [ showJupyterLabModal, setShowJupyterLabModal ] = useState(false);
  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);
  const { modalOpen: openUserSettingsModal } = useModalUserSettings();

  const showNavigation = auth.isAuthenticated && ui.showChrome;
  const version = process.env.VERSION || '';
  const shortVersion = version.replace(/^(\d+\.\d+\.\d+).*?$/i, '$1');
  const isVersionLong = version !== shortVersion;
  const username = auth.user?.username || 'Anonymous';

  const cluster = useMemo(() => {
    if (overview[ResourceType.ALL].allocation === 0) return undefined;
    const totalSlots = resourcePools.reduce((totalSlots, currentPool) => {
      return totalSlots + currentPool.maxAgents * (currentPool.slotsPerAgent ?? 0);
    }, 0);
    if (totalSlots === 0) return `${overview[ResourceType.ALL].allocation}%`;
    return `${percent((overview[ResourceType.ALL].total - overview[ResourceType.ALL].available)
      / totalSlots)}%`;
  }, [ overview, resourcePools ]);

  const handleCollapse = useCallback(() => {
    updateSettings({ navbarCollapsed: !settings.navbarCollapsed });
  }, [ settings.navbarCollapsed, updateSettings ]);

  if (!showNavigation) return null;

  return (
    <CSSTransition
      appear={true}
      classNames={{
        appear: css.collapsedAppear,
        appearActive: settings.navbarCollapsed ? css.collapsedEnterActive : css.collapsedExitActive,
        appearDone: settings.navbarCollapsed ? css.collapsedEnterDone : css.collapsedExitDone,
        enter: css.collapsedEnter,
        enterActive: css.collapsedEnterActive,
        enterDone: css.collapsedEnterDone,
        exit: css.collapsedExit,
        exitActive: css.collapsedExitActive,
        exitDone: css.collapsedExitDone,
      }}
      in={settings.navbarCollapsed}
      nodeRef={nodeRef}
      timeout={200}>
      <nav className={css.base} ref={nodeRef}>
        <header>
          <Dropdown
            content={(
              <Menu>
                <Menu.Item key="settings" onClick={() => openUserSettingsModal()}>
                  Settings
                </Menu.Item>
                <Menu.Item key="sign-out">
                  <Link path={paths.logout()}>Sign Out</Link>
                </Menu.Item>
              </Menu>
            )}
            offset={settings.navbarCollapsed ? { x: -8, y: 0 } : { x: 16, y: -8 }}
            placement={settings.navbarCollapsed ? Placement.Right : Placement.BottomLeft}>
            <div className={css.user}>
              <Avatar hideTooltip name={username} />
              <span>{username}</span>
            </div>
          </Dropdown>
        </header>
        <main>
          <section className={css.launch}>
            <div className={css.launchBlock}>
              <Button
                className={css.launchButton}
                onClick={() => setShowJupyterLabModal(true)}>Launch JupyterLab
              </Button>
              {settings.navbarCollapsed ? (
                <Button className={css.launchIcon} onClick={() => setShowJupyterLabModal(true)}>
                  <Icon
                    name={'add-small'}
                    size="tiny"
                  />
                </Button>
              ) : null}
            </div>
            <JupyterLabModal
              visible={showJupyterLabModal}
              onCancel={() => setShowJupyterLabModal(false)}
              onLaunch={() => setShowJupyterLabModal(false)}
            />
          </section>
          <section className={css.top}>
            {menuConfig.top.map(config => (
              <NavigationItem
                key={config.icon}
                status={config.icon === 'cluster' ? cluster : undefined}
                tooltip={settings.navbarCollapsed}
                {...config}
              />
            ))}
          </section>
          <section className={css.bottom}>
            {menuConfig.bottom.map(config => (
              <NavigationItem
                key={config.icon}
                tooltip={settings.navbarCollapsed}
                {...config}
              />
            ))}
            <NavigationItem
              icon={settings.navbarCollapsed ? 'expand' : 'collapse'}
              label={settings.navbarCollapsed ? 'Expand' : 'Collapse'}
              tooltip={settings.navbarCollapsed}
              onClick={handleCollapse}
            />
          </section>
        </main>
        <footer>
          <div className={css.version}>
            {isVersionLong && settings.navbarCollapsed ? (
              <Tooltip placement="right" title={`Version ${version}`}>
                <span className={css.versionLabel}>{shortVersion}</span>
              </Tooltip>
            ) : (
              <span className={css.versionLabel}>{version}</span>
            )}
          </div>
        </footer>
      </nav>
    </CSSTransition>
  );
};

export default NavigationSideBar;
