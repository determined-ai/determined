import { Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import WorkspaceCreateModalComponent from 'components/WorkspaceCreateModal';
import WorkspaceDeleteModalComponent from 'components/WorkspaceDeleteModal';
import usePermissions from 'hooks/usePermissions';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import { ValueOf } from 'shared/types';
import workspaceStore from 'stores/workspaces';
import { Workspace } from 'types';
import handleError from 'utils/error';

interface Props {
  children?: React.ReactNode;
  className?: string;
  direction?: 'vertical' | 'horizontal';
  onComplete?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  returnIndexOnDelete?: boolean;
  trigger?: ('click' | 'hover' | 'contextMenu')[];
  workspace: Workspace;
}

const stopPropagation = (e: React.UIEvent): void => e.stopPropagation();

interface WorkspaceMenuPropsIn {
  onComplete?: () => void;
  returnIndexOnDelete?: boolean;
  workspace: Workspace;
}

interface WorkspaceMenuPropsOut {
  contextHolders: React.ReactElement;
  menuProps: MenuProps;
}

export const useWorkspaceActionMenu: (props: WorkspaceMenuPropsIn) => WorkspaceMenuPropsOut = ({
  onComplete,
  returnIndexOnDelete = true,
  workspace,
}: WorkspaceMenuPropsIn) => {
  const WorkspaceDeleteModal = useModal(WorkspaceDeleteModalComponent);
  const WorkspaceEditModal = useModal(WorkspaceCreateModalComponent);

  const contextHolders = useMemo(() => {
    return (
      <>
        <WorkspaceDeleteModal.Component
          returnIndexOnDelete={returnIndexOnDelete}
          workspace={workspace}
          onClose={onComplete}
        />
        <WorkspaceEditModal.Component workspaceId={workspace.id} onClose={onComplete} />
      </>
    );
  }, [WorkspaceDeleteModal, WorkspaceEditModal, onComplete, workspace, returnIndexOnDelete]);

  const { canDeleteWorkspace, canModifyWorkspace } = usePermissions();

  const handleArchiveClick = useCallback(() => {
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
  }, [onComplete, workspace.archived, workspace.id]);

  const handlePinClick = useCallback(() => {
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
  }, [onComplete, workspace.id, workspace.pinned]);

  const MenuKey = {
    Delete: 'delete',
    Edit: 'edit',
    SwitchArchived: 'switchArchive',
    SwitchPin: 'switchPin',
  } as const;

  const funcs = {
    [MenuKey.SwitchPin]: () => {
      handlePinClick();
    },
    [MenuKey.Edit]: () => {
      WorkspaceEditModal.open();
    },
    [MenuKey.SwitchArchived]: () => {
      handleArchiveClick();
    },
    [MenuKey.Delete]: () => {
      WorkspaceDeleteModal.open();
    },
  };

  const onItemClick: MenuProps['onClick'] = (e) => {
    funcs[e.key as ValueOf<typeof MenuKey>]();
    stopPropagation(e.domEvent);
  };

  const menuItems: MenuProps['items'] = [
    {
      key: MenuKey.SwitchPin,
      label: workspace.pinned ? 'Unpin from sidebar' : 'Pin to sidebar',
    },
  ];

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
  return { contextHolders, menuProps: { items: menuItems, onClick: onItemClick } };
};

const WorkspaceActionDropdown: React.FC<Props> = ({
  children,
  className,
  direction = 'vertical',
  returnIndexOnDelete = true,
  workspace,
  onComplete,
  trigger,
  onVisibleChange,
}: Props) => {
  const { menuProps, contextHolders } = useWorkspaceActionMenu({
    onComplete,
    returnIndexOnDelete,
    workspace,
  });

  return children ? (
    <>
      <Dropdown
        menu={menuProps}
        placement="bottomLeft"
        trigger={trigger ?? ['contextMenu', 'click']}
        onOpenChange={onVisibleChange}>
        {children}
      </Dropdown>
      {contextHolders}
    </>
  ) : (
    <div
      className={[css.base, className].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown menu={menuProps} placement="bottomRight" trigger={trigger ?? ['click']}>
        <Button ghost icon={<Icon name={`overflow-${direction}`} />} onClick={stopPropagation} />
      </Dropdown>
      {contextHolders}
    </div>
  );
};

export default WorkspaceActionDropdown;
