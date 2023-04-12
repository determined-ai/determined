import React, { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import ActionSheet, { ActionItem } from 'components/ActionSheet';
import DynamicIcon from 'components/DynamicIcon';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Link, { Props as LinkProps } from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { handlePath, paths } from 'routes/utils';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import { AnyMouseEvent, routeToReactUrl } from 'shared/utils/routes';
import authStore from 'stores/auth';
import clusterStore from 'stores/cluster';
import determinedStore, { BrandingType } from 'stores/determinedInfo';
import userStore from 'stores/users';
import workspaceStore from 'stores/workspaces';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './NavigationTabbar.module.scss';
import UserBadge from './UserBadge';
import WorkspaceCreateModalComponent from './WorkspaceCreateModal';

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
  const isAuthenticated = useObservable(authStore.isAuthenticated);
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));

  const clusterStatus = useObservable(clusterStore.clusterStatus);

  const info = useObservable(determinedStore.info);
  const loadablePinnedWorkspaces = useObservable(workspaceStore.pinned);
  const pinnedWorkspaces = Loadable.getOrElse([], loadablePinnedWorkspaces);

  const { ui } = useUI();

  const [isShowingOverflow, setIsShowingOverflow] = useState(false);
  const [isShowingPinnedWorkspaces, setIsShowingPinnedWorkspaces] = useState(false);

  const showNavigation = isAuthenticated && ui.showChrome;

  const { canCreateWorkspace } = usePermissions();

  const WorkspaceCreateModal = useModal(WorkspaceCreateModalComponent);

  const handleOverflowOpen = useCallback(() => setIsShowingOverflow(true), []);
  const handleWorkspacesOpen = useCallback(() => {
    if (pinnedWorkspaces.length === 0) {
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

  const workspaceActions = Loadable.quickMatch(
    loadablePinnedWorkspaces,
    [{ icon: <Spinner />, label: 'Loading...' }],
    (workspaces) =>
      workspaces.map(
        (workspace) =>
          ({
            icon: <DynamicIcon name={workspace.name} size={24} style={{ color: 'black' }} />,
            label: workspace.name,
            onClick: (e: AnyMouseEvent) =>
              handlePathUpdate(e, paths.workspaceDetails(workspace.id)),
          } as ActionItem),
      ),
  );

  if (canCreateWorkspace) {
    workspaceActions.push({
      icon: <Icon name="add-small" size="large" />,
      label: 'New Workspace',
      onClick: WorkspaceCreateModal.open,
    });
  }

  const overflowActionsTop = [
    {
      render: () => (
        <div className={css.user}>
          <UserBadge compact key="avatar" user={currentUser} />
        </div>
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
          ...workspaceActions,
        ]}
        show={isShowingPinnedWorkspaces}
        onCancel={handleActionSheetCancel}
      />
      <ActionSheet
        actions={[...overflowActionsTop, ...overflowActionsBottom]}
        show={isShowingOverflow}
        onCancel={handleActionSheetCancel}
      />
      <WorkspaceCreateModal.Component />
    </nav>
  );
};

export default NavigationTabbar;
