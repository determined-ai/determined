import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import React, { useCallback, useMemo, useState } from 'react';

import css from 'components/ActionDropdown/ActionDropdown.module.scss';
import WorkspaceCreateModalComponent from 'components/WorkspaceCreateModal';
import WorkspaceDeleteModalComponent from 'components/WorkspaceDeleteModal';
import usePermissions from 'hooks/usePermissions';
import clusterStore from 'stores/cluster';
import workspaceStore from 'stores/workspaces';
import { Workspace } from 'types';
import handleError from 'utils/error';

interface Props {
  children?: React.ReactNode;
  className?: string;
  direction?: 'vertical' | 'horizontal';
  isContextMenu?: boolean;
  onComplete?: () => void;
  returnIndexOnDelete?: boolean;
  workspace: Workspace;
}

interface WorkspaceMenuPropsIn {
  onComplete?: () => void;
  returnIndexOnDelete?: boolean;
  workspace?: Workspace;
}

interface WorkspaceMenuPropsOut {
  contextHolders: React.ReactElement;
  menu: MenuItem[];
  onClick: (key: string) => void;
}

export const useWorkspaceActionMenu: (props: WorkspaceMenuPropsIn) => WorkspaceMenuPropsOut = ({
  onComplete,
  returnIndexOnDelete = true,
  workspace,
}: WorkspaceMenuPropsIn) => {
  const WorkspaceDeleteModal = useModal(WorkspaceDeleteModalComponent);
  const WorkspaceEditModal = useModal(WorkspaceCreateModalComponent);
  const [editingWorkspaceId, setEditingWorkspaceId] = useState<number | undefined>(undefined);

  const openWorkspaceEditModal = () => {
    clusterStore.fetchKubernetesResourceManagers();
    setEditingWorkspaceId(workspace?.id);
    WorkspaceEditModal.open();
  };

  const contextHolders = useMemo(() => {
    return (
      <>
        {workspace && (
          <>
            <WorkspaceDeleteModal.Component
              returnIndexOnDelete={returnIndexOnDelete}
              workspace={workspace}
              onClose={onComplete}
            />
            <WorkspaceEditModal.Component
              workspaceId={editingWorkspaceId}
              onClose={() => {
                onComplete?.();
                setEditingWorkspaceId(undefined);
              }}
            />
          </>
        )}
      </>
    );
  }, [
    workspace,
    WorkspaceDeleteModal,
    returnIndexOnDelete,
    onComplete,
    WorkspaceEditModal,
    editingWorkspaceId,
  ]);

  const { canDeleteWorkspace, canModifyWorkspace } = usePermissions();

  const handleArchiveClick = useCallback(() => {
    if (!workspace) return;
    if (workspace.archived) {
      workspaceStore
        .unarchiveWorkspace(workspace.id)
        .then(() => onComplete?.())
        .catch((e) => handleError(e, { publicSubject: 'Unable to unarchive workspace.' }));
    } else {
      workspaceStore
        .archiveWorkspace(workspace.id)
        .then(() => onComplete?.())
        .catch((e) => handleError(e, { publicSubject: 'Unable to archive workspace.' }));
    }
  }, [onComplete, workspace]);

  const handlePinClick = useCallback(() => {
    if (!workspace) return;

    if (workspace.pinned) {
      workspaceStore
        .unpinWorkspace(workspace.id)
        .then(() => onComplete?.())
        .catch((e) => handleError(e, { publicSubject: 'Unable to unpin workspace.' }));
    } else {
      workspaceStore
        .pinWorkspace(workspace.id)
        .then(() => onComplete?.())
        .catch((e) => handleError(e, { publicSubject: 'Unable to pin workspace.' }));
    }
  }, [onComplete, workspace]);

  const MenuKey = {
    Delete: 'delete',
    Edit: 'edit',
    SwitchArchived: 'switchArchive',
    SwitchPin: 'switchPin',
  } as const;

  const handleDropdown = (key: string) => {
    switch (key) {
      case MenuKey.Edit:
        openWorkspaceEditModal();
        break;
      case MenuKey.Delete:
        WorkspaceDeleteModal.open();
        break;
      case MenuKey.SwitchArchived:
        handleArchiveClick();
        break;
      case MenuKey.SwitchPin:
        handlePinClick();
        break;
    }
  };

  const menuItems: MenuItem[] = [];

  if (workspace && !workspace.immutable) {
    menuItems.push({
      key: MenuKey.SwitchPin,
      label: workspace.pinned ? 'Unpin from sidebar' : 'Pin to sidebar',
    });

    if (canModifyWorkspace({ workspace })) {
      if (!workspace.archived) {
        menuItems.push({ key: MenuKey.Edit, label: 'Edit...' });
      }
      menuItems.push({
        key: MenuKey.SwitchArchived,
        label: workspace.archived ? 'Unarchive' : 'Archive',
      });
    } else if (!workspace.archived) {
      menuItems.push({ key: MenuKey.Edit, label: 'View Config' });
    }
    if (canDeleteWorkspace({ workspace }) && workspace.numExperiments === 0) {
      menuItems.push({ type: 'divider' });
      menuItems.push({ danger: true, key: MenuKey.Delete, label: 'Delete...' });
    }
  }

  return { contextHolders, menu: menuItems, onClick: handleDropdown };
};

const WorkspaceActionDropdown: React.FC<Props> = ({
  children,
  className,
  direction = 'vertical',
  isContextMenu,
  returnIndexOnDelete = true,
  workspace,
  onComplete,
}: Props) => {
  const { contextHolders, menu, onClick } = useWorkspaceActionMenu({
    onComplete,
    returnIndexOnDelete,
    workspace,
  });

  return children ? (
    <>
      <Dropdown isContextMenu={isContextMenu} menu={menu} onClick={onClick}>
        {children}
      </Dropdown>
      {contextHolders}
    </>
  ) : (
    <div className={[css.base, className].join(' ')} title="Open actions menu">
      <Dropdown menu={menu} placement="bottomRight" onClick={onClick}>
        <Button icon={<Icon name={`overflow-${direction}`} title="Action menu" />} />
      </Dropdown>
      {contextHolders}
    </div>
  );
};

export default WorkspaceActionDropdown;
