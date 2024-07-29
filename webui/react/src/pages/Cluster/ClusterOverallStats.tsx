import Card from 'hew/Card';
import Spinner from 'hew/Spinner';
import { Loadable } from 'hew/utils/loadable';
import React, { ReactNode, useEffect, useMemo } from 'react';

import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { activeRunStates } from 'constants/states';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { GetExperimentsParams } from 'services/types';
import clusterStore, { maxClusterSlotCapacity } from 'stores/cluster';
import determinedStore from 'stores/determinedInfo';
import experimentStore from 'stores/experiments';
import taskStore from 'stores/tasks';
import { ResourceType } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import { useObservable } from 'utils/observable';

const ACTIVE_EXPERIMENTS_PARAMS: Readonly<GetExperimentsParams> = {
  limit: -2, // according to API swagger doc, [limit] -2 - returns pagination info but no experiments.
  states: activeRunStates,
};

export const ClusterOverallStats: React.FC = () => {
  const agents = useObservable(clusterStore.agents);
  const resourcePools = useObservable(clusterStore.resourcePools);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const activeTasks = useObservable(taskStore.activeTasks);
  const activeExperiments = useObservable(
    experimentStore.getExperimentsByParams(ACTIVE_EXPERIMENTS_PARAMS),
  );
  const { rbacEnabled } = useObservable(determinedStore.info);

  const auxContainers = useMemo(() => {
    const tally = {
      running: 0,
      total: 0,
    };
    Loadable.isLoaded(resourcePools) &&
      resourcePools.data.forEach((rp) => {
        tally.total += rp.auxContainerCapacity;
        tally.running += rp.auxContainersRunning;
      });
    return tally;
  }, [resourcePools]);

  const maxTotalSlots = useMemo(() => {
    return Loadable.map(agents, (agents) =>
      maxClusterSlotCapacity(
        (Loadable.isLoaded(resourcePools) && resourcePools.data) || [],
        agents,
      ),
    );
  }, [resourcePools, agents]);

  useEffect(() => taskStore.startPolling(), []);
  useEffect(() => experimentStore.startPolling({ args: [ACTIVE_EXPERIMENTS_PARAMS] }), []);

  return (
    <Section hideTitle title="Overview Stats">
      <Card.Group size="small">
        <OverviewStats title="Connected Agents">
          {Loadable.match(agents, {
            Failed: () => null,
            Loaded: (agents) => (agents ? agents.length : '?'),
            NotLoaded: (): ReactNode => <Spinner spinning />, // TODO correctly handle error state
          })}
        </OverviewStats>
        {Loadable.match(Loadable.all([maxTotalSlots, agents]), {
          _: () => null,
          Loaded: ([maxTotalSlots, agents]) =>
            [ResourceType.CUDA, ResourceType.ROCM, ResourceType.CPU].map((resType) =>
              maxTotalSlots[resType] > 0 ? (
                <OverviewStats key={resType} title={`${resType} Slots Allocated`}>
                  {getSlotContainerStates(agents || [], resType).length}
                  <small> / {maxTotalSlots[resType]}</small>
                </OverviewStats>
              ) : null,
            ),
        })}
        {auxContainers.total > 0 && (
          <OverviewStats title="Aux Containers Running">
            {auxContainers.running} <small> / {auxContainers.total}</small>
          </OverviewStats>
        )}
        {(usePermissions().canAdministrateUsers || !rbacEnabled) && (
          <>
            {Loadable.match(activeExperiments, {
              _: () => null,
              Loaded: (activeExperiments) =>
                (activeExperiments.pagination?.total ?? 0) > 0 && (
                  <OverviewStats title={`Active ${f_flat_runs ? 'Searches' : 'Experiments'}`}>
                    {activeExperiments.pagination?.total}
                  </OverviewStats>
                ),
            })}
            {Loadable.match(activeTasks, {
              _: () => null,
              Loaded: (activeTasks) =>
                activeTasks.notebooks > 0 && (
                  <OverviewStats title="Active JupyterLabs">{activeTasks.notebooks}</OverviewStats>
                ),
            })}
            {Loadable.match(activeTasks, {
              _: () => null,
              Loaded: (activeTasks) =>
                activeTasks.tensorboards > 0 && (
                  <OverviewStats title="Active TensorBoards">
                    {activeTasks.tensorboards}
                  </OverviewStats>
                ),
            })}
            {Loadable.match(activeTasks, {
              _: () => null,
              Loaded: (activeTasks) =>
                activeTasks.shells > 0 && (
                  <OverviewStats title="Active Shells">{activeTasks.shells}</OverviewStats>
                ),
            })}
            {Loadable.match(activeTasks, {
              _: () => null,
              Loaded: (activeTasks) =>
                activeTasks.commands > 0 && (
                  <OverviewStats title="Active Commands">{activeTasks.commands}</OverviewStats>
                ),
            })}
          </>
        )}
      </Card.Group>
    </Section>
  );
};
