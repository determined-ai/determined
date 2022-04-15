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
  onVisibleChange?: (visible: boolean) => void;
  project: Project;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ProjectActionDropdown: React.FC<Props> = (
  { project, children, curUser, onVisibleChange, className }: PropsWithChildren<Props>,
) => {
  const { modalOpen: openProjectMove } = useModalProjectMove({ projectId: project.id });
  const { modalOpen: openProjectDelete } = useModalProjectDelete({ project: project });
  const { modalOpen: openProjectEdit } = useModalProjectEdit({ project: project });

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
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive workspace.' });
      }
    } else {
      try {
        archiveProject({ id: project.id });
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive workspace.' });
      }
    }
  }, [ project.archived, project.id ]);

  const handleDeleteClick = useCallback(() => {
    openProjectDelete();
  }, [ openProjectDelete ]);

  const ProjectActionMenu = useMemo(() => {
    return (
      <Menu>
        {userHasPermissions &&
          <Menu.Item onClick={handleEditClick}>Edit...</Menu.Item>}
        {userHasPermissions &&
          <Menu.Item onClick={handleMoveClick}>Move...</Menu.Item>}
        {userHasPermissions && (
          <Menu.Item onClick={handleArchiveClick}>
            {project.archived ? 'Unarchive' : 'Archive'}
          </Menu.Item>
        )}
        {userHasPermissions &&
        <Menu.Item danger onClick={handleDeleteClick}>Delete...</Menu.Item>}
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
          <Icon name="overflow-vertical" />
        </button>
      </div>
    );
  }

  return children ? (
    <Dropdown
      overlay={ProjectActionMenu}
      placement="bottomLeft"
      trigger={[ 'contextMenu' ]}
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
          <Icon name="overflow-horizontal" />
        </button>
      </Dropdown>
    </div>
  );
};

export default ProjectActionDropdown;
