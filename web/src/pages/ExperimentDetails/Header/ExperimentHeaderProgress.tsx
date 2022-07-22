import { Progress, Tooltip } from 'antd';
import React from 'react';

import { getStateColorCssVar } from 'themes';
import { ExperimentBase, JobState, RunState } from 'types';

import css from './ExperimentHeaderProgress.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const activeStates = [ JobState.SCHEDULED, JobState.SCHEDULEDBACKFILLED, RunState.Active ];

const ExperimentHeaderProgress: React.FC<Props> = ({ experiment }: Props) => {
  const progressPercent = (experiment.progress ?? 0) * 100;
  const status = activeStates.includes(experiment.state) ? 'active' : undefined;

  return experiment.progress === undefined ? null : (
    <Tooltip title={progressPercent.toFixed(0) + '%'}>
      <Progress
        className={css.base}
        percent={progressPercent}
        showInfo={false}
        status={status}
        strokeColor={getStateColorCssVar(experiment.state)}
      />
    </Tooltip>
  );
};

export default ExperimentHeaderProgress;
