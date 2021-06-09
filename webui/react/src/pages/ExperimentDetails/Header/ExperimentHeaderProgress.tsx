import { Progress, Tooltip } from 'antd';
import React from 'react';

import { ExperimentBase, RunState } from 'types';

import css from './ExperimentHeaderProgress.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentHeaderProgress: React.FC<Props> = ({ experiment }: Props) => {
  if (experiment.state !== RunState.Active || !experiment.progress) {
    return null;
  }

  return (
    <Tooltip title={experiment.progress.toFixed(2) + '%'}>
      <Progress
        className={css.base}
        percent={experiment.progress}
        showInfo={false}
        status="active"
      />
    </Tooltip>
  );
};

export default ExperimentHeaderProgress;
