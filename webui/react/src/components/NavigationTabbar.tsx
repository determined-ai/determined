import { Modal } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';

import { useStore } from 'contexts/Store';
import useModalUserSettings from 'hooks/useModal/UserSettings/useModalUserSettings';
import { handlePath, paths } from 'routes/utils';
import { ResourceType, Workspace } from 'types';
import { percent } from 'utils/number';

import ActionSheet from './ActionSheet';
import AvatarCard from './AvatarCard';
import Icon from './Icon';
import JupyterLabModal from './JupyterLabModal';
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
  const { auth, cluster: overview, ui, resourcePools } = useStore();
  const [ isShowingOverflow, setIsShowingOverflow ] = useState(false);
  const [ isShowingPinnedWorkspaces, setIsShowingPinnedWorkspaces ] = useState(false);
  const [ showJupyterLabModal, setShowJupyterLabModal ] = useState(false);
  const [ modal, contextHolder ] = Modal.useModal();
  const { modalOpen: openUserSettingsModal } = useModalUserSettings(modal);
  const [ pinnedWorkspaces, setPinnedWorkspaces ] = useState<Workspace[]>([]);

  const cluster = useMemo(() => {
    if (overview[ResourceType.ALL].allocation === 0) return undefined;
    const totalSlots = resourcePools.reduce((totalSlots, currentPool) => {
      return totalSlots + currentPool.maxAgents * (currentPool.slotsPerAgent ?? 0);
    }, 0);
    if (totalSlots === 0) return `${overview[ResourceType.ALL].allocation}%`;
    return `${percent((overview[ResourceType.ALL].total - overview[ResourceType.ALL].available)
        / totalSlots)}%`;
  }, [ overview, resourcePools ]);

  const showNavigation = auth.isAuthenticated && ui.showChrome;

  const handleOverflowOpen = useCallback(() => setIsShowingOverflow(true), []);
  const handleWorkspacesOpen = useCallback(() => {
    if (pinnedWorkspaces.length === 0) return;
    setIsShowingPinnedWorkspaces(true);
  }, [ pinnedWorkspaces.length ]);
  const handleActionSheetCancel = useCallback(() => {
    setIsShowingOverflow(false);
    setIsShowingPinnedWorkspaces(false);
  }, []);
  const handleLaunchJupyterLab = useCallback(() => {
    setShowJupyterLabModal(true);
    setIsShowingOverflow(false);
  }, []);

  const handlePathUpdate = useCallback((e, path) => {
    handlePath(e, { path });
    setIsShowingOverflow(false);
    setIsShowingPinnedWorkspaces(false);
  }, []);

  if (!showNavigation) return null;

  return (
    <nav className={css.base}>
      {contextHolder}
      <div className={css.toolbar}>
        <ToolbarItem icon="experiment" label="Uncategorized" path={paths.projectDetails(1)} />
        <ToolbarItem icon="model" label="Model Registry" path={paths.modelList()} />
        <ToolbarItem icon="tasks" label="Tasks" path={paths.taskList()} />
        <ToolbarItem icon="cluster" label="Cluster" path={paths.cluster()} status={cluster} />
        <ToolbarItem
          icon="workspaces"
          label="Workspaces"
          path={pinnedWorkspaces.length === 0 ? paths.workspaceList() : undefined}
          onClick={handleWorkspacesOpen}
        />
        <ToolbarItem icon="overflow-vertical" label="Overflow Menu" onClick={handleOverflowOpen} />
      </div>
      <ActionSheet
        actions={[
          {
            render: () => {
              return <AvatarCard className={css.user} user={auth.user} />;
            },
          },
          {
            label: 'Settings',
            onClick: () => openUserSettingsModal(),
          },
          {
            label: 'Sign out',
            onClick: e => handlePathUpdate(e, paths.logout()),
          },
          {
            icon: 'jupyter-lab',
            label: 'Launch JupyterLab',
            onClick: () => handleLaunchJupyterLab(),
          },
          {
            icon: 'logs',
            label: 'Cluster Logs',
            onClick: e => handlePathUpdate(e, paths.clusterLogs()),
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
        onCancel={handleActionSheetCancel}
      />
      <ActionSheet
        actions={[
          {
            render: () => {
              return <Link path={paths.workspaceList()} />;
            },
          },
          ...pinnedWorkspaces.map(workspace => (
            {
              render: () => {
                return <Link path={paths.workspaceDetails(workspace.id)}>{workspace.name}</Link>;
              },
            })),
        ]}
        show={isShowingPinnedWorkspaces}
        onCancel={handleActionSheetCancel}
      />
      <JupyterLabModal
        visible={showJupyterLabModal}
        onCancel={() => setShowJupyterLabModal(false)}
      />
    </nav>
  );
};

export default NavigationTabbar;
