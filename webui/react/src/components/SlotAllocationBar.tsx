import React, { useMemo } from 'react';

import Bar, { BarPart } from 'components/Bar';
import { getStateColorCssVar } from 'themes';
import { ResourceState } from 'types';

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

const ProgressBar: React.FC<Props> = ({ barOnly, resourceStates, totalSlots }: Props) => {

  const parts = useMemo(() => {
    const tally = {
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
    ];

    return parts;

  }, [ resourceStates, totalSlots ]);

  return (
    <div>
      <div>
        <header>GPU Slots Allocated</header>
        <span>3/10(33%)</span>
      </div>
      <Bar parts={parts} />
    </div>
  );
};

export default ProgressBar;
