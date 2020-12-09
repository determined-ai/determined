import React, { useMemo } from 'react';

import Bar, { BarPart } from 'components/Bar';
import { getStateColorCssVar } from 'themes';
import { ResourceState } from 'types';
import { floatToPercent } from 'utils/string';

import css from './SlotAllocation.module.scss';

export interface Props {
  barOnly?: boolean;
  resourceStates: ResourceState[];
  totalSlots: number;
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

const ProgressBar: React.FC<Props> = ({ barOnly, resourceStates, totalSlots }: Props) => {

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

    const parts: BarPart[] = [
      {
        color: getStateColorCssVar(ResourceState.Running),
        label: 'Running',
        percent: tally.running / totalSlots,
      },
      {
        color: 'purple', // TODO
        label: 'Pending',
        percent: tally.pending / totalSlots,
      },
      {
        color: 'Green', // TODO
        label: 'Free',
        percent: tally.free / totalSlots,
      },
    ];

    const legends = [
      legend(parts[0], tally.running),
      legend(parts[1], tally.pending),
      legend(parts[2], tally.free),
    ];

    return { legends, parts };
  }, [ resourceStates, totalSlots ]);

  return (
    <div className={css.base}>
      <div>
        <header>GPU Slots Allocated</header>
        <span>3/10(33%)</span>
      </div>
      <Bar parts={parts} />
      <div className={css.legends}>
        <ol>
          {legends}
        </ol>
      </div>
    </div>
  );
};

export default ProgressBar;
