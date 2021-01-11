import React, { useCallback, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Message from 'components/Message';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import ResourcePoolCard from 'components/ResourcePoolCard';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SlotAllocationBar from 'components/SlotAllocationBar';
import Spinner, { Indicator } from 'components/Spinner';
import { defaultRowClassName, getPaginationConfig, isAlternativeAction } from 'components/Table';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';
import { columns } from 'pages/HGICluster.table';
import { getResourcePools } from 'services/api';
import { ShirtSize } from 'themes';
import { Resource } from 'types';
import { ResourcePool } from 'types/ResourcePool';
import { getSlotContainerStates } from 'utils/cluster';
import { categorize } from 'utils/data';

const resourcePools = getResourcePools();

const HGICluster: React.FC = () => {
  const agents = Agents.useStateContext();
  const overview = ClusterOverview.useStateContext();
  const [ detailVisible, setDetailVisible ] = useState(false);

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

  const slotContainerStates = getSlotContainerStates(agents.data || []);

  const toggleModal = useCallback(
    () => {
      setDetailVisible((cur: boolean) => !cur);
    },
    [],
  );

  const handleTableRow = useCallback((record: ResourcePool) => {
    const handleClick = (event: React.MouseEvent) => {
      if (isAlternativeAction(event)) return;
      toggleModal();
    };
    return { onAuxClick: handleClick, onClick: handleClick };
  }, []);

  if (!agents.data) {
    return <Spinner />;
  } else if (agents.data.length === 0) {
    return <Message title="No Agents connected" />;
  } else if (availableResourceTypes.length === 0) {
    return <Message title="No Slots available" />;
  }

  return (
    <Page id="cluster" title="HGI Cluster">
      <Section hideTitle title="Overview Stats">
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
      </Section>
      <Section hideTitle title="Overall Allocation">
        <SlotAllocationBar
          resourceStates={slotContainerStates}
          showLegends
          size={ShirtSize.enormous}
          totalSlots={overview.GPU.total} />
      </Section>
      <Section title={`${resourcePools.length} Resource Pools`}>
        <Grid gap={ShirtSize.medium} minItemWidth={30} mode={GridMode.AutoFill}>
          {resourcePools.map((_, idx) => {
            const rp = resourcePools[Math.floor(
              Math.random() * resourcePools.length,
            )];
            return <ResourcePoolCard
              containerStates={getSlotContainerStates(agents.data || [], rp.name)}
              key={idx}
              rpIndex={idx} />;
          })}
        </Grid>
      </Section>
      <Section title={`${resourcePools.length} Resource Pools`}>
        <ResponsiveTable<ResourcePool>
          columns={columns}
          dataSource={resourcePools}
          loading={{
            indicator: <Indicator />,
            spinning: agents.isLoading, // TODO replace with resource pools
          }}
          pagination={getPaginationConfig(resourcePools.length, 10)} // TODO config page size
          rowClassName={defaultRowClassName({ clickable: true })}
          rowKey="batchNum"
          scroll={{ x: 1000 }}
          showSorterTooltip={false}
          size="small"
          onRow={handleTableRow}
        />
        <ResourcePoolDetails finally={toggleModal} rpIndex={0} visible={detailVisible} />
      </Section>
    </Page>
  );
};

export default HGICluster;
