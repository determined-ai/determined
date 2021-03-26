import { Tooltip } from 'antd';
import React from 'react';

import { CommonProps } from 'types';

interface Props extends CommonProps {
  num: number;
  precision?: number;
  tooltipPrefix?: string;
}

const defaultProps: Props = {
  num: 0,
  precision: 6,
};

const HumanReadableFloat: React.FC<Props> = ({ num, precision, tooltipPrefix }: Props) => {
  const isInteger = Number.isInteger(num);
  const absoluteNum = Math.abs(num);
  const stringNum = num.toString();
  let content: string = stringNum;

  if (!isInteger) {
    content = num.toFixed(precision);
    if (absoluteNum < 0.01 || absoluteNum > 999) {
      content = num.toExponential(precision);
    }
  }

  return (
    <Tooltip title={`${tooltipPrefix}${stringNum}`}>
      <span>{content}</span>
    </Tooltip>
  );
};

HumanReadableFloat.defaultProps = defaultProps;

export default HumanReadableFloat;
