import React, { useMemo } from 'react';

import Badge from 'components/Badge';
import Bar from 'components/Bar';
import { getStateColorCssVar, ShirtSize } from 'themes';
import { ResourceState } from 'types';
import { floatToPercent } from 'utils/string';

import { BadgeType } from './Badge';
import css from './SlotAllocation.module.scss';

export interface Props {
  barOnly?: boolean;
  className?: string;
  resourceStates: ResourceState[];
  showLegends?: boolean;
  size?: ShirtSize;
  totalSlots: number;
}

const pendingStates = new Set<ResourceState>([
  ResourceState.Assigned,
  ResourceState.Pulling,
  ResourceState.Terminated,
  ResourceState.Unspecified,
  ResourceState.Starting,
]);

const legend = (label: React.ReactNode , count: number, totalSlots: number) => {
  return <li>
    <span>
      {count} ({floatToPercent(count/totalSlots, 1)})
    </span>
    <span>
      {' '}
      {label}
    </span>
  </li>;
};

const SlotAllocationBar: React.FC<Props> = ({
  resourceStates,
  totalSlots,
  showLegends,
  className,
  ...barProps
}: Props) => {

  const { parts, legends } = useMemo(() => {
    const tally = {
      free: totalSlots - resourceStates.length,
      pending: 0,
      running: 0,
    };
    resourceStates.forEach(state => {
      if (pendingStates.has(state)) tally.pending++;
      if (state === ResourceState.Running) tally.running++;
    });

    const parts = {
      free: {
        color: 'var(--theme-colors-monochrome-15)', // TODO
        label: 'Free',
        percent: tally.free / totalSlots,
      },
      pending: {
        color: '#6666CC', // TODO
        label: 'Pending',
        percent: tally.pending / totalSlots,
      },
      running: {
        color: getStateColorCssVar(ResourceState.Running),
        label: 'Running',
        percent: tally.running / totalSlots,
      },
    };

    const legends = [
      legend(
        <Badge bgColor={parts.running.color} type={BadgeType.Custom}>{parts.running.label}</Badge>
        , tally.running,
        totalSlots,
      ),
      legend(
        <Badge bgColor={parts.pending.color} type={BadgeType.Custom}>
          {parts.pending.label}
        </Badge>
        , tally.pending,
        totalSlots,
      ),
      legend(
        <Badge bgColor={parts.free.color} fgColor="#234B65" type={BadgeType.Custom}>
          {parts.free.label}
        </Badge>
        , tally.free,
        totalSlots,
      ),
    ];

    return { legends, parts: [ parts.running, parts.pending ] };
  }, [ resourceStates, totalSlots ]);

  const classes = [ css.base ];
  if (className) classes.push(className);

  return (
    <div className={classes.join(' ')}>
      <div className={css.header}>
        <header>GPU Slots Allocated</header>
        <span>
          {resourceStates.length}/{totalSlots}
          {totalSlots > 0 ? ` (${floatToPercent( resourceStates.length/totalSlots, 0)})` : ''}
        </span>
      </div>
      <div className={css.bar}>
        <Bar {...barProps} parts={parts} />
      </div>
      {showLegends &&
      <div className={css.legends}>
        <ol>
          {legends}
        </ol>
      </div>
      }
    </div>
  );
};

export default SlotAllocationBar;
