import React, {useMemo} from "react";
import Section from 'components/Section';
import Grid, { GridMode } from 'components/Grid';
import { ShirtSize } from 'themes';
import OverviewStats from 'components/OverviewStats';
import { useStore } from 'contexts/Store';
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

  const [ cudaTotalSlots, rocmTotalSlots ] = useMemo(() => {
    return resourcePools.reduce((acc, pool) => {
      let index;
      switch (pool.slotType) {
        case ResourceType.CUDA:
          index = 0;
          break;
        case ResourceType.ROCM:
          index = 1;
          break;
        default:
          index = undefined;
      }
      if (index === undefined) return acc;
      acc[index] += pool.maxAgents * (pool.slotsPerAgent ?? 0);
      return acc;
    }, [ 0, 0 ]);
  }, [ resourcePools ]);

  return (
    <Section hideTitle title="Overview Stats">
        <Grid gap={ShirtSize.medium} minItemWidth={150} mode={GridMode.AutoFill}>
          <OverviewStats title="Connected Agents">
            {agents ? agents.length : '?'}
          </OverviewStats>
          {cudaTotalSlots ? (
            <OverviewStats title="CUDA Slots Allocated">
              {overview.CUDA.total - overview.CUDA.available} <small>/ {cudaTotalSlots}</small>
            </OverviewStats>
          ) : null}
          {rocmTotalSlots ? (
            <OverviewStats title="ROCm Slots Allocated">
              {overview.ROCM.total - overview.ROCM.available} <small>/ {rocmTotalSlots}</small>
            </OverviewStats>
          ) : null}
          {overview.CPU.total ? (
            <OverviewStats title="CPU Slots Allocated">
              {overview.CPU.total - overview.CPU.available} <small>/ {overview.CPU.total}</small>
            </OverviewStats>
          ) : null}
          {auxContainers.total ? (
            <OverviewStats title="Aux Containers Running">
              {auxContainers.running} <small>/ {auxContainers.total}</small>
            </OverviewStats>
          ) : null}
        </Grid>
      </Section>
  )
}