import React, { useMemo } from 'react';

import Bar from 'components/Bar';
import { getStateColorCssVar, ShirtSize } from 'themes';
import { ResourceState } from 'types';
import { floatToPercent } from 'utils/string';

import css from './SlotAllocation.module.scss';

export interface Props {
  barOnly?: boolean;
  showLegends?: boolean;
  resourceStates: ResourceState[];
  totalSlots: number;
  size?: ShirtSize;
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

const ProgressBar: React.FC<Props> = ({
  resourceStates,
  totalSlots,
  showLegends,
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
        color: getStateColorCssVar(ResourceState.Terminated), // TODO
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
      legend(parts.running.label, tally.running, totalSlots),
      legend(parts.pending.label, tally.pending, totalSlots),
      legend(parts.free.label, tally.free, totalSlots),
    ];

    return { legends, parts: [ parts.running, parts.pending ] };
  }, [ resourceStates, totalSlots ]);

  return (
    <div className={css.base}>
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

export default ProgressBar;
