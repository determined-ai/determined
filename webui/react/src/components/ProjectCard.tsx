import Avatar, { Size } from 'hew/Avatar';
import Card from 'hew/Card';
import Column from 'hew/Column';
import Icon from 'hew/Icon';
import Row from 'hew/Row';
import Tooltip from 'hew/Tooltip';
import { Title, TypographySize } from 'hew/Typography';
import React from 'react';

import TimeAgo from 'components/TimeAgo';
import { handlePath, paths } from 'routes/utils';
import { Project } from 'types';
import { nearestCardinalNumber } from 'utils/number';
import { AnyMouseEvent } from 'utils/routes';

import { useProjectActionMenu } from './ProjectActionDropdown';
import css from './ProjectCard.module.scss';

interface Props {
  hideActionMenu?: boolean;
  onEdit?: (name: string, archived: boolean) => void;
  onRemove?: () => void;
  project: Project;
  showWorkspace?: boolean;
  workspaceArchived?: boolean;
}

const ProjectCard: React.FC<Props> = ({
  hideActionMenu,
  onRemove,
  onEdit,
  project,
  showWorkspace,
  workspaceArchived,
}: Props) => {
  const { contextHolders, menu, onClick } = useProjectActionMenu({
    onDelete: onRemove,
    onEdit,
    onMove: onRemove,
    project,
    workspaceArchived,
  });

  const classnames = [css.base];
  if (project.archived) classnames.push(css.archived);

  return (
    <Card
      actionMenu={!project.immutable && !hideActionMenu ? menu : undefined}
      onClick={(e: AnyMouseEvent) => handlePath(e, { path: paths.projectDetails(project.id) })}
      onDropdown={onClick}>
      <div className={classnames.join(' ')}>
        <Column>
          <Row width={125}>
            <Title size={TypographySize.XS} truncate={{ rows: 1, tooltip: true }}>
              {project.name}
            </Title>
          </Row>
          <Row>
            <div className={css.workspaceContainer}>
              {showWorkspace && project.workspaceId !== 1 && (
                <Tooltip content={project.workspaceName}>
                  <div className={css.workspaceIcon}>
                    <Avatar palette="muted" size={Size.Small} square text={project.workspaceName} />
                  </div>
                </Tooltip>
              )}
            </div>
          </Row>
          <Row>
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
                    datetime={project.lastExperimentStartedAt}
                    tooltipFormat="[Last experiment started: \n]MMM D, YYYY - h:mm a"
                  />
                )
              )}
            </div>
          </Row>
        </Column>
      </div>
      {contextHolders}
    </Card>
  );
};

export default ProjectCard;
