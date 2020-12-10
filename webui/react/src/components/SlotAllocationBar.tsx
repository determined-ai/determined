import React, { useMemo } from 'react';

import Bar, { BarPart } from 'components/Bar';
import { getStateColorCssVar, ShirtSize } from 'themes';
import { CommandState, ResourceState } from 'types';
import { floatToPercent } from 'utils/string';

import css from './SlotAllocation.module.scss';

export interface Props {
  barOnly?: boolean;
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

const legend = (part: BarPart , count: number) => {
  return <li>
    <span>
      {count} ({floatToPercent(part.percent, 1)})
    </span>
    <span style={{ color: part.color }}>
      {' ' + part.label}
    </span>
  </li>;
};

const ProgressBar: React.FC<Props> = ({ resourceStates, totalSlots, ...barProps }: Props) => {

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
        color: getStateColorCssVar(ResourceState.Unspecified), // TODO
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
      legend(parts.running, tally.running),
      legend(parts.pending, tally.pending),
      legend(parts.free, tally.free),
    ];

    return { legends, parts: [ parts.running, parts.pending, parts.free ] };
  }, [ resourceStates, totalSlots ]);

  return (
    <div className={css.base}>
      <div className={css.header}>
        <header>GPU Slots Allocated</header>
        <span>3/10(33%)</span>
      </div>
      <div className={css.bar}>
        <Bar {...barProps} parts={parts} />
      </div>
      <div className={css.legends}>
        <ol>
          {legends}
        </ol>
      </div>
    </div>
  );
};

export default ProgressBar;
