import React from 'react';

import Badge from 'components/kit/Badge';
import Tooltip from 'components/kit/Tooltip';
import { Metric } from 'types';

import css from './MetricBadgeTag.module.scss';

interface Props {
  metric: Metric;
}

const TOOLTIP_DELAY = 1.0;

const MetricBadgeTag: React.FC<Props> = ({ metric }: Props) => {
  /**
   * TODO - also see `utils/metrics.ts`
   * Metric group may sometimes end up being `undefined` when an old metric setting
   * is restored and the UI attempts to use it. Adding a safeguard for now.
   * Better approach of hunting down all the places it can be stored as a setting
   * and validating it upon loading and discarding it if invalid.
   */
  return (
    <span className={css.base}>
      <Tooltip content={metric.group} mouseEnterDelay={TOOLTIP_DELAY}>
        <Badge text={(metric.group ?? '').substring(0, 1).toUpperCase()} />
      </Tooltip>
      <Tooltip content={metric.name} mouseEnterDelay={TOOLTIP_DELAY}>
        <span className={css.label}>{metric.name}</span>
      </Tooltip>
    </span>
  );
};

export default MetricBadgeTag;
