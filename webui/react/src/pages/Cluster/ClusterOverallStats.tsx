import React, { ReactNode, useMemo } from 'react';

import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { useStore } from 'contexts/Store';
import Spinner from 'shared/components/Spinner';
import { useAgents, useClusterOverview } from 'stores/agents';
import { ShirtSize } from 'themes';
import { ResourceType } from 'types';
import { Loadable } from 'utils/loadable';

import { maxClusterSlotCapacity } from '../Clusters/ClustersOverview';

export const ClusterOverallStats: React.FC = () => {
  const { activeExperiments, activeTasks, resourcePools } = useStore();
  const overview = useClusterOverview();
  const agents = useAgents();

  const auxContainers = useMemo(() => {
    const tally = {
      running: 0,
      total: 0,
    };
    resourcePools.forEach((rp) => {
      tally.total += rp.auxContainerCapacity;
      tally.running += rp.auxContainersRunning;
    });
    return tally;
  }, [resourcePools]);

  const maxTotalSlots = useMemo(() => {
    return Loadable.map(agents, (agents) => maxClusterSlotCapacity(resourcePools, agents));
  }, [resourcePools, agents]);

  return (
    <Section hideTitle title="Overview Stats">
      <Grid gap={ShirtSize.Medium} minItemWidth={150} mode={GridMode.AutoFill}>
        <OverviewStats title="Connected Agents">
          {Loadable.match(agents, {
            Loaded: (agents) => (agents ? agents.length : '?'),
            NotLoaded: (): ReactNode => <Spinner />,
          })}
        </OverviewStats>
        {[ResourceType.CUDA, ResourceType.ROCM, ResourceType.CPU].map((resType) =>
          Loadable.match(Loadable.all([overview, maxTotalSlots]), {
            Loaded: ([overview, maxTotalSlots]) =>
              maxTotalSlots[resType] > 0 ? (
                <OverviewStats key={resType} title={`${resType} Slots Allocated`}>
                  {overview[resType].total - overview[resType].available}
                  <small>/ {maxTotalSlots[resType]}</small>
                </OverviewStats>
              ) : null,
            NotLoaded: () => undefined,
          }),
        )}
        {auxContainers.total ? (
          <OverviewStats title="Aux Containers Running">
            {auxContainers.running} <small>/ {auxContainers.total}</small>
          </OverviewStats>
        ) : null}
        {activeExperiments ? (
          <OverviewStats title="Active Experiments">{activeExperiments}</OverviewStats>
        ) : null}
        {activeTasks.notebooks ? (
          <OverviewStats title="Active JupyterLabs">{activeTasks.notebooks}</OverviewStats>
        ) : null}
        {activeTasks.tensorboards ? (
          <OverviewStats title="Active TensorBoards">{activeTasks.tensorboards}</OverviewStats>
        ) : null}
        {activeTasks.shells ? (
          <OverviewStats title="Active Shells">{activeTasks.shells}</OverviewStats>
        ) : null}
        {activeTasks.commands ? (
          <OverviewStats title="Active Commands">{activeTasks.commands}</OverviewStats>
        ) : null}
      </Grid>
    </Section>
  );
};
