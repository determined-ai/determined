import { Dropdown } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Button from 'components/kit/Button';
import useModalProjectDelete from 'hooks/useModal/Project/useModalProjectDelete';
import useModalProjectEdit from 'hooks/useModal/Project/useModalProjectEdit';
import useModalProjectMove from 'hooks/useModal/Project/useModalProjectMove';
import usePermissions from 'hooks/usePermissions';
import { archiveProject, unarchiveProject } from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { DetailedUser, Project } from 'types';
import handleError from 'utils/error';

interface Props {
  children?: React.ReactNode;
  className?: string;
  curUser?: DetailedUser;
  direction?: 'vertical' | 'horizontal';
  onComplete?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  project: Project;
  showChildrenIfEmpty?: boolean;
  trigger?: ('click' | 'hover' | 'contextMenu')[];
  workspaceArchived?: boolean;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ProjectActionDropdown: React.FC<Props> = ({
  project,
  children,
  onVisibleChange,
  showChildrenIfEmpty = true,
  className,
  direction = 'vertical',
  onComplete,
  trigger,
  workspaceArchived = false,
}: Props) => {
  const { contextHolder: modalProjectMoveContextHolder, modalOpen: openProjectMove } =
    useModalProjectMove({ onClose: onComplete, project });
  const { contextHolder: modalProjectDeleteContextHolder, modalOpen: openProjectDelete } =
    useModalProjectDelete({ onClose: onComplete, project });
  const { contextHolder: modalProjectEditContextHolder, modalOpen: openProjectEdit } =
    useModalProjectEdit({ onClose: onComplete, project });

  const { canDeleteProjects, canModifyProjects, canMoveProjects } = usePermissions();

  const handleEditClick = useCallback(() => openProjectEdit(), [openProjectEdit]);

  const handleMoveClick = useCallback(() => openProjectMove(), [openProjectMove]);

  const handleArchiveClick = useCallback(async () => {
    if (project.archived) {
      try {
        await unarchiveProject({ id: project.id });
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive project.' });
      }
    } else {
      try {
        await archiveProject({ id: project.id });
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive project.' });
      }
    }
  }, [onComplete, project.archived, project.id]);

  const handleDeleteClick = useCallback(() => {
    openProjectDelete();
  }, [openProjectDelete]);

  const menuProps: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      Delete: 'delete',
      Edit: 'edit',
      Move: 'move',
      SwitchArchived: 'switchArchive',
    } as const;

    const funcs = {
      [MenuKey.Edit]: () => {
        handleEditClick();
      },
      [MenuKey.Move]: () => {
        handleMoveClick();
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

    const items: MenuProps['items'] = [];
    if (
      canModifyProjects({ project, workspace: { id: project.workspaceId } }) &&
      !project.archived
    ) {
      items.push({ key: MenuKey.Edit, label: 'Edit...' });
    }
    if (canMoveProjects({ project }) && !project.archived) {
      items.push({ key: MenuKey.Move, label: 'Move...' });
    }
    if (
      canModifyProjects({ project, workspace: { id: project.workspaceId } }) &&
      !workspaceArchived
    ) {
      const label = project.archived ? 'Unarchive' : 'Archive';
      items.push({ key: MenuKey.SwitchArchived, label: label });
    }
    if (
      canDeleteProjects({ project, workspace: { id: project.workspaceId } }) &&
      !project.archived &&
      project.numExperiments === 0
    ) {
      items.push({ danger: true, key: MenuKey.Delete, label: 'Delete...' });
    }
    return { items: items, onClick: onItemClick };
  }, [
    canDeleteProjects,
    canModifyProjects,
    canMoveProjects,
    handleArchiveClick,
    handleDeleteClick,
    handleEditClick,
    handleMoveClick,
    project,
    workspaceArchived,
  ]);

  const contextHolders = useMemo(
    () => (
      <>
        {modalProjectDeleteContextHolder}
        {modalProjectEditContextHolder}
        {modalProjectMoveContextHolder}
      </>
    ),
    [modalProjectDeleteContextHolder, modalProjectEditContextHolder, modalProjectMoveContextHolder],
  );

  if (menuProps.items?.length === 0 && !showChildrenIfEmpty) {
    return null;
  }

  return children ? (
    <>
      <Dropdown
        disabled={menuProps.items?.length === 0}
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
      <Dropdown
        disabled={menuProps.items?.length === 0}
        menu={menuProps}
        placement="bottomRight"
        trigger={trigger ?? ['click']}>
        <Button ghost={true} type="text" onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </Button>
      </Dropdown>
      {contextHolders}
    </div>
  );
};

export default ProjectActionDropdown;
