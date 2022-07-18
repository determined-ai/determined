import React, { useCallback, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import OverviewStats from 'components/OverviewStats';
import Section from 'components/Section';
import { useStore } from 'contexts/Store';
import { useFetchActiveExperiments, useFetchTasks } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import { ShirtSize } from 'themes';
import { CommandType, ResourceType } from 'types';

import { maxClusterSlotCapacity } from '../Clusters/ClustersOverview';

export const ClusterOverallStats: React.FC = () => {
  const {
    activeExperiments,
    activeTasks,
    agents,
    cluster: overview,
    resourcePools,
  } = useStore();

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
  }, [ resourcePools ]);

  const maxTotalSlots = useMemo(() => {
    return maxClusterSlotCapacity(resourcePools, agents);
  }, [ resourcePools, agents ]);

  const [ canceler ] = useState(new AbortController());
  const fetchActiveExperiments = useFetchActiveExperiments(canceler);
  const fetchTasks = useFetchTasks(canceler);

  const fetchActiveRunning = useCallback(() => {
    fetchActiveExperiments();
    fetchTasks();
  }, [ fetchActiveExperiments, fetchTasks ]);
  usePolling(fetchActiveRunning);

  return (
    <Section hideTitle title="Overview Stats">
      <Grid gap={ShirtSize.medium} minItemWidth={150} mode={GridMode.AutoFill}>
        <OverviewStats title="Connected Agents">
          {agents ? agents.length : '?'}
        </OverviewStats>
        {[ ResourceType.CUDA, ResourceType.ROCM, ResourceType.CPU ].map((resType) => (
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
          {activeExperiments ? (
            <OverviewStats title="Active Experiments">
              {activeExperiments}
            </OverviewStats>
          ) : null}
          {activeTasks[CommandType.JupyterLab] ? (
            <OverviewStats title="Active JupyterLabs">
              {activeTasks[CommandType.JupyterLab]}
            </OverviewStats>
          ) : null}
          {activeTasks[CommandType.TensorBoard] ? (
            <OverviewStats title="Active TensorBoards">
              {activeTasks[CommandType.TensorBoard]}
            </OverviewStats>
          ) : null}
          {activeTasks[CommandType.Shell] ? (
            <OverviewStats title="Active Shells">
              {activeTasks[CommandType.Shell]}
            </OverviewStats>
          ) : null}
          {activeTasks[CommandType.Command] ? (
            <OverviewStats title="Active Commands">
              {activeTasks[CommandType.Command]}
            </OverviewStats>
          ) : null}
        </Grid>
      </Section>
    </>
  );
};
