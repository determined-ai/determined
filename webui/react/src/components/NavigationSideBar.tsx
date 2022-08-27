import { Button, Menu, Tooltip, Typography } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef,
  useState } from 'react';
import { useLocation } from 'react-router-dom';
import { CSSTransition } from 'react-transition-group';

import Dropdown, { Placement } from 'components/Dropdown';
import DynamicIcon from 'components/DynamicIcon';
import Link, { Props as LinkProps } from 'components/Link';
import AvatarCard from 'components/UserAvatarCard';
import { useStore } from 'contexts/Store';
import useModalJupyterLab from 'hooks/useModal/JupyterLab/useModalJupyterLab';
import useModalWorkspaceCreate from 'hooks/useModal/Workspace/useModalWorkspaceCreate';
import useSettings, { BaseType, SettingsConfig } from 'hooks/useSettings';
import { clusterStatusText } from 'pages/Clusters/ClustersOverview';
import WorkspaceQuickSearch from 'pages/WorkspaceDetails/WorkspaceQuickSearch';
import WorkspaceActionDropdown from 'pages/WorkspaceList/WorkspaceActionDropdown';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import { BrandingType } from 'types';

import css from './NavigationSideBar.module.scss';
import ThemeToggle from './ThemeToggle';

interface ItemProps extends LinkProps {
  action?: React.ReactNode;
  badge?: number;
  icon: string | React.ReactNode;
  label: string;
  labelRender?: React.ReactNode;
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

const NavigationItem: React.FC<ItemProps> = ({ path, status, action, ...props }: ItemProps) => {
  const location = useLocation();
  const [ isActive, setIsActive ] = useState(false);
  const classes = [ css.navItem ];
  const containerClasses = [ css.navItemContainer ];

  if (isActive) {
    containerClasses.push(css.active);
    classes.push(css.active);
  }
  if (status) containerClasses.push(css.hasStatus);

  useEffect(() => {
    setIsActive(location.pathname === path);
  }, [ location.pathname, path ]);

  const link = (
    <div className={containerClasses.join(' ')}>
      <Link className={classes.join(' ')} path={path} {...props}>
        {typeof props.icon === 'string' ?
          <div className={css.icon}><Icon name={props.icon} size="large" /></div> :
          <div className={css.icon}>{props.icon}</div>}
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
    <Tooltip placement="right" title={props.label}><div>{link}</div></Tooltip>
  ) : link;
};

const NavigationSideBar: React.FC = () => {
  // `nodeRef` padding is required for CSSTransition to work with React.StrictMode.
  const nodeRef = useRef(null);
  const { agents, auth, cluster: overview, ui, resourcePools, info, pinnedWorkspaces } = useStore();
  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);
  const {
    contextHolder: modalJupyterLabContextHolder,
    modalOpen: openJupyterLabModal,
  } = useModalJupyterLab();
  const {
    contextHolder: modalWorkspaceCreateContextHolder,
    modalOpen: openWorkspaceCreateModal,
  } = useModalWorkspaceCreate();
  const showNavigation = auth.isAuthenticated && ui.showChrome;
  const version = process.env.VERSION || '';
  const shortVersion = version.replace(/^(\d+\.\d+\.\d+).*?$/i, '$1');
  const isVersionLong = version !== shortVersion;

  const menuConfig = useMemo(() => ({
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
        label: 'Share Feedback',
        path: paths.submitProductFeedback(info.branding || BrandingType.Determined),
        popout: true,
      },
    ],
    top: [
      { icon: 'experiment', label: 'Uncategorized', path: paths.uncategorized() },
      { icon: 'model', label: 'Model Registry', path: paths.modelList() },
      { icon: 'tasks', label: 'Tasks', path: paths.taskList() },
      { icon: 'cluster', label: 'Cluster', path: paths.cluster() },
    ],
  }), [ info.branding ]);

  const handleCollapse = useCallback(() => {
    updateSettings({ navbarCollapsed: !settings.navbarCollapsed });
  }, [ settings.navbarCollapsed, updateSettings ]);

  const handleCreateWorkspace = useCallback(() => {
    openWorkspaceCreateModal();
  }, [ openWorkspaceCreateModal ]);

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
              <Menu
                items={[
                  { key: 'theme-toggle', label: <ThemeToggle /> },
                  {
                    key: 'settings',
                    label: <Link path={paths.settings('account')}>Settings</Link>,
                  },
                  { key: 'sign-out', label: <Link path={paths.logout()}>Sign Out</Link> },
                ]}
                selectable={false}
              />
            )}
            offset={settings.navbarCollapsed ? { x: -8, y: 16 } : { x: 16, y: -8 }}
            placement={settings.navbarCollapsed ? Placement.RightTop : Placement.BottomLeft}>
            <AvatarCard className={css.user} darkLight={ui.darkLight} user={auth.user} />
          </Dropdown>
        </header>
        <main>
          <section className={css.launch}>
            <div className={css.launchBlock}>
              <Button
                className={css.launchButton}
                onClick={() => openJupyterLabModal()}>Launch JupyterLab
              </Button>
              {settings.navbarCollapsed ? (
                <Button className={css.launchIcon} onClick={() => openJupyterLabModal()}>
                  <Icon name="jupyter-lab" />
                </Button>
              ) : null}
            </div>
          </section>
          <section className={css.top}>
            {menuConfig.top.map((config) => (
              <NavigationItem
                key={config.icon}
                status={config.icon === 'cluster' ?
                  clusterStatusText(overview, resourcePools, agents) : undefined}
                tooltip={settings.navbarCollapsed}
                {...config}
              />
            ))}
          </section>
          <section className={css.workspaces}>
            <NavigationItem
              action={(
                <div className={css.actionButtons}>
                  <WorkspaceQuickSearch>
                    <Button type="text">
                      <Icon name="search" size="tiny" />
                    </Button>
                  </WorkspaceQuickSearch>
                  <Button type="text" onClick={handleCreateWorkspace}>
                    <Icon name="add-small" size="tiny" />
                  </Button>
                </div>
              )}
              icon="workspaces"
              key="workspaces"
              label="Workspaces"
              path={paths.workspaceList()}
              tooltip={settings.navbarCollapsed}
            />
            {pinnedWorkspaces.length === 0 ?
              <p className={css.noWorkspaces}>No pinned workspaces</p> : (
                <ul className={css.pinnedWorkspaces} role="list">
                  {pinnedWorkspaces.map((workspace) => (
                    <WorkspaceActionDropdown
                      curUser={auth.user}
                      key={workspace.id}
                      trigger={[ 'contextMenu' ]}
                      workspace={workspace}>
                      <li>
                        <NavigationItem
                          icon={(
                            <DynamicIcon
                              name={workspace.name}
                              size={24}
                            />
                          )}
                          label={workspace.name}
                          labelRender={(
                            <Typography.Paragraph
                              ellipsis={{ rows: 1, tooltip: true }}>
                              {workspace.name}
                            </Typography.Paragraph>
                          )}
                          path={paths.workspaceDetails(workspace.id)}
                        />
                      </li>
                    </WorkspaceActionDropdown>
                  ))}
                </ul>
              )}
          </section>
          <section className={css.bottom}>
            {menuConfig.bottom.map((config) => (
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
        {modalJupyterLabContextHolder}
        {modalWorkspaceCreateContextHolder}
      </nav>
    </CSSTransition>
  );
};

export default NavigationSideBar;
