import React, { ReactNode, useCallback, useMemo, useState } from 'react';

import Card from 'components/kit/Card';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { activeRunStates } from 'constants/states';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { GetExperimentsParams } from 'services/types';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { useClusterStore } from 'stores/cluster';
import experimentStore from 'stores/experiments';
import { TasksStore } from 'stores/tasks';
import { ResourceType, TaskCounts } from 'types';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import { maxClusterSlotCapacity } from '../Clusters/ClustersOverview';

const ACTIVE_EXPERIMENTS_PARAMS: Readonly<GetExperimentsParams> = {
  limit: -2, // according to API swagger doc, [limit] -2 - returns pagination info but no experiments.
  states: activeRunStates,
};

export const ClusterOverallStats: React.FC = () => {
  const resourcePools = useObservable(useClusterStore().resourcePools);
  const agents = useObservable(useClusterStore().agents);
  const clusterOverview = useObservable(useClusterStore().clusterOverview);

  const [canceler] = useState(new AbortController());
  const fetchActiveExperiments = experimentStore.fetchExperiments(
    ACTIVE_EXPERIMENTS_PARAMS,
    canceler,
  );
  const activeTasks = useObservable<Loadable<TaskCounts>>(TasksStore.getActiveTaskCounts());

  const fetchActiveRunning = useCallback(async () => {
    await fetchActiveExperiments();
    TasksStore.fetchActiveTasks(canceler);
  }, [fetchActiveExperiments, canceler]);

  usePolling(fetchActiveRunning);
  const activeExperiments = useObservable(
    experimentStore.getExperimentsByParams(ACTIVE_EXPERIMENTS_PARAMS),
  );
  const rbacEnabled = useFeature().isOn('rbac');

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

  return (
    <Section hideTitle title="Overview Stats">
      <Card.Group size="small">
        <OverviewStats title="Connected Agents">
          {Loadable.match(agents, {
            Loaded: (agents) => (agents ? agents.length : '?'),
            NotLoaded: (): ReactNode => <Spinner />,
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
        )}
      </Card.Group>
    </Section>
  );
};
