import { Dropdown } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import useModalWorkspaceCreate from 'hooks/useModal/Workspace/useModalWorkspaceCreate';
import useModalWorkspaceDelete from 'hooks/useModal/Workspace/useModalWorkspaceDelete';
import usePermissions from 'hooks/usePermissions';
import { archiveWorkspace, pinWorkspace, unarchiveWorkspace, unpinWorkspace } from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { useUpdateWorkspace } from 'stores/workspaces';
import { Workspace } from 'types';
import handleError from 'utils/error';

interface Props {
  children?: React.ReactNode;
  className?: string;
  direction?: 'vertical' | 'horizontal';
  onComplete?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  trigger?: ('click' | 'hover' | 'contextMenu')[];
  workspace: Workspace;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const WorkspaceActionDropdown: React.FC<Props> = ({
  children,
  className,
  direction = 'vertical',
  workspace,
  onComplete,
  trigger,
  onVisibleChange,
}: Props) => {
  const { contextHolder: modalWorkspaceDeleteContextHolder, modalOpen: openWorkspaceDelete } =
    useModalWorkspaceDelete({ onClose: onComplete, workspace });
  const { contextHolder: modalWorkspaceEditContextHolder, modalOpen: openWorkspaceEdit } =
    useModalWorkspaceCreate({ onClose: onComplete, workspaceID: workspace.id });

  const { canDeleteWorkspace, canModifyWorkspace } = usePermissions();

  const updateWorkspace = useUpdateWorkspace();

  const handleArchiveClick = useCallback(async () => {
    if (workspace.archived) {
      try {
        await unarchiveWorkspace({ id: workspace.id });
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive workspace.' });
      }
    } else {
      try {
        await archiveWorkspace({ id: workspace.id });
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive workspace.' });
      }
    }
  }, [onComplete, workspace.archived, workspace.id]);

  const handlePinClick = useCallback(async () => {
    if (workspace.pinned) {
      try {
        await unpinWorkspace({ id: workspace.id });
        updateWorkspace(workspace.id, (w) => ({ ...w, pinned: false }));
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unpin workspace.' });
      }
    } else {
      try {
        await pinWorkspace({ id: workspace.id });
        updateWorkspace(workspace.id, (w) => ({ ...w, pinned: true }));
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to pin workspace.' });
      }
    }
  }, [onComplete, workspace.id, workspace.pinned, updateWorkspace]);

  const handleEditClick = useCallback(() => {
    openWorkspaceEdit();
  }, [openWorkspaceEdit]);

  const handleDeleteClick = useCallback(() => {
    openWorkspaceDelete();
  }, [openWorkspaceDelete]);

  const WorkspaceActionMenu: DropDownProps['menu'] = useMemo(() => {
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
        handleEditClick();
      },
      [MenuKey.SwitchArchived]: () => {
        handleArchiveClick();
      },
      [MenuKey.Delete]: () => {
        handleDeleteClick();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
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
      menuItems.push({ key: MenuKey.Delete, label: 'Delete...' });
    }
    return { items: menuItems, onClick: onItemClick };
  }, [
    canDeleteWorkspace,
    canModifyWorkspace,
    handlePinClick,
    workspace,
    handleEditClick,
    handleArchiveClick,
    handleDeleteClick,
  ]);

  return children ? (
    <>
      <Dropdown
        menu={WorkspaceActionMenu}
        placement="bottomLeft"
        trigger={trigger ?? ['contextMenu', 'click']}
        onOpenChange={onVisibleChange}>
        {children}
      </Dropdown>
      {modalWorkspaceDeleteContextHolder}
      {modalWorkspaceEditContextHolder}
    </>
  ) : (
    <div
      className={[css.base, className].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown menu={WorkspaceActionMenu} placement="bottomRight" trigger={trigger ?? ['click']}>
        <button onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </button>
      </Dropdown>
      {modalWorkspaceDeleteContextHolder}
      {modalWorkspaceEditContextHolder}
    </div>
  );
};

export default WorkspaceActionDropdown;
