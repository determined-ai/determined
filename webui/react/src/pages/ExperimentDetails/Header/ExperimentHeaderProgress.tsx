import { Progress, Tooltip } from 'antd';
import React from 'react';

import { getStateColorCssVar } from 'themes';
import { ExperimentBase } from 'types';

import css from './ExperimentHeaderProgress.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentHeaderProgress: React.FC<Props> = ({ experiment }: Props) => {
  const progressPercent = (experiment.progress ?? 0) * 100;
  return experiment.progress === undefined ? null : (
    <Tooltip title={progressPercent.toFixed(0) + '%'}>
      <Progress
        className={css.base}
        percent={progressPercent}
        showInfo={false}
        status="active"
        strokeColor={getStateColorCssVar(experiment.state)}
      />
    </Tooltip>
  );
};

export default ExperimentHeaderProgress;
