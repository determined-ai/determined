import React, { ReactNode, useCallback, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { activeRunStates } from 'constants/states';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { useAgents, useClusterOverview } from 'stores/agents';
import { useExperiments, useFetchExperiments } from 'stores/experiments';
import { useResourcePools } from 'stores/resourcePools';
import { useActiveTasks, useFetchActiveTasks } from 'stores/tasks';
import { ShirtSize } from 'themes';
import { ResourceType } from 'types';
import { Loadable } from 'utils/loadable';

import { maxClusterSlotCapacity } from '../Clusters/ClustersOverview';

export const ClusterOverallStats: React.FC = () => {
  const loadableResourcePools = useResourcePools();
  const resourcePools = Loadable.getOrElse([], loadableResourcePools); // TODO show spinner when this is loading
  const overview = useClusterOverview();
  const agents = useAgents();

  const [canceler] = useState(new AbortController());
  const fetchActiveExperiments = useFetchExperiments(
    { limit: -2, states: activeRunStates },
    canceler,
  );
  const fetchActiveTasks = useFetchActiveTasks(canceler);
  const fetchActiveRunning = useCallback(async () => {
    await fetchActiveExperiments();
    await fetchActiveTasks();
  }, [fetchActiveExperiments, fetchActiveTasks]);

  usePolling(fetchActiveRunning);
  const activeExperiments = useExperiments({ limit: -2, states: activeRunStates });
  const activeTasks = useActiveTasks();
  const rbacEnabled = useFeature().isOn('rbac');

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
        {usePermissions().canAdministrateUsers || !rbacEnabled ? (
          <>
            <OverviewStats title="Active Experiments">
              {Loadable.match(activeExperiments, {
                Loaded: (activeExperiments) => activeExperiments.pagination?.total ?? 0,
                NotLoaded: (): ReactNode => <Spinner />,
              })}
            </OverviewStats>
            <OverviewStats title="Active JupyterLabs">
              {Loadable.match(activeTasks, {
                Loaded: (activeTasks) => activeTasks.notebooks ?? 0,
                NotLoaded: (): ReactNode => <Spinner />,
              })}
            </OverviewStats>
            <OverviewStats title="Active TensorBoards">
              {Loadable.match(activeTasks, {
                Loaded: (activeTasks) => activeTasks.tensorboards ?? 0,
                NotLoaded: (): ReactNode => <Spinner />,
              })}
            </OverviewStats>
            <OverviewStats title="Active Shells">
              {Loadable.match(activeTasks, {
                Loaded: (activeTasks) => activeTasks.shells ?? 0,
                NotLoaded: (): ReactNode => <Spinner />,
              })}
            </OverviewStats>
            <OverviewStats title="Active Commands">
              {Loadable.match(activeTasks, {
                Loaded: (activeTasks) => activeTasks.commands ?? 0,
                NotLoaded: (): ReactNode => <Spinner />,
              })}
            </OverviewStats>
          </>
        ) : null}
      </Grid>
    </Section>
  );
};
