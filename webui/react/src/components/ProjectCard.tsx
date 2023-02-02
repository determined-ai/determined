import { Typography } from 'antd';
import React, { useCallback } from 'react';

import Tooltip from 'components/kit/Tooltip';
import TimeAgo from 'components/TimeAgo';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import { nearestCardinalNumber } from 'shared/utils/number';
import { routeToReactUrl } from 'shared/utils/routes';
import { Project } from 'types';

import Card from './kit/Card';
import { useProjectActionMenu } from './ProjectActionDropdown';
import css from './ProjectCard.module.scss';

interface Props {
  fetchProjects?: () => void;
  project: Project;
  showWorkspace?: boolean;
  workspaceArchived?: boolean;
}

const ProjectCard: React.FC<Props> = ({ project, fetchProjects, workspaceArchived }: Props) => {
  const { menuProps, contextHolders } = useProjectActionMenu({
    onComplete: fetchProjects,
    project,
    workspaceArchived,
  });

  const handleCardClick = useCallback(() => {
    routeToReactUrl(paths.projectDetails(project.id));
  }, [project.id]);

  const classnames = [css.base];
  if (project.archived) classnames.push(css.archived);

  return (
    <Card actionMenu={!project.immutable ? menuProps : undefined} onClick={handleCardClick}>
      <div className={classnames.join(' ')}>
        <Typography.Title className={css.name} ellipsis={{ rows: 4, tooltip: true }} level={5}>
          {project.name}
        </Typography.Title>
        <div className={css.footer}>
          <div className={css.experiments}>
            <Tooltip
              title={
                `${project.numExperiments.toLocaleString()}` +
                ` experiment${project.numExperiments === 1 ? '' : 's'}`
              }>
              <Icon name="experiment" size="small" />
              <span>{nearestCardinalNumber(project.numExperiments)}</span>
            </Tooltip>
          </div>
          {project.archived ? (
            <div className={css.archivedBadge}>Archived</div>
          ) : (
            project.lastExperimentStartedAt && (
              <TimeAgo
                className={css.lastExperiment}
                datetime={project.lastExperimentStartedAt}
                tooltipFormat="[Last experiment started ]MMM D, YYYY - h:mm a"
              />
            )
          )}
        </div>
        {contextHolders}
      </div>
    </Card>
  );
};

export default ProjectCard;
