import React, { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import ActionSheet from 'components/ActionSheet';
import DynamicIcon from 'components/DynamicIcon';
import Link, { Props as LinkProps } from 'components/Link';
import AvatarCard from 'components/UserAvatarCard';
import { useStore } from 'contexts/Store';
import useModalJupyterLab from 'hooks/useModal/JupyterLab/useModalJupyterLab';
import { clusterStatusText } from 'pages/Clusters/ClustersOverview';
import { handlePath, paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import useUI from 'shared/contexts/stores/UI';
import { AnyMouseEvent, routeToReactUrl } from 'shared/utils/routes';
import { useAgents, useClusterOverview } from 'stores/agents';
import { useResourcePools } from 'stores/resourcePools';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { BrandingType } from 'types';
import { Loadable } from 'utils/loadable';

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
  const { auth, pinnedWorkspaces } = useStore();
  const loadableResourcePools = useResourcePools();
  const resourcePools = Loadable.getOrElse([], loadableResourcePools); // TODO show spinner when this is loading
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
  const { ui } = useUI();
  const overview = useClusterOverview();
  const agents = useAgents();
  const clusterStatus = Loadable.match(Loadable.all([agents, overview]), {
    Loaded: ([agents, overview]) => clusterStatusText(overview, resourcePools, agents),
    NotLoaded: () => undefined, // TODO show spinner when this is loading
  });
  const [isShowingOverflow, setIsShowingOverflow] = useState(false);
  const [isShowingPinnedWorkspaces, setIsShowingPinnedWorkspaces] = useState(false);
  const { contextHolder: modalJupyterLabContextHolder, modalOpen: openJupyterLabModal } =
    useModalJupyterLab();

  const showNavigation = auth.isAuthenticated && ui.showChrome;

  const handleOverflowOpen = useCallback(() => setIsShowingOverflow(true), []);
  const handleWorkspacesOpen = useCallback(() => {
    if (pinnedWorkspaces.length === 0) {
      routeToReactUrl(paths.workspaceList());
      return;
    }
    setIsShowingPinnedWorkspaces(true);
  }, [pinnedWorkspaces.length]);
  const handleActionSheetCancel = useCallback(() => {
    setIsShowingOverflow(false);
    setIsShowingPinnedWorkspaces(false);
  }, []);
  const handleLaunchJupyterLab = useCallback(() => {
    setIsShowingOverflow(false);
    openJupyterLabModal();
  }, [openJupyterLabModal]);

  const handlePathUpdate = useCallback((e: AnyMouseEvent, path?: string) => {
    handlePath(e, { path });
    setIsShowingOverflow(false);
    setIsShowingPinnedWorkspaces(false);
  }, []);

  if (!showNavigation) return null;

  return (
    <nav className={css.base}>
      <div className={css.toolbar}>
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
          ...pinnedWorkspaces.map((workspace) => ({
            icon: <DynamicIcon name={workspace.name} size={24} style={{ color: 'black' }} />,
            label: workspace.name,
            onClick: (e: AnyMouseEvent) =>
              handlePathUpdate(e, paths.workspaceDetails(workspace.id)),
          })),
        ]}
        show={isShowingPinnedWorkspaces}
        onCancel={handleActionSheetCancel}
      />
      <ActionSheet
        actions={[
          {
            render: () => (
              <AvatarCard
                className={css.user}
                darkLight={ui.darkLight}
                key="avatar"
                user={auth.user}
              />
            ),
          },
          {
            icon: 'settings',
            label: 'Settings',
            onClick: (e) => handlePathUpdate(e, paths.settings('account')),
          },
          {
            icon: 'user',
            label: 'Sign out',
            onClick: (e) => handlePathUpdate(e, paths.logout()),
          },
          {
            icon: 'jupyter-lab',
            label: 'Launch JupyterLab',
            onClick: () => handleLaunchJupyterLab(),
          },
          {
            icon: 'logs',
            label: 'Cluster Logs',
            onClick: (e) => handlePathUpdate(e, paths.clusterLogs()),
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
            label: 'Share Feedback',
            path: paths.submitProductFeedback(info.branding || BrandingType.Determined),
            popout: true,
          },
        ]}
        show={isShowingOverflow}
        onCancel={handleActionSheetCancel}
      />
      {modalJupyterLabContextHolder}
    </nav>
  );
};

export default NavigationTabbar;
