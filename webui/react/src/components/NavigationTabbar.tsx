import React, { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import ActionSheet from 'components/ActionSheet';
import DynamicIcon from 'components/DynamicIcon';
import Link, { Props as LinkProps } from 'components/Link';
import AvatarCard from 'components/UserAvatarCard';
import useModalWorkspaceCreate from 'hooks/useModal/Workspace/useModalWorkspaceCreate';
import usePermissions from 'hooks/usePermissions';
import { handlePath, paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import { AnyMouseEvent, routeToReactUrl } from 'shared/utils/routes';
import { selectIsAuthenticated } from 'stores/auth';
import { useClusterStore } from 'stores/cluster';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import usersStore from 'stores/users';
import { useWorkspaces } from 'stores/workspaces';
import { BrandingType } from 'types';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './NavigationTabbar.module.scss';

interface ToolbarItemProps extends LinkProps {
  badge?: number;
  icon: string;
  label: string;
  status?: string;
}

const ToolbarItem: React.FC<ToolbarItemProps> = ({ path, status, ...props }: ToolbarItemProps) => {
  const location = useLocation();
  const classes = [css.toolbarItem];
  const [isActive, setIsActive] = useState(false);

  if (isActive) classes.push(css.active);

  useEffect(() => setIsActive(location.pathname === path), [location.pathname, path]);

  return (
    <Link className={classes.join(' ')} path={path} {...props}>
      <Icon name={props.icon} size="large" />
      {status && <div className={css.status}>{status}</div>}
    </Link>
  );
};

const NavigationTabbar: React.FC = () => {
  const isAuthenticated = useObservable(selectIsAuthenticated);
  const loadableCurrentUser = useObservable(usersStore.getCurrentUser());
  const authUser = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });

  const clusterStatus = useObservable(useClusterStore().clusterStatus);

  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
  const { ui } = useUI();

  const [isShowingOverflow, setIsShowingOverflow] = useState(false);
  const [isShowingPinnedWorkspaces, setIsShowingPinnedWorkspaces] = useState(false);

  const showNavigation = isAuthenticated && ui.showChrome;

  const { canCreateWorkspace } = usePermissions();
  const { contextHolder: modalWorkspaceCreateContextHolder, modalOpen: openWorkspaceCreateModal } =
    useModalWorkspaceCreate();
  const handleCreateWorkspace = useCallback(() => {
    openWorkspaceCreateModal();
  }, [openWorkspaceCreateModal]);

  const pinnedWorkspaces = useWorkspaces({ pinned: true });
  const handleOverflowOpen = useCallback(() => setIsShowingOverflow(true), []);
  const handleWorkspacesOpen = useCallback(() => {
    if (Loadable.getOrElse([], pinnedWorkspaces).length === 0) {
      routeToReactUrl(paths.workspaceList());
      return;
    }
    setIsShowingPinnedWorkspaces(true);
  }, [pinnedWorkspaces]);
  const handleActionSheetCancel = useCallback(() => {
    setIsShowingOverflow(false);
    setIsShowingPinnedWorkspaces(false);
  }, []);

  const handlePathUpdate = useCallback((e: AnyMouseEvent, path?: string) => {
    handlePath(e, { path });
    setIsShowingOverflow(false);
    setIsShowingPinnedWorkspaces(false);
  }, []);

  if (!showNavigation) return null;

  const overflowActionsTop = [
    {
      render: () => (
        <AvatarCard className={css.user} darkLight={ui.darkLight} key="avatar" user={authUser} />
      ),
    },
    {
      icon: 'settings',
      label: 'Settings',
      onClick: (e: AnyMouseEvent) => handlePathUpdate(e, paths.settings('account')),
    },
    {
      icon: 'user',
      label: 'Sign out',
      onClick: (e: AnyMouseEvent) => handlePathUpdate(e, paths.logout()),
    },
  ];

  const overflowActionsBottom = [
    {
      icon: 'logs',
      label: 'Cluster Logs',
      onClick: (e: AnyMouseEvent) => handlePathUpdate(e, paths.clusterLogs()),
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
    {
      external: true,
      icon: 'pencil',
      label: 'Feedback',
      path: paths.submitProductFeedback(info.branding || BrandingType.Determined),
      popout: true,
    },
  ];

  return (
    <nav className={css.base}>
      <div className={css.toolbar}>
        <ToolbarItem icon="home" label="Home" path={paths.dashboard()} />
        <ToolbarItem icon="experiment" label="Uncategorized" path={paths.uncategorized()} />
        <ToolbarItem icon="model" label="Model Registry" path={paths.modelList()} />
        <ToolbarItem icon="tasks" label="Tasks" path={paths.taskList()} />
        <ToolbarItem icon="cluster" label="Cluster" path={paths.cluster()} status={clusterStatus} />
        <ToolbarItem icon="workspaces" label="Workspaces" onClick={handleWorkspacesOpen} />
        <ToolbarItem icon="overflow-vertical" label="Overflow Menu" onClick={handleOverflowOpen} />
      </div>
      <ActionSheet
        actions={[
          {
            icon: 'workspaces',
            label: 'Workspaces',
            onClick: (e: AnyMouseEvent) => handlePathUpdate(e, paths.workspaceList()),
            path: paths.workspaceList(),
          },
          ...Loadable.match(pinnedWorkspaces, {
            Loaded: (workspaces) => {
              const workspaceIcons = workspaces.map((workspace) => ({
                icon: <DynamicIcon name={workspace.name} size={24} style={{ color: 'black' }} />,
                label: workspace.name,
                onClick: (e: AnyMouseEvent) =>
                  handlePathUpdate(e, paths.workspaceDetails(workspace.id)),
              }));
              if (canCreateWorkspace) {
                workspaceIcons.push({
                  icon: <Icon name="add-small" size="large" />,
                  label: 'New Workspace',
                  onClick: handleCreateWorkspace,
                });
              }
              return workspaceIcons;
            },
            NotLoaded: () => [
              {
                icon: <Spinner />,
                label: 'Loading...',
              },
            ],
          }),
        ]}
        show={isShowingPinnedWorkspaces}
        onCancel={handleActionSheetCancel}
      />
      <ActionSheet
        actions={[...overflowActionsTop, ...overflowActionsBottom]}
        show={isShowingOverflow}
        onCancel={handleActionSheetCancel}
      />
      {modalWorkspaceCreateContextHolder}
    </nav>
  );
};

export default NavigationTabbar;
