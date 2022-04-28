import { Dropdown, Menu } from 'antd';
import React, { PropsWithChildren, useCallback, useMemo } from 'react';

import css from 'components/ActionDropdown.module.scss';
import Icon from 'components/Icon';
import useModalProjectDelete from 'hooks/useModal/Project/useModalProjectDelete';
import useModalProjectEdit from 'hooks/useModal/Project/useModalProjectEdit';
import useModalProjectMove from 'hooks/useModal/Project/useModalProjectMove';
import { archiveProject, unarchiveProject } from 'services/api';
import { DetailedUser, Project } from 'types';
import handleError from 'utils/error';

interface Props {
  className?: string;
  curUser?: DetailedUser;
  direction?: 'vertical' | 'horizontal';
  onComplete?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  project: Project;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ProjectActionDropdown: React.FC<Props> = (
  { project, children, curUser, onVisibleChange, className, direction = 'vertical', onComplete }
  : PropsWithChildren<Props>,
) => {
  const { modalOpen: openProjectMove } = useModalProjectMove({ onClose: onComplete, project });
  const { modalOpen: openProjectDelete } = useModalProjectDelete({
    onClose: onComplete,
    project,
  });
  const { modalOpen: openProjectEdit } = useModalProjectEdit({ onClose: onComplete, project });

  const userHasPermissions = useMemo(() => {
    return curUser?.isAdmin || curUser?.username === project.username;
  }, [ curUser?.isAdmin, curUser?.username, project.username ]);

  const handleEditClick = useCallback(() => {
    openProjectEdit();
  }, [ openProjectEdit ]);

  const handleMoveClick = useCallback(() => {
    openProjectMove();
  }, [ openProjectMove ]);

  const handleArchiveClick = useCallback(() => {
    if (project.archived) {
      try {
        unarchiveProject({ id: project.id });
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive project.' });
      }
    } else {
      try {
        archiveProject({ id: project.id });
        onComplete?.();
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive project.' });
      }
    }
  }, [ onComplete, project.archived, project.id ]);

  const handleDeleteClick = useCallback(() => {
    openProjectDelete();
  }, [ openProjectDelete ]);

  const ProjectActionMenu = useMemo(() => {
    return (
      <Menu>
        {userHasPermissions && !project.archived &&
          <Menu.Item key="edit" onClick={handleEditClick}>Edit...</Menu.Item>}
        {userHasPermissions && !project.archived &&
          <Menu.Item key="move" onClick={handleMoveClick}>Move...</Menu.Item>}
        {userHasPermissions && (
          <Menu.Item key="switchArchive" onClick={handleArchiveClick}>
            {project.archived ? 'Unarchive' : 'Archive'}
          </Menu.Item>
        )}
        {userHasPermissions && !project.archived &&
        <Menu.Item danger key="delete" onClick={handleDeleteClick}>Delete...</Menu.Item>}
      </Menu>
    );
  }, [ handleArchiveClick,
    handleDeleteClick,
    handleEditClick,
    handleMoveClick,
    project.archived,
    userHasPermissions ]);

  if (!userHasPermissions) {
    return (children as JSX.Element) ?? (
      <div className={css.base} title="No actions available" onClick={stopPropagation}>
        <button disabled>
          <Icon name={`overflow-${direction}`} />
        </button>
      </div>
    );
  }

  return children ? (
    <Dropdown
      overlay={ProjectActionMenu}
      placement="bottomLeft"
      trigger={[ 'contextMenu', 'click' ]}
      onVisibleChange={onVisibleChange}>
      {children}
    </Dropdown>
  ) : (
    <div
      className={[ css.base, className ].join(' ')}
      title="Open actions menu"
      onClick={stopPropagation}>
      <Dropdown overlay={ProjectActionMenu} placement="bottomRight" trigger={[ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name={`overflow-${direction}`} />
        </button>
      </Dropdown>
    </div>
  );
};

export default ProjectActionDropdown;
