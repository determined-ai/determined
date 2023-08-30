import React from 'react';

import BadgeTag from 'components/BadgeTag';
import { Metric } from 'types';

interface Props {
  metric: Metric;
}

const MetricBadgeTag: React.FC<Props> = ({ metric }: Props) => {
  /**
   * TODO - also see `utils/metrics.ts`
   * Metric group may sometimes end up being `undefined` when an old metric setting
   * is restored and the UI attempts to use it. Adding a safeguard for now.
   * Better approach of hunting down all the places it can be stored as a setting
   * and validating it upon loading and discarding it if invalid.
   */
  return (
    <BadgeTag label={metric.name} tooltip={metric.group}>
      {(metric.group ?? '').substring(0, 1).toUpperCase()}
    </BadgeTag>
  );
};

export default MetricBadgeTag;
