import React, { ReactNode, useMemo } from 'react';

import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { useStore } from 'contexts/Store';
import Spinner from 'shared/components/Spinner';
import { useAgents, useClusterOverview } from 'stores/agents';
import { useExperiments } from 'stores/experiments';
import { useTasks } from 'stores/tasks';
import { ShirtSize } from 'themes';
import { ResourceType } from 'types';
import { Loadable } from 'utils/loadable';

import { maxClusterSlotCapacity } from '../Clusters/ClustersOverview';

export const ClusterOverallStats: React.FC = () => {
  const { resourcePools } = useStore();
  const overview = useClusterOverview();
  const agents = useAgents();
  const activeExperiments = useExperiments(); // { active: true }
  const activeTasks = useTasks(); // { active: true }

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
        <OverviewStats title="Active Experiments">
          {Loadable.match(activeExperiments, {
            Loaded: (activeExperiments) => (activeExperiments ? activeExperiments.length : '?'),
            NotLoaded: (): ReactNode => <Spinner />,
          })}
        </OverviewStats>
        <OverviewStats title="Active JupyterLabs">
          {Loadable.match(activeTasks, {
            Loaded: (activeTasks) => activeTasks.notebooks,
            NotLoaded: (): ReactNode => <Spinner />,
          })}
        </OverviewStats>
        <OverviewStats title="Active TensorBoards">
          {Loadable.match(activeTasks, {
            Loaded: (activeTasks) => activeTasks.tensorboards,
            NotLoaded: (): ReactNode => <Spinner />,
          })}
        </OverviewStats>
        <OverviewStats title="Active Shells">
          {Loadable.match(activeTasks, {
            Loaded: (activeTasks) => activeTasks.shells,
            NotLoaded: (): ReactNode => <Spinner />,
          })}
        </OverviewStats>
        <OverviewStats title="Active Commands">
          {Loadable.match(activeTasks, {
            Loaded: (activeTasks) => activeTasks.commands,
            NotLoaded: (): ReactNode => <Spinner />,
          })}
        </OverviewStats>
      </Grid>
    </Section>
  );
};
