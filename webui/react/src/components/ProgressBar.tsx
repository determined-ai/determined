import React from 'react';

import Bar from 'components/Bar';
import { floatToPercent } from 'shared/utils/string';
import { getStateColorCssVar, StateOfUnion } from 'themes';

export interface Props {
  barOnly?: boolean;
  percent: number;
  state: StateOfUnion;
}

const ProgressBar: React.FC<Props> = ({ barOnly, percent, state }: Props) => {
  return (
    <Bar
      barOnly={barOnly}
      parts={[
        {
          color: getStateColorCssVar(state),
          label: floatToPercent(percent / 100, 0),
          percent: percent / 100,
        },
      ]}
    />
  );
};

export default ProgressBar;
