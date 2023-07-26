import React, { ReactNode, useEffect, useMemo } from 'react';

import Card from 'components/kit/Card';
import Spinner from 'components/kit/Spinner';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { activeRunStates } from 'constants/states';
import usePermissions from 'hooks/usePermissions';
import { GetExperimentsParams } from 'services/types';
import clusterStore from 'stores/cluster';
import { maxClusterSlotCapacity } from 'stores/cluster';
import determinedStore from 'stores/determinedInfo';
import experimentStore from 'stores/experiments';
import taskStore from 'stores/tasks';
import { ResourceType } from 'types';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

const ACTIVE_EXPERIMENTS_PARAMS: Readonly<GetExperimentsParams> = {
  limit: -2, // according to API swagger doc, [limit] -2 - returns pagination info but no experiments.
  states: activeRunStates,
};

export const ClusterOverallStats: React.FC = () => {
  const agents = useObservable(clusterStore.agents);
  const resourcePools = useObservable(clusterStore.resourcePools);
  const clusterOverview = useObservable(clusterStore.clusterOverview);
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
            Loaded: (agents) => (agents ? agents.length : '?'),
            NotLoaded: (): ReactNode => <Spinner spinning />,
          })}
        </OverviewStats>
        {Loadable.match(Loadable.all([maxTotalSlots, clusterOverview]), {
          Loaded: ([maxTotalSlots, clusterOverview]) =>
            [ResourceType.CUDA, ResourceType.ROCM, ResourceType.CPU].map((resType) =>
              maxTotalSlots[resType] > 0 ? (
                <OverviewStats key={resType} title={`${resType} Slots Allocated`}>
                  {clusterOverview[resType].total - clusterOverview[resType].available}
                  <small> / {maxTotalSlots[resType]}</small>
                </OverviewStats>
              ) : null,
            ),
          NotLoaded: () => null,
        })}
        {auxContainers.total > 0 && (
          <OverviewStats title="Aux Containers Running">
            {auxContainers.running} <small> / {auxContainers.total}</small>
          </OverviewStats>
        )}
        {(usePermissions().canAdministrateUsers || !rbacEnabled) && (
          <>
            <OverviewStats title="Active Experiments">
              {Loadable.match(activeExperiments, {
                Loaded: (activeExperiments) => activeExperiments.pagination?.total ?? 0,
                NotLoaded: (): ReactNode => <Spinner spinning />,
              })}
            </OverviewStats>
            <OverviewStats title="Active JupyterLabs">
              {Loadable.match(activeTasks, {
                Loaded: (activeTasks) => activeTasks.notebooks ?? 0,
                NotLoaded: (): ReactNode => <Spinner spinning />,
              })}
            </OverviewStats>
            <OverviewStats title="Active TensorBoards">
              {Loadable.match(activeTasks, {
                Loaded: (activeTasks) => activeTasks.tensorboards ?? 0,
                NotLoaded: (): ReactNode => <Spinner spinning />,
              })}
            </OverviewStats>
            <OverviewStats title="Active Shells">
              {Loadable.match(activeTasks, {
                Loaded: (activeTasks) => activeTasks.shells ?? 0,
                NotLoaded: (): ReactNode => <Spinner spinning />,
              })}
            </OverviewStats>
            <OverviewStats title="Active Commands">
              {Loadable.match(activeTasks, {
                Loaded: (activeTasks) => activeTasks.commands ?? 0,
                NotLoaded: (): ReactNode => <Spinner spinning />,
              })}
            </OverviewStats>
          </>
        )}
      </Card.Group>
    </Section>
  );
};
