import React, { useMemo } from 'react';

import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { useStore } from 'contexts/Store';
import { ShirtSize } from 'themes';
import { ResourceType } from 'types';

export const ClusterOverallStats: React.FC = () => {
  const { agents, cluster: overview, resourcePools } = useStore();

  const auxContainers = useMemo(() => {
    const tally = {
      running: 0,
      total: 0,
    };
    resourcePools.forEach(rp => {
      tally.total += rp.auxContainerCapacity;
      tally.running += rp.auxContainersRunning;
    });
    return tally;
  }, [ resourcePools ]);

  /** theoretical max capacity for each slot type in the cluster */
  const maxTotalSlots = useMemo(() => {
    return agents.reduce((acc, agent) => {
      agent.resources.forEach(resource => {
        if (!(resource.type in acc)) acc[resource.type] = 0;
        acc[resource.type] += 1;
      });
      return acc;
    }, {} as { [key in ResourceType]: number });
  }, [ agents ]);

  return (
    <Section hideTitle title="Overview Stats">
      <Grid gap={ShirtSize.medium} minItemWidth={150} mode={GridMode.AutoFill}>
        <OverviewStats title="Connected Agents">
          {agents ? agents.length : '?'}
        </OverviewStats>
        {[ ResourceType.CUDA, ResourceType.ROCM, ResourceType.CPU ].map(resType => (
          (maxTotalSlots[resType] > 0) ? (
            <OverviewStats
              key={resType}
              title={`${resType} Slots Allocated`}>
              {overview[resType].total - overview[resType].available}
              <small>
                / {maxTotalSlots[resType]}
              </small>
            </OverviewStats>
          ) : null))}
        {auxContainers.total ? (
          <OverviewStats title="Aux Containers Running">
            {auxContainers.running} <small>/ {auxContainers.total}</small>
          </OverviewStats>
        ) : null}
      </Grid>
    </Section>
  );
};
