import Progress from 'determined-ui/Progress';
import { getStateColorCssVar } from 'determined-ui/Theme';
import React from 'react';

import { ExperimentBase } from 'types';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentHeaderProgress: React.FC<Props> = ({ experiment }: Props) => {
  const progressPercent = (experiment.progress ?? 0) * 100;
  return experiment.progress === undefined ? null : (
    <Progress
      flat
      parts={[
        {
          color: getStateColorCssVar(experiment.state),
          label: `${Math.round(progressPercent)}%`,
          percent: progressPercent / 100,
        },
      ]}
      showTooltips
    />
  );
};

export default ExperimentHeaderProgress;
