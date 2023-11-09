import Progress from 'hew/Progress';
import { useTheme } from 'hew/Theme';
import React from 'react';

import { ExperimentBase } from 'types';
import { getStateColorThemeVar } from 'utils/color';

interface Props {
  experiment: ExperimentBase;
}

const ExperimentHeaderProgress: React.FC<Props> = ({ experiment }: Props) => {
  const { getThemeVar } = useTheme();
  const progressPercent = (experiment.progress ?? 0) * 100;
  return experiment.progress === undefined ? null : (
    <Progress
      flat
      parts={[
        {
          color: getThemeVar(getStateColorThemeVar(experiment.state)),
          label: `${Math.round(progressPercent)}%`,
          percent: progressPercent / 100,
        },
      ]}
      showTooltips
    />
  );
};

export default ExperimentHeaderProgress;
