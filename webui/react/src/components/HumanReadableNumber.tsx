import React from 'react';

import Tooltip from 'components/kit/Tooltip';
import { CommonProps } from 'shared/types';
import { humanReadableNumber } from 'shared/utils/number';

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
