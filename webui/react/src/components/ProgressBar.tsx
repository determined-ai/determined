import React from 'react';

import Bar from 'components/Bar';
import { getStateColorCssVar } from 'themes';
import { CommandState, RunState } from 'types';

export interface Props {
  barOnly?: boolean;
  percent: number;
  state: RunState | CommandState;
}

const ProgressBar: React.FC<Props> = ({ barOnly, percent, state }: Props) => {
  return (
    <Bar
      barOnly={barOnly}
      parts={[ {
        color: getStateColorCssVar(state),
        label: `${percent}%`,
        percent: percent / 100,
      } ]} />
  );
};

export default ProgressBar;
