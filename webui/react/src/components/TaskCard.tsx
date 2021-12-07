import React from 'react';
import TimeAgo from 'timeago-react';

import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import Link from 'components/Link';
import ProgressBar from 'components/ProgressBar';
import TaskActionDropdown from 'components/TaskActionDropdown';
import { AnyTask, DetailedUser, ExperimentTask, RecentCommandTask, RecentEvent } from 'types';
import { percent } from 'utils/number';
import { canBeOpened, isExperimentTask } from 'utils/task';
import { openCommand } from 'wait';

import css from './TaskCard.module.scss';

type Props = AnyTask & RecentEvent & {curUser?: DetailedUser}

const TaskCard: React.FC<Props> = ({ curUser, ...task }: Props) => {
  const classes = [ css.base ];

  const isExperiment = isExperimentTask(task);
  const progress = (task as ExperimentTask).progress;
  const hasProgress = isExperiment && progress != null;
  const isComplete = isExperiment && progress === 1;
  const iconName = isExperiment ? 'experiment' : (task as RecentCommandTask).type.toLowerCase();

  if (canBeOpened(task)) classes.push(css.link);

  return (
    <div className={classes.join(' ')}>
      <Link
        disabled={!canBeOpened(task)}
        inherit
        path={task.url ? task.url : undefined}
        popout={!isExperimentTask(task)}
        onClick={!isExperimentTask(task) ? (() => openCommand(task)) : undefined}>
        {isExperimentTask(task) && (
          <div className={css.progressBar}>
            <ProgressBar barOnly percent={(task.progress || 0) * 100} state={task.state} />
          </div>
        )}
        <div className={css.upper}>
          <div className={css.icon}><Icon name={iconName} /></div>
          <div className={css.info}>
            <div className={css.name}>{task.name}</div>
            <div className={css.age}>
              <div className={css.event}>{task.lastEvent.name}</div>
              <TimeAgo datetime={task.lastEvent.date} />
            </div>
          </div>
        </div>
        <div className={css.lower}>
          <div className={css.badges}>
            <Badge type={BadgeType.Default}>{`${task.id}`.slice(0, 4)}</Badge>
            <Badge state={task.state} type={BadgeType.State} />
            {isExperimentTask(task) && hasProgress && !isComplete && (
              <div className={css.percent}>{`${percent(task.progress || 0)}%`}</div>
            )}
          </div>
          <TaskActionDropdown curUser={curUser} task={task} />
        </div>
      </Link>
    </div>
  );
};

export default TaskCard;
