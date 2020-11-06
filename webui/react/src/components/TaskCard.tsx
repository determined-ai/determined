import React from 'react';
import TimeAgo from 'timeago-react';

import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import Link from 'components/Link';
import ProgressBar from 'components/ProgressBar';
import TaskActionDropdown from 'components/TaskActionDropdown';
import { RecentCommandTask, RecentTask } from 'types';
import { percent } from 'utils/number';
import { canBeOpened, isExperimentTask } from 'utils/task';
import { openCommand } from 'wait';

import css from './TaskCard.module.scss';

const TaskCard: React.FC<RecentTask> = (props: RecentTask) => {
  let [ hasProgress, isComplete ] = [ false, false ];
  if (isExperimentTask(props)) {
    hasProgress = !!props.progress;
    isComplete = props.progress === 1;
  }
  const classes = [ css.base ];

  const iconName = isExperimentTask(props) ?
    'experiment' : (props as RecentCommandTask).type.toLowerCase();
  if (canBeOpened(props)) classes.push(css.link);

  return (
    <div className={classes.join(' ')}>
      <Link
        disabled={!canBeOpened(props)}
        inherit
        path={props.url ? props.url : undefined}
        popout={!isExperimentTask(props)}
        onClick={!isExperimentTask(props) ? (() => openCommand(props)) : undefined}>
        {isExperimentTask(props) && <div className={css.progressBar}>
          <ProgressBar barOnly percent={(props.progress || 0) * 100} state={props.state} />
        </div>}
        <div className={css.upper}>
          <div className={css.icon}><Icon name={iconName} /></div>
          <div className={css.info}>
            <div className={css.name}>{props.name}</div>
            <div className={css.age}>
              <div className={css.event}>{props.lastEvent.name}</div>
              <TimeAgo datetime={props.lastEvent.date} />
            </div>
          </div>
        </div>
        <div className={css.lower}>
          <div className={css.badges}>
            <Badge type={BadgeType.Default}>{`${props.id}`.slice(0,4)}</Badge>
            <Badge state={props.state} type={BadgeType.State} />
            {isExperimentTask(props) && hasProgress && !isComplete
                && <div className={css.percent}>{`${percent(props.progress || 0)}%`}</div>}
          </div>
          <TaskActionDropdown task={props} />
        </div>
      </Link>
    </div>
  );
};

export default TaskCard;
