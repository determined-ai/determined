import { Tooltip } from 'antd';
import React from 'react';

import { CommonProps } from 'types';

interface Props extends CommonProps {
  num: number;
  precision?: number;
}

const defaultProps: Props = {
  num: 0,
  precision: 6,
};

const HumanReadableFloat: React.FC<Props> = ({ num, precision }: Props) => {
  let numToString: string = num.toFixed(precision);
  if (num < 0.01 || num > 999) {
    numToString = num.toExponential(precision);
  }

  return (
    <Tooltip title={num.toString()}>
      <span>{numToString}</span>
    </Tooltip>
  );
};

HumanReadableFloat.defaultProps = defaultProps;

export default HumanReadableFloat;
