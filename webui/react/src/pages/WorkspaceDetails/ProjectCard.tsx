import { Tooltip, Typography } from 'antd';
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
        <div className={css.nameRow}>
          <h6 className={css.name}>
            <Link inherit path={paths.projectDetails(project.id)}>
              <Typography.Paragraph ellipsis={{ rows: 1, tooltip: true }}>
                {project.name}
              </Typography.Paragraph>
            </Link>
          </h6>
          {project.archived && (
            <Tooltip title="Archived">
              <div>
                <Icon name="archive" size="small" />
              </div>
            </Tooltip>
          )}
        </div>
        {!project.immutable && (
          <ProjectActionDropdown
            className={css.action}
            curUser={curUser}
            direction="horizontal"
            fetchProjects={fetchProjects}
            project={project}
          />
        )}
        <Typography.Paragraph className={css.description} ellipsis={{ rows: 2, tooltip: true }}>
          {project.description}
        </Typography.Paragraph>
        <div className={css.experiments}>
          <Tooltip title={`${project.numExperiments.toLocaleString()}` +
            ` experiment${project.numExperiments === 1 ? '' : 's'}`}>
            <Icon name="experiment" size="small" />
            <span>{project.numExperiments.toLocaleString()}</span>
          </Tooltip>
          {project.lastExperimentStartedAt && (
            <TimeAgo
              className={css.lastExperiment}
              datetime={project.lastExperimentStartedAt}
              tooltipFormat="[Last experiment started ]MMM D, YYYY - h:mm a"
            />
          )}
        </div>
        <div className={css.avatar}><Avatar username={project.username} /></div>
      </div>
    </ProjectActionDropdown>
  );
};

export default ProjectCard;
