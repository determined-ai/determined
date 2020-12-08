import React, { useMemo } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message from 'components/Message';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import { ShirtSize } from 'themes';
import { Resource } from 'types';
import { categorize } from 'utils/data';

const Cluster: React.FC = () => {
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
          7/300 {/* TODO: blocked on resource pools API */}
        </OverviewStats>
      </Grid>
    </Page>
  );
};

export default Cluster;
