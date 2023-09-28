import { Typography } from 'antd';
import { boolean } from 'io-ts';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { CSSTransition } from 'react-transition-group';

import DynamicIcon from 'components/DynamicIcon';
import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon, { IconName, IconSize } from 'components/kit/Icon';
import { matchesShortcut, shortcutToString } from 'components/kit/InputShortcut';
import { useModal } from 'components/kit/Modal';
import Spinner from 'components/kit/Spinner';
import useUI from 'components/kit/Theme';
import Tooltip from 'components/kit/Tooltip';
import { Loadable } from 'components/kit/utils/loadable';
import Link, { Props as LinkProps } from 'components/Link';
import UserSettings from 'components/UserSettings';
import shortCutSettingsConfig, {
  Settings as ShortcutSettings,
} from 'components/UserSettings.settings';
import { keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import usePermissions from 'hooks/usePermissions';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import WorkspaceQuickSearch from 'pages/WorkspaceDetails/WorkspaceQuickSearch';
import WorkspaceActionDropdown from 'pages/WorkspaceList/WorkspaceActionDropdown';
import { paths } from 'routes/utils';
import authStore from 'stores/auth';
import clusterStore from 'stores/cluster';
import determinedStore, { BrandingType } from 'stores/determinedInfo';
import userStore from 'stores/users';
import workspaceStore from 'stores/workspaces';
import { useObservable } from 'utils/observable';

import css from './NavigationSideBar.module.scss';
import ThemeToggle from './ThemeToggle';
import UserBadge from './UserBadge';
import WorkspaceCreateModalComponent from './WorkspaceCreateModal';

interface ItemProps extends LinkProps {
  action?: React.ReactNode;
  badge?: number;
  icon: IconName | React.ReactElement;
  iconSize?: IconSize;
  label: string;
  labelRender?: React.ReactNode;
  status?: string;
  tooltip?: string | boolean;
}

interface Settings {
  navbarCollapsed: boolean;
}

const settingsConfig: SettingsConfig<Settings> = {
  settings: {
    navbarCollapsed: {
      defaultValue: false,
      skipUrlEncoding: true,
      storageKey: 'navbarCollapsed',
      type: boolean,
    },
  },
  storagePath: 'navigation',
};

export const NavigationItem: React.FC<ItemProps> = ({
  path,
  status,
  action,
  ...props
}: ItemProps) => {
  const location = useLocation();
  const [isActive, setIsActive] = useState(false);
  const classes = [css.navItem];
  const containerClasses = [css.navItemContainer];

  if (isActive) {
    containerClasses.push(css.active);
    classes.push(css.active);
  }
  if (status) containerClasses.push(css.hasStatus);

  useEffect(() => {
    setIsActive(location.pathname === path);
  }, [location.pathname, path]);

  const link = (
    <div className={containerClasses.join(' ')}>
      <Link className={classes.join(' ')} path={path} {...props}>
        {typeof props.icon === 'string' ? (
          <div className={css.icon}>
            <Icon decorative name={props.icon} size={props.iconSize ?? 'large'} />
          </div>
        ) : (
          <div className={css.icon}>{props.icon}</div>
        )}
        <div className={css.label}>{props.labelRender ? props.labelRender : props.label}</div>
      </Link>
      <div className={css.navItemExtra}>
        {status && (
          <Link path={path} {...props}>
            <div className={css.status}>{status}</div>
          </Link>
        )}
        {action && <div className={css.action}>{action}</div>}
      </div>
    </div>
  );

  return props.tooltip ? (
    <Tooltip
      content={typeof props.tooltip === 'string' ? props.tooltip : props.label}
      placement="right">
      <div>{link}</div>
    </Tooltip>
  ) : (
    link
  );
};

const NavigationSideBar: React.FC = () => {
  // `nodeRef` padding is required for CSSTransition to work with React.StrictMode.
  const nodeRef = useRef(null);

  const [showSettings, setShowSettings] = useState<boolean>(false);

  const clusterStatus = useObservable(clusterStore.clusterStatus);

  const isAuthenticated = useObservable(authStore.isAuthenticated);
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));

  const info = useObservable(determinedStore.info);
  const { ui } = useUI();

  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);
  const {
    settings: { navbarCollapsed: navbarCollapsedShortcut },
  } = useSettings<ShortcutSettings>(shortCutSettingsConfig);

  const WorkspaceCreateModal = useModal(WorkspaceCreateModalComponent);

  const showNavigation = isAuthenticated && ui.showChrome;
  const version = process.env.VERSION || '';
  const shortVersion = version.replace(/^(\d+\.\d+\.\d+).*?$/i, '$1');
  const isVersionLong = version !== shortVersion;

  const { canCreateWorkspace, canViewWorkspace, canEditWebhooks } = usePermissions();

  const canAccessUncategorized = canViewWorkspace({ workspace: { id: 1 } });

  const pinnedWorkspaces = useObservable(workspaceStore.pinned);

  interface MenuItemProps {
    icon: IconName;
    label: string;
    path: string;
    external?: boolean;
    popout?: boolean;
  }

  const menuConfig: { bottom: MenuItemProps[]; top: MenuItemProps[] } = useMemo(() => {
    const topNav: MenuItemProps[] = canAccessUncategorized
      ? [{ icon: 'experiment', label: 'Uncategorized', path: paths.uncategorized() }]
      : [];
    const dashboardTopNav: MenuItemProps[] = [
      { icon: 'home', label: 'Home', path: paths.dashboard() },
    ];
    const topItems: MenuItemProps[] = [
      ...dashboardTopNav.concat(topNav),
      { icon: 'model', label: 'Model Registry', path: paths.modelList() },
      { icon: 'tasks', label: 'Tasks', path: paths.taskList() },
      { icon: 'cluster', label: 'Cluster', path: paths.clusters() },
    ];
    if (canEditWebhooks) {
      topItems.splice(topItems.length - 1, 0, {
        icon: 'webhooks',
        label: 'Webhooks',
        path: paths.webhooks(),
      });
    }
    return {
      bottom: [
        { external: true, icon: 'docs', label: 'Docs', path: paths.docs(), popout: true },
        {
          external: true,
          icon: 'cloud',
          label: 'API (Beta)',
          path: paths.docs('/rest-api/'),
          popout: true,
        },
        {
          external: true,
          icon: 'pencil',
          label: 'Feedback',
          path: paths.submitProductFeedback(info.branding || BrandingType.Determined),
          popout: true,
        },
      ],
      top: topItems,
    };
  }, [canAccessUncategorized, canEditWebhooks, info.branding]);

  const handleCollapse = useCallback(() => {
    updateSettings({ navbarCollapsed: !settings.navbarCollapsed });
  }, [settings.navbarCollapsed, updateSettings]);

  useEffect(() => {
    const keyDownListener = (e: KeyboardEvent) => {
      if (matchesShortcut(e, navbarCollapsedShortcut)) {
        handleCollapse();
      }
    };

    keyEmitter.on(KeyEvent.KeyDown, keyDownListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyDown, keyDownListener);
    };
  }, [handleCollapse, navbarCollapsedShortcut]);

  const { canAdministrateUsers } = usePermissions();

  const menuItems = useMemo(() => {
    const items: MenuItem[] = [
      {
        key: 'settings',
        label: <Link onClick={() => setShowSettings(true)}>Settings</Link>,
      },
      { key: 'theme-toggle', label: <ThemeToggle /> },
      { key: 'sign-out', label: <Link path={paths.logout()}>Sign Out</Link> },
    ];
    if (canAdministrateUsers) {
      items.unshift({
        key: 'admin',
        label: <Link path={paths.admin()}>Admin</Link>,
      });
    }
    return items;
  }, [canAdministrateUsers]);

  if (!showNavigation) return null;

  return (
    <>
      <CSSTransition
        appear
        classNames={{
          appear: css.collapsedAppear,
          appearActive: settings.navbarCollapsed
            ? css.collapsedEnterActive
            : css.collapsedExitActive,
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
            <Dropdown menu={menuItems}>
              <div className={css.user}>
                <UserBadge compact hideAvatarTooltip user={currentUser} />
              </div>
            </Dropdown>
          </header>
          <section>
            <section className={css.top}>
              {menuConfig.top.map((config) => (
                <NavigationItem
                  key={config.icon}
                  status={config.icon === 'cluster' ? clusterStatus : undefined}
                  tooltip={settings.navbarCollapsed}
                  {...config}
                />
              ))}
            </section>
            <section className={css.workspaces}>
              <NavigationItem
                action={
                  <div className={css.actionButtons}>
                    <WorkspaceQuickSearch>
                      <Button
                        icon={<Icon name="search" size="tiny" title="Search workspaces" />}
                        type="text"
                      />
                    </WorkspaceQuickSearch>
                    {canCreateWorkspace && (
                      <Button
                        icon={<Icon name="add-small" size="tiny" title="Create workspace" />}
                        type="text"
                        onClick={WorkspaceCreateModal.open}
                      />
                    )}
                  </div>
                }
                icon="workspaces"
                key="workspaces"
                label="Workspaces"
                path={paths.workspaceList()}
                tooltip={settings.navbarCollapsed}
              />
              {Loadable.match(pinnedWorkspaces, {
                Loaded: (workspaces) => (
                  <ul className={css.pinnedWorkspaces} role="list">
                    {workspaces
                      .sort((a, b) => ((a.pinnedAt ?? 0) < (b.pinnedAt ?? 0) ? -1 : 1))
                      .map((workspace) => (
                        <WorkspaceActionDropdown
                          isContextMenu
                          key={workspace.id}
                          returnIndexOnDelete={false}
                          workspace={workspace}>
                          <li>
                            <NavigationItem
                              icon={<DynamicIcon name={workspace.name} size={24} />}
                              label={workspace.name}
                              labelRender={
                                <Typography.Paragraph ellipsis={{ rows: 1, tooltip: true }}>
                                  {workspace.name}
                                </Typography.Paragraph>
                              }
                              path={paths.workspaceDetails(workspace.id)}
                            />
                          </li>
                        </WorkspaceActionDropdown>
                      ))}
                    {workspaces.length === 0 && (
                      <div className={css.noWorkspaces}>No pinned workspaces</div>
                    )}
                    {canCreateWorkspace && (
                      <li>
                        <NavigationItem
                          icon="add-small"
                          iconSize="tiny"
                          label="New Workspace"
                          labelRender={
                            <Typography.Paragraph ellipsis={{ rows: 1, tooltip: true }}>
                              New Workspace
                            </Typography.Paragraph>
                          }
                          tooltip={settings.navbarCollapsed}
                          onClick={WorkspaceCreateModal.open}
                        />
                      </li>
                    )}
                  </ul>
                ),
                NotLoaded: () => <Spinner center spinning />,
              })}
            </section>
            <section className={css.bottom}>
              {menuConfig.bottom.map((config) => (
                <NavigationItem key={config.icon} tooltip={settings.navbarCollapsed} {...config} />
              ))}
              <NavigationItem
                icon={settings.navbarCollapsed ? 'expand' : 'collapse'}
                label={settings.navbarCollapsed ? 'Expand' : 'Collapse'}
                tooltip={
                  settings.navbarCollapsed
                    ? `Expand (${shortcutToString(navbarCollapsedShortcut)})`
                    : `Collapse (${shortcutToString(navbarCollapsedShortcut)})`
                }
                onClick={handleCollapse}
              />
            </section>
          </section>
          <footer>
            <div className={css.version}>
              {isVersionLong && settings.navbarCollapsed ? (
                <Tooltip content={`Version ${version}`} placement="right">
                  <span className={css.versionLabel}>{shortVersion}</span>
                </Tooltip>
              ) : (
                <span className={css.versionLabel}>{version}</span>
              )}
            </div>
          </footer>
          <WorkspaceCreateModal.Component />
        </nav>
      </CSSTransition>
      <UserSettings show={showSettings} onClose={() => setShowSettings(false)} />
    </>
  );
};

export default NavigationSideBar;
