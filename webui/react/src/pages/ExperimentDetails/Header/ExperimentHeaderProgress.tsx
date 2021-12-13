import { Progress, Tooltip } from 'antd';
import React from 'react';

import { isProgressingRunStates } from 'constants/states';
import { ExperimentBase } from 'types';

import css from './ExperimentHeaderProgress.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentHeaderProgress: React.FC<Props> = ({ experiment }: Props) => {
  if (!isProgressingRunStates.has(experiment.state) || !experiment.progress) {
    return null;
  }

  const progressPercent = experiment.progress * 100;

  return (
    <Tooltip title={progressPercent.toFixed(0) + '%'}>
      <Progress
        className={css.base}
        percent={progressPercent}
        showInfo={false}
        status="active"
      />
    </Tooltip>
  );
};

export default ExperimentHeaderProgress;
