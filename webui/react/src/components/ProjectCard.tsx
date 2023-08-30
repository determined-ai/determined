import { Typography } from 'antd';
import React from 'react';

import DynamicIcon from 'components/DynamicIcon';
import Card from 'components/kit/Card';
import Icon from 'components/kit/Icon';
import Tooltip from 'components/kit/Tooltip';
import { useProjectActionMenu } from 'components/ProjectActionDropdown';
import css from 'components/ProjectCard.module.scss';
import TimeAgo from 'components/TimeAgo';
import { paths } from 'routes/utils';
import { Project } from 'types';
import { nearestCardinalNumber } from 'utils/number';

interface Props {
  fetchProjects?: () => void;
  project: Project;
  showWorkspace?: boolean;
  workspaceArchived?: boolean;
}

const ProjectCard: React.FC<Props> = ({
  project,
  fetchProjects,
  workspaceArchived,
  showWorkspace,
}: Props) => {
  const { contextHolders, menu, onClick } = useProjectActionMenu({
    onComplete: fetchProjects,
    onDelete: fetchProjects,
    project,
    workspaceArchived,
  });

  const classnames = [css.base];
  if (project.archived) classnames.push(css.archived);

  return (
    <>
      <Card
        actionMenu={!project.immutable ? menu : undefined}
        href={paths.projectDetails(project.id)}
        onDropdown={onClick}>
        <div className={classnames.join(' ')}>
          <div className={css.headerContainer}>
            <Typography.Title className={css.name} ellipsis={{ rows: 3, tooltip: true }} level={5}>
              {project.name}
            </Typography.Title>
          </div>
          <div className={css.workspaceContainer}>
            {showWorkspace && project.workspaceId !== 1 && (
              <Tooltip content={project.workspaceName}>
                <div className={css.workspaceIcon}>
                  <DynamicIcon name={project.workspaceName} size={20} />
                </div>
              </Tooltip>
            )}
          </div>
          <div className={css.footerContainer}>
            <div className={css.experiments}>
              <Tooltip
                content={
                  `${project.numExperiments.toLocaleString()}` +
                  ` experiment${project.numExperiments === 1 ? '' : 's'}`
                }>
                <Icon name="experiment" size="small" title="Number of experiments" />
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
        </div>
      </Card>
      {/*
        contextHolders must be outside of Card component to prevent unexpected action
        for more info, refer PR #6185
      */}
      {contextHolders}
    </>
  );
};

export default ProjectCard;
