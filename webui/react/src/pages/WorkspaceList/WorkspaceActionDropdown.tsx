import Button from 'determined-ui/Button';
import Dropdown, { MenuItem } from 'determined-ui/Dropdown';
import Icon from 'determined-ui/Icon';
import { useModal } from 'determined-ui/Modal';
import React, { RefObject, useCallback, useMemo } from 'react';

import css from 'components/ActionDropdown/ActionDropdown.module.scss';
import WorkspaceCreateModalComponent from 'components/WorkspaceCreateModal';
import WorkspaceDeleteModalComponent from 'components/WorkspaceDeleteModal';
import usePermissions from 'hooks/usePermissions';
import workspaceStore from 'stores/workspaces';
import { Workspace } from 'types';
import handleError from 'utils/error';

interface Props {
  children?: React.ReactNode;
  className?: string;
  containerRef: RefObject<HTMLElement>;
  direction?: 'vertical' | 'horizontal';
  isContextMenu?: boolean;
  onComplete?: () => void;
  returnIndexOnDelete?: boolean;
  workspace: Workspace;
}

interface WorkspaceMenuPropsIn {
  containerRef: RefObject<HTMLElement>;
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
  containerRef,
  onComplete,
  returnIndexOnDelete = true,
  workspace,
}: WorkspaceMenuPropsIn) => {
  const WorkspaceDeleteModal = useModal(WorkspaceDeleteModalComponent);
  const WorkspaceEditModal = useModal(WorkspaceCreateModalComponent);

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
            <WorkspaceEditModal.Component workspaceId={workspace.id} onClose={onComplete} />
          </>
        )}
      </>
    );
  }, [WorkspaceDeleteModal, WorkspaceEditModal, onComplete, workspace, returnIndexOnDelete]);

  const { canDeleteWorkspace, canModifyWorkspace } = usePermissions();

  const handleArchiveClick = useCallback(() => {
    if (!workspace) return;
    if (workspace.archived) {
      workspaceStore
        .unarchiveWorkspace(workspace.id)
        .then(() => onComplete?.())
        .catch((e) =>
          handleError(containerRef, e, { publicSubject: 'Unable to unarchive workspace.' }),
        );
    } else {
      workspaceStore
        .archiveWorkspace(workspace.id)
        .then(() => onComplete?.())
        .catch((e) =>
          handleError(containerRef, e, { publicSubject: 'Unable to archive workspace.' }),
        );
    }
  }, [containerRef, onComplete, workspace]);

  const handlePinClick = useCallback(() => {
    if (!workspace) return;

    if (workspace.pinned) {
      workspaceStore
        .unpinWorkspace(workspace.id)
        .then(() => onComplete?.())
        .catch((e) =>
          handleError(containerRef, e, { publicSubject: 'Unable to unpin workspace.' }),
        );
    } else {
      workspaceStore
        .pinWorkspace(workspace.id)
        .then(() => onComplete?.())
        .catch((e) => handleError(containerRef, e, { publicSubject: 'Unable to pin workspace.' }));
    }
  }, [containerRef, onComplete, workspace]);

  const MenuKey = {
    Delete: 'delete',
    Edit: 'edit',
    SwitchArchived: 'switchArchive',
    SwitchPin: 'switchPin',
  } as const;

  const handleDropdown = (key: string) => {
    switch (key) {
      case MenuKey.Edit:
        WorkspaceEditModal.open();
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
    }
    if (canDeleteWorkspace({ workspace }) && workspace.numExperiments === 0) {
      menuItems.push({ type: 'divider' });
      menuItems.push({ danger: true, key: MenuKey.Delete, label: 'Delete...' });
    }
  }

  return { contextHolders, menu: menuItems, onClick: handleDropdown };
};

const WorkspaceActionDropdown: React.FC<Props> = ({
  containerRef,
  children,
  className,
  direction = 'vertical',
  isContextMenu,
  returnIndexOnDelete = true,
  workspace,
  onComplete,
}: Props) => {
  const { contextHolders, menu, onClick } = useWorkspaceActionMenu({
    containerRef,
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
