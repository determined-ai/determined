import { Tooltip, Typography } from 'antd';
import React, { useCallback } from 'react';

import Link from 'components/Link';
import TimeAgo from 'components/TimeAgo';
import Avatar from 'components/UserAvatar';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import { routeToReactUrl } from 'shared/utils/routes';
import { useUsers } from 'stores/users';
import { DetailedUser, Project } from 'types';
import { Loadable } from 'utils/loadable';

import ProjectActionDropdown from './ProjectActionDropdown';
import css from './ProjectCard.module.scss';

interface Props {
  curUser?: DetailedUser;
  fetchProjects?: () => void;
  project: Project;
  workspaceArchived?: boolean;
}

const ProjectCard: React.FC<Props> = ({
  project,
  curUser,
  fetchProjects,
  workspaceArchived,
}: Props) => {
  const handleCardClick = useCallback(() => {
    routeToReactUrl(paths.projectDetails(project.id));
  }, [project.id]);

  const users = Loadable.getOrElse([], useUsers());
  const user = users.find((user) => user.id === project.userId);
  return (
    <ProjectActionDropdown
      curUser={curUser}
      project={project}
      trigger={['contextMenu']}
      workspaceArchived={workspaceArchived}
      onComplete={fetchProjects}>
      <div className={css.base} onClick={handleCardClick}>
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
            project={project}
            workspaceArchived={workspaceArchived}
            onComplete={fetchProjects}
          />
        )}
        <Typography.Paragraph className={css.description} ellipsis={{ rows: 2, tooltip: true }}>
          {project.description}
        </Typography.Paragraph>
        <div className={css.experiments}>
          <Tooltip
            title={
              `${project.numExperiments.toLocaleString()}` +
              ` experiment${project.numExperiments === 1 ? '' : 's'}`
            }>
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
        <div className={css.avatar}>
          <Avatar user={user} />
        </div>
      </div>
    </ProjectActionDropdown>
  );
};

export default ProjectCard;
