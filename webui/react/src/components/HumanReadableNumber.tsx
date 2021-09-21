import { Tooltip } from 'antd';
import React from 'react';

import { CommonProps } from 'types';

interface Props extends CommonProps {
  num?: number | null;
  precision?: number;
  tooltipPrefix?: string;
}

const HumanReadableNumber: React.FC<Props> = ({
  num,
  precision = 6,
  tooltipPrefix = '',
}: Props) => {
  if (num == null) return null;

  const stringNum = num.toString();
  let content: string = stringNum;

  if (isNaN(num)) {
    content = 'NaN';
  } else if (!Number.isFinite(num)) {
    content = `${num < 0 ? '-' : ''}Infinity`;
  } else if (!Number.isInteger(num)) {
    content = num.toFixed(precision);

    const absoluteNum = Math.abs(num);
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

export default HumanReadableNumber;
