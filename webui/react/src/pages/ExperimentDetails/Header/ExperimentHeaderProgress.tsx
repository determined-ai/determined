import React from 'react';

import Progress from 'components/kit/Progress';
import { getStateColorCssVar } from 'components/kit/Theme';
import { ExperimentBase } from 'types';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentHeaderProgress: React.FC<Props> = ({ experiment }: Props) => {
  const progressPercent = (experiment.progress ?? 0) * 100;
  return experiment.progress === undefined ? null : (
    <Progress
      inline
      parts={[
        {
          color: getStateColorCssVar(experiment.state),
          percent: progressPercent / 100,
        },
      ]}
      tooltip={progressPercent.toFixed(0) + '%'}
    />
  );
};

export default ExperimentHeaderProgress;
