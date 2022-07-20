import { Dropdown, Menu } from 'antd';
import type { MenuProps } from 'antd';
import React, { PropsWithChildren, useCallback, useMemo } from 'react';

import useModalProjectDelete from 'hooks/useModal/Project/useModalProjectDelete';
import useModalProjectEdit from 'hooks/useModal/Project/useModalProjectEdit';
import useModalProjectMove from 'hooks/useModal/Project/useModalProjectMove';
import { archiveProject, unarchiveProject } from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { DetailedUser, Project } from 'types';
import handleError from 'utils/error';

interface Props {
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

const ProjectActionDropdown: React.FC<Props> = (
  {
    project, children, curUser, onVisibleChange, showChildrenIfEmpty = true,
    className, direction = 'vertical', onComplete, trigger, workspaceArchived = false,
  }
  : PropsWithChildren<Props>,
) => {
  const {
    contextHolder: modalProjectMoveContextHolder,
    modalOpen: openProjectMove,
  } = useModalProjectMove({ onClose: onComplete, project });
  const {
    contextHolder: modalProjectDeleteContextHolder,
    modalOpen: openProjectDelete,
  } = useModalProjectDelete({ onClose: onComplete, project });
  const {
    contextHolder: modalProjectEditContextHolder,
    modalOpen: openProjectEdit,
  } = useModalProjectEdit({ onClose: onComplete, project });

  const userHasPermissions = useMemo(() => {
    return curUser?.isAdmin || curUser?.id === project.userId;
  }, [ curUser?.id, curUser?.isAdmin, project.userId ]);

  const handleEditClick = useCallback(() => openProjectEdit(), [ openProjectEdit ]);

  const handleMoveClick = useCallback(() => openProjectMove(), [ openProjectMove ]);

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
  }, [ onComplete, project.archived, project.id ]);

  const handleDeleteClick = useCallback(() => {
    openProjectDelete();
  }, [ openProjectDelete ]);

  const menuProps: {items: MenuProps['items'], onClick: MenuProps['onClick']} = useMemo(() => {
    const EDIT = 'edit';
    const MOVE = 'move';
    const SWITCH_ARCHIVED = 'switchArchive';
    const DELETE = 'delete';
    const items: MenuProps['items'] = [];

    const onItemClick: MenuProps['onClick'] = (e) => {
      switch(e.key) {
        case EDIT:
          handleEditClick();
          break;
        case MOVE:
          handleMoveClick();
          break;
        case SWITCH_ARCHIVED:
          handleArchiveClick();
          break;
        case DELETE:
          handleDeleteClick();
          break;
        default:
          break;
      }
    };

    if (userHasPermissions && !project.archived) {
      items.push({ key: EDIT, label: 'Edit...' });
      items.push({ key: MOVE, label: 'Move...' });
    }
    if (userHasPermissions && !workspaceArchived) {
      items.push({ key: SWITCH_ARCHIVED, label: project.archived ? 'Unarchive' : 'Archive' });
    }
    if (userHasPermissions && !project.archived && project.numExperiments === 0) {
      items.push({ danger: true, key: 'delete', label: 'Delete...' });
    }
    return { items: items, onClick: onItemClick };
  }, [
    handleArchiveClick,
    handleDeleteClick,
    handleEditClick,
    handleMoveClick,
    project.archived,
    project.numExperiments,
    userHasPermissions,
    workspaceArchived,
  ]);

  const contextHolders = useMemo(() => (
    <>
      {modalProjectDeleteContextHolder}
      {modalProjectEditContextHolder}
      {modalProjectMoveContextHolder}
    </>
  ), [
    modalProjectDeleteContextHolder,
    modalProjectEditContextHolder,
    modalProjectMoveContextHolder,
  ]);

  if (menuProps.items?.length === 0 && !showChildrenIfEmpty) {
    return null;
  }

  return children ? (
    <>
      <Dropdown
        disabled={menuProps.items?.length === 0}
        overlay={<Menu {...menuProps} />}
        placement="bottomLeft"
        trigger={trigger ?? [ 'contextMenu', 'click' ]}
        onVisibleChange={onVisibleChange}>
        {children}
      </Dropdown>
      {contextHolders}
    </>
  ) : (
    <div
      className={[ css.base, className ].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown
        disabled={menuProps.items?.length === 0}
        overlay={<Menu {...menuProps} />}
        placement="bottomRight"
        trigger={trigger ?? [ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </button>
      </Dropdown>
      {contextHolders}
    </div>
  );
};

export default ProjectActionDropdown;
