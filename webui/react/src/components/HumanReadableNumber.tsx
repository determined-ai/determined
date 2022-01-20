import { Tooltip } from 'antd';
import React from 'react';

import { CommonProps } from 'types';
import { humanReadableNumber } from 'utils/number';

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
  const content = humanReadableNumber(num, precision);

  return (
    <Tooltip title={`${tooltipPrefix}${stringNum}`}>
      <span>{content}</span>
    </Tooltip>
  );
};

export default HumanReadableNumber;
