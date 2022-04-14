import { EllipsisOutlined } from '@ant-design/icons';
import { Dropdown, Menu } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Avatar from 'components/Avatar';
import Icon from 'components/Icon';
import Link from 'components/Link';
import TimeAgo from 'components/TimeAgo';
import useModalProjectDelete from 'hooks/useModal/Project/useModalProjectDelete';
import useModalProjectEdit from 'hooks/useModal/Project/useModalProjectEdit';
import useModalProjectMove from 'hooks/useModal/Project/useModalProjectMove';
import { paths } from 'routes/utils';
import { archiveProject, unarchiveProject } from 'services/api';
import { Project } from 'types';
import handleError from 'utils/error';

import css from './ProjectCard.module.scss';

interface Props {
  project: Project;
}

const ProjectCard: React.FC<Props> = ({ project }: Props) => {
  const { modalOpen: openProjectMove } = useModalProjectMove({ projectId: project.id });
  const { modalOpen: openProjectDelete } = useModalProjectDelete({ project: project });
  const { modalOpen: openProjectEdit } = useModalProjectEdit({ project: project });

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

  const ActionMenu = useMemo(() => {
    return (
      <Menu>
        <Menu.Item onClick={handleEditClick}>Edit...</Menu.Item>
        <Menu.Item onClick={handleMoveClick}>Move...</Menu.Item>
        <Menu.Item onClick={handleArchiveClick}>
          {project.archived ? 'Unarchive' : 'Archive'}
        </Menu.Item>
        <Menu.Item danger onClick={handleDeleteClick}>Delete...</Menu.Item>
      </Menu>
    );
  }, [ handleArchiveClick, handleDeleteClick, handleEditClick, handleMoveClick, project.archived ]);

  return (
    <Dropdown disabled={project.immutable} overlay={ActionMenu} trigger={[ 'contextMenu' ]}>
      <div className={css.base}>
        <h6 className={css.name}>
          <Link inherit path={paths.projectDetails(project.id)}>
            {project.name}
          </Link>
        </h6>
        {!project.immutable && (
          <Dropdown arrow className={css.action} overlay={ActionMenu} trigger={[ 'click' ]}>
            <EllipsisOutlined />
          </Dropdown>
        )}
        <p className={css.description}>{project.description}</p>
        <div className={css.experiments}>
          <Icon name="experiment" size="small" />
          <span>{project.numExperiments}</span>
        </div>
        {project.lastExperimentStartedAt && (
          <TimeAgo className={css.lastExperiment} datetime={project.lastExperimentStartedAt} />
        )}
        <div className={css.avatar}><Avatar username={project.username} /></div>
      </div>
    </Dropdown>
  );
};

export default ProjectCard;
