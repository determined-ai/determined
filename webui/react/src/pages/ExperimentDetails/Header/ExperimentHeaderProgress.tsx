import { Progress } from 'antd';
import { getStateColorCssVar } from 'determined-ui/Theme';
import Tooltip from 'determined-ui/Tooltip';
import React from 'react';

import { ExperimentBase, JobState, RunState } from 'types';

import css from './ExperimentHeaderProgress.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentHeaderProgress: React.FC<Props> = ({ experiment }: Props) => {
  const progressPercent = (experiment.progress ?? 0) * 100;
  const status =
    experiment.state === JobState.SCHEDULED ||
    experiment.state === JobState.SCHEDULEDBACKFILLED ||
    experiment.state === RunState.Active
      ? 'active'
      : undefined;

  return experiment.progress === undefined ? null : (
    <Tooltip content={progressPercent.toFixed(0) + '%'}>
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
