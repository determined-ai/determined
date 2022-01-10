import React from 'react';

import { MetricName } from 'types';

import BadgeTag from './BadgeTag';

interface Props {
  metric: MetricName;
}

const MetricBadgeTag: React.FC<Props> = ({ metric }: Props) => {
  return (
    <BadgeTag label={metric.name} tooltip={metric.type}>
      {metric.type.substr(0, 1).toUpperCase()}
    </BadgeTag>
  );
};

export default MetricBadgeTag;
