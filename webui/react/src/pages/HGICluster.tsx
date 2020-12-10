import React, { useMemo } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message from 'components/Message';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import SlotAllocationBar from 'components/SlotAllocationBar';
import Spinner from 'components/Spinner';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import { getResourcePools } from 'services/api';
import { ShirtSize } from 'themes';
import { Resource, ResourceState } from 'types';
import { categorize } from 'utils/data';

const resourcePools = getResourcePools();

const HGICluster: React.FC = () => {
  const agents = Agents.useStateContext();
  const overview = ClusterOverview.useStateContext();

  const availableResources = useMemo(() => {
    if (!agents.data) return {};
    const resourceList = agents.data
      .map(agent => agent.resources)
      .flat()
      .filter(resource => resource.enabled);
    return categorize(resourceList, (res: Resource) => res.type);
  }, [ agents ]);

  const availableResourceTypes = Object.keys(availableResources);

  const cpuContainers = useMemo(() => {
    const tally = {
      running: 0,
      total: 0,
    };
    resourcePools.forEach(rp => {
      tally.total += rp.cpuContainerCapacity;
      tally.running += rp.cpuContainersRunning;
    });
    return tally;
  }, [ ]);

  const slotContainerStates = agents.data?.map(agent => agent.resources)
    .reduce((acc, cur) => {
      acc.push(...cur);
      return acc;
    }, [])
    .filter(res => res.enabled && res.container)
    .map(res => res.container?.state) as ResourceState[];

  if (!agents.data) {
    return <Spinner />;
  } else if (agents.data.length === 0) {
    return <Message title="No Agents connected" />;
  } else if (availableResourceTypes.length === 0) {
    return <Message title="No Slots available" />;
  }

  return (
    <Page id="cluster" title="Cluster">
      <Grid gap={ShirtSize.medium} minItemWidth={15} mode={GridMode.AutoFill}>
        <OverviewStats title="Number of Agents">
          {agents.data ? agents.data.length : '?'}
        </OverviewStats>
        <OverviewStats title="GPU Slots Allocated">
          {overview.GPU.total - overview.GPU.available} / {overview.GPU.total}
        </OverviewStats>
        <OverviewStats title="CPU Containers Running">
          {cpuContainers.running}/{cpuContainers.total}
        </OverviewStats>
      </Grid>
      <SlotAllocationBar
        resourceStates={slotContainerStates}
        showLegends
        totalSlots={overview.GPU.total} />
    </Page>
  );
};

export default HGICluster;
