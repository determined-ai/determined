import React from 'react';

import { MetricName } from 'types';
import { capitalize } from 'utils/string';

import BadgeTag from './BadgeTag';

interface Props {
  metric: MetricName;
}

const MetricBadgeTag: React.FC<Props> = ({ metric }: Props) => {
  return (
    <BadgeTag label={metric.name} tooltip={metric.type}>{capitalize(metric.type)}</BadgeTag>
  );
};

export default MetricBadgeTag;
