import React from 'react';

import Bar from 'components/Bar';
import { getStateColorCssVar } from 'themes';
import { CommandState, RunState } from 'types';
import { floatToPercent } from 'utils/string';

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
        label: floatToPercent(percent/100, 0),
        percent: percent / 100,
      } ]} />
  );
};

export default ProgressBar;
