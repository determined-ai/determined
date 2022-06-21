import { Dropdown, Menu } from 'antd';
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
  const { modalOpen: openProjectMove } = useModalProjectMove({ onClose: onComplete, project });
  const { modalOpen: openProjectDelete } = useModalProjectDelete({
    onClose: onComplete,
    project,
  });
  const { modalOpen: openProjectEdit } = useModalProjectEdit({ onClose: onComplete, project });

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

  const menuItems = useMemo(() => {
    const items: React.ReactNode[] = [];
    if (userHasPermissions && !project.archived) {
      items.push(<Menu.Item key="edit" onClick={handleEditClick}>Edit...</Menu.Item>);
    }
    if (userHasPermissions && !project.archived) {
      items.push(<Menu.Item key="move" onClick={handleMoveClick}>Move...</Menu.Item>);
    }
    if (userHasPermissions && !workspaceArchived) {
      items.push((
        <Menu.Item key="switchArchive" onClick={handleArchiveClick}>
          {project.archived ? 'Unarchive' : 'Archive'}
        </Menu.Item>));
    }
    if (userHasPermissions && !project.archived && project.numExperiments === 0) {
      items.push(<Menu.Item danger key="delete" onClick={handleDeleteClick}>Delete...</Menu.Item>);
    }
    return items;
  }, [ handleArchiveClick,
    handleDeleteClick,
    handleEditClick,
    handleMoveClick,
    project.archived,
    project.numExperiments,
    userHasPermissions,
    workspaceArchived ]);

  if (menuItems.length === 0 && !showChildrenIfEmpty) {
    return null;
  }

  return children ? (
    <Dropdown
      disabled={menuItems.length === 0}
      overlay={(
        <Menu>
          {menuItems}
        </Menu>
      )}
      placement="bottomLeft"
      trigger={trigger ?? [ 'contextMenu', 'click' ]}
      onVisibleChange={onVisibleChange}>
      {children}
    </Dropdown>
  ) : (
    <div
      className={[ css.base, className ].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown
        disabled={menuItems.length === 0}
        overlay={(
          <Menu>
            {menuItems}
          </Menu>
        )}
        placement="bottomRight"
        trigger={trigger ?? [ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </button>
      </Dropdown>
    </div>
  );
};

export default ProjectActionDropdown;
