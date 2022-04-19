import React from 'react';

import Avatar from 'components/Avatar';
import Icon from 'components/Icon';
import Link from 'components/Link';
import TimeAgo from 'components/TimeAgo';
import { paths } from 'routes/utils';
import { DetailedUser, Project } from 'types';

import ProjectActionDropdown from './ProjectActionDropdown';
import css from './ProjectCard.module.scss';

interface Props {
  curUser?: DetailedUser;
  fetchProjects?: () => void;
  project: Project;
}

const ProjectCard: React.FC<Props> = ({ project, curUser, fetchProjects }: Props) => {

  return (
    <ProjectActionDropdown curUser={curUser} fetchProjects={fetchProjects} project={project}>
      <div className={css.base}>
        <h6 className={css.name}>
          <Link inherit path={paths.projectDetails(project.id)}>
            {project.name}
          </Link>
        </h6>
        {!project.immutable && (
          <ProjectActionDropdown
            className={css.action}
            curUser={curUser}
            direction="horizontal"
            fetchProjects={fetchProjects}
            project={project}
          />
        )}
        <p className={css.description}>{project.description}</p>
        <div className={css.experiments}>
          <Icon name="experiment" size="small" />
          <span>{project.numExperiments.toLocaleString()}</span>
          {project.lastExperimentStartedAt && (
            <TimeAgo className={css.lastExperiment} datetime={project.lastExperimentStartedAt} />
          )}
        </div>
        <div className={css.avatar}><Avatar username={project.username} /></div>
      </div>
    </ProjectActionDropdown>
  );
};

export default ProjectCard;
