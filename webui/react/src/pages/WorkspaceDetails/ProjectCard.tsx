import { EllipsisOutlined } from '@ant-design/icons';
import { Dropdown, Menu } from 'antd';
import React, { useCallback, useMemo } from 'react';

import Avatar from 'components/Avatar';
import Icon from 'components/Icon';
import TimeAgo from 'components/TimeAgo';
import useModalProjectMove from 'hooks/useModal/useModalProjectMove';
import { Project } from 'types';
import handleError from 'utils/error';

import css from './ProjectCard.module.scss';

interface Props {
  project: Project;
}

const ProjectCard: React.FC<Props> = ({ project }: Props) => {
  const { modalOpen: openMoveModal } = useModalProjectMove({ projectId: project.id });

  const handleEditClick = useCallback(() => {
    // bring up edit project modal
  }, [ ]);

  const handleMoveClick = useCallback(() => {
    openMoveModal();
  }, [ openMoveModal ]);

  const handleArchiveClick = useCallback(() => {
    if (project.archived) {
      try {
        // unarchive project
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to unarchive workspace.' });
      }
    } else {
      try {
        // archive project
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to archive workspace.' });
      }
    }
  }, [ project.archived ]);

  const handleDeleteClick = useCallback(() => {
    // bring up delete workspace modal
  }, []);

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
        <h6 className={css.name}>{project.name}</h6>
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
