import Avatar, { Size } from 'hew/Avatar';
import Badge from 'hew/Badge';
import Card from 'hew/Card';
import Column from 'hew/Column';
import Icon from 'hew/Icon';
import Row from 'hew/Row';
import Tooltip from 'hew/Tooltip';
import { Title, TypographySize } from 'hew/Typography';
import { isUndefined } from 'lodash';
import React from 'react';

import TimeAgo from 'components/TimeAgo';
import useFeature from 'hooks/useFeature';
import { handlePath, paths } from 'routes/utils';
import { Project } from 'types';
import { nearestCardinalNumber } from 'utils/number';
import { AnyMouseEvent } from 'utils/routes';
import { pluralizer } from 'utils/string';

import { useProjectActionMenu } from './ProjectActionDropdown';
import css from './ProjectCard.module.scss';

interface Props {
  hideActionMenu?: boolean;
  onEdit?: (name: string, archived: boolean, description?: string) => void;
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
  const f_flat_runs = useFeature().isOn('flat_runs');

  const { contextHolders, menu, onClick } = useProjectActionMenu({
    onDelete: onRemove,
    onEdit,
    onMove: onRemove,
    project,
    workspaceArchived,
  });

  const classnames = [];
  if (project.archived) classnames.push(css.archived);
  if (project.workspaceId === 1) classnames.push(css.uncategorized);
  const testId = `card-${project.name}`;
  const f_flat_runs = useFeature().isOn('flat_runs');

  return (
    <Card
      actionMenu={!project.immutable && !hideActionMenu ? menu : undefined}
      testId={testId}
      onClick={(e: AnyMouseEvent) => handlePath(e, { path: paths.projectDetails(project.id) })}
      onDropdown={onClick}>
      <div className={classnames.join(' ')}>
        <Column>
          <Row justifyContent="space-between" width={125}>
            <Title size={TypographySize.XS} truncate={{ rows: 1, tooltip: true }}>
              {project.name}
            </Title>
          </Row>
          <Row>
            <div className={css.workspaceContainer}>
              {showWorkspace && (
                <div className={css.workspaceIcon}>
                  <Avatar
                    palette="muted"
                    size={Size.Small}
                    square
                    text={project.workspaceName || ''}
                  />
                </div>
              )}
            </div>
          </Row>
          <Row justifyContent="space-between" width="fill">
            <div className={css.footerContainer}>
              {f_flat_runs && !isUndefined(project.numRuns) && (
                <div className={css.experiments}>
                  <Tooltip
                    content={`${project.numRuns?.toLocaleString()} ${pluralizer(project.numRuns, 'run')}`}>
                    <Icon name="experiment" size="small" title="Number of runs" />
                    <span>{nearestCardinalNumber(project.numRuns)}</span>
                  </Tooltip>
                </div>
              )}
              {!f_flat_runs && !isUndefined(project.numExperiments) && (
                <div className={css.experiments}>
                  <Tooltip
                    content={`${project.numExperiments?.toLocaleString()} ${pluralizer(project.numExperiments, 'experiment')}`}>
                    <Icon name="experiment" size="small" title="Number of experiments" />
                    <span>{nearestCardinalNumber(project.numExperiments)}</span>
                  </Tooltip>
                </div>
              )}
              {project.archived ? (
                <Badge backgroundColor={{ h: 0, l: 40, s: 0 }} text="Archived" />
              ) : (
                project.lastExperimentStartedAt && (
                  <TimeAgo
                    datetime={project.lastExperimentStartedAt}
                    tooltipFormat={`[Last ${f_flat_runs ? 'search' : 'experiment'} started: \n]MMM D, YYYY - h:mm a`}
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
