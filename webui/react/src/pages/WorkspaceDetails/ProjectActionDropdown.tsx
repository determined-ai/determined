import { Dropdown, Menu } from 'antd';
import type { MenuProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import useModalProjectDelete from 'hooks/useModal/Project/useModalProjectDelete';
import useModalProjectEdit from 'hooks/useModal/Project/useModalProjectEdit';
import useModalProjectMove from 'hooks/useModal/Project/useModalProjectMove';
import { archiveProject, unarchiveProject } from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
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

const ProjectActionDropdown: React.FC<Props> = (
  {
    project, children, curUser, onVisibleChange, showChildrenIfEmpty = true,
    className, direction = 'vertical', onComplete, trigger, workspaceArchived = false,
  }
  : Props,
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
    enum MenuKey {
      EDIT = 'edit',
      MOVE = 'move',
      SWITCH_ARCHIVED = 'switchArchive',
      DELETE = 'delete',
    }

    const funcs = {
      [MenuKey.EDIT]: () => { handleEditClick(); },
      [MenuKey.MOVE]: () => { handleMoveClick(); },
      [MenuKey.SWITCH_ARCHIVED]: () => { handleArchiveClick(); },
      [MenuKey.DELETE]: () => { handleDeleteClick(); },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as MenuKey]();
    };

    const items: MenuProps['items'] = [];
    if (userHasPermissions && !project.archived) {
      items.push({ key: MenuKey.EDIT, label: 'Edit...' });
      items.push({ key: MenuKey.MOVE, label: 'Move...' });
    }
    if (userHasPermissions && !workspaceArchived) {
      const label = project.archived ? 'Unarchive' : 'Archive';
      items.push({ key: MenuKey.SWITCH_ARCHIVED, label: label });
    }
    if (userHasPermissions && !project.archived && project.numExperiments === 0) {
      items.push({ danger: true, key: MenuKey.DELETE, label: 'Delete...' });
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
