import { Radio } from 'antd';
import { RadioChangeEvent } from 'antd/lib/radio';
import React, { useCallback, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Icon from 'components/Icon';
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
import ClusterOverview, { agentsToOverview } from 'contexts/ClusterOverview';
import usePolling from 'hooks/usePolling';
import { columns as defaultColumns } from 'pages/Cluster.table';
import { getResourcePools } from 'services/api';
import { ShirtSize } from 'themes';
import { Resource, ResourcePool, ResourceState } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import { categorize } from 'utils/data';

import css from './Cluster.module.scss';

enum View {
  List,
  Grid
}

const Cluster: React.FC = () => {
  const agents = Agents.useStateContext();
  const overview = ClusterOverview.useStateContext();
  const [ rpDetail, setRpDetail ] = useState<ResourcePool>();
  const [ selectedView, setSelectedView ] = useState<View>(View.Grid);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);

  const pollResourcePools = useCallback(async () => {
    const resourcePools = await getResourcePools({});
    setResourcePools(resourcePools);
  }, []);

  usePolling(pollResourcePools, { delay: 10000 });

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
  }, [ resourcePools ]);

  const slotContainerStates = getSlotContainerStates(agents.data || []);

  const getTotalGpuSlots = useCallback((resPoolName: string) => {
    if (!agents.hasLoaded || !agents.data) return 0;
    const resPoolAgents = agents.data.filter(agent => agent.resourcePool === resPoolName);
    const overview = agentsToOverview(resPoolAgents);
    return overview.GPU.total;
  }, [ agents ]);

  const columns = useMemo(() => {

    const descriptionRender = (_: unknown, record: ResourcePool): React.ReactNode =>
      <div className={css.descriptionColumn}>{record.description}</div>;

    const chartRender = (_:unknown, record: ResourcePool): React.ReactNode => {
      const containerStates: ResourceState[] =
          getSlotContainerStates(agents.data || [], record.name);

      const totalGpuSlots = getTotalGpuSlots(record.name);

      if (totalGpuSlots === 0) return null;
      return <SlotAllocationBar
        className={css.chartColumn}
        hideHeader
        resourceStates={containerStates}
        totalSlots={totalGpuSlots} />;

    };

    const newColumns = [ ...defaultColumns ].map(column => {
      if (column.key === 'description') column.render = descriptionRender;
      if (column.key === 'chart') column.render = chartRender;
      return column;
    });

    return newColumns;
  }, [ agents.data, getTotalGpuSlots ]);

  const hideModal = useCallback(() => setRpDetail(undefined), []);

  const onChange = useCallback((e: RadioChangeEvent) => {
    setSelectedView(e.target.value as View);
  }, []);

  const handleTableRow = useCallback((record: ResourcePool) => {
    const handleClick = (event: React.MouseEvent) => {
      if (isAlternativeAction(event)) return;
      setRpDetail(record);
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

  const viewOptions = (
    <Radio.Group value={selectedView} onChange={onChange}>
      <Radio.Button value={View.Grid}>
        <Icon name="grid" size="large" title="Card View" />
      </Radio.Button>
      <Radio.Button value={View.List}>
        <Icon name="list" size="large" title="Table View" />
      </Radio.Button>
    </Radio.Group>
  );

  return (
    <Page className={css.base} id="cluster" title="Cluster">
      <Section hideTitle title="Overview Stats">
        <Grid gap={ShirtSize.medium} minItemWidth={15} mode={GridMode.AutoFill}>
          <OverviewStats title="Number of Agents">
            {agents.data ? agents.data.length : '?'}
          </OverviewStats>
          {overview.GPU.total ?
            <OverviewStats title="GPU Slots Allocated">
              {overview.GPU.total - overview.GPU.available} / {overview.GPU.total}
            </OverviewStats>: null
          }
          {overview.CPU.total ?
            <OverviewStats title="CPU Slots Allocated">
              {overview.CPU.total - overview.CPU.available} / {overview.CPU.total}
            </OverviewStats> : null
          }
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
      <Section
        options={viewOptions}
        title={`${resourcePools.length} Resource Pools`}
      >
        {selectedView === View.Grid &&
          <Grid gap={ShirtSize.medium} minItemWidth={30} mode={GridMode.AutoFill}>
            {resourcePools.map((_, idx) => {
              const rp = resourcePools[Math.floor(
                Math.random() * resourcePools.length,
              )];
              return <ResourcePoolCard
                containerStates={getSlotContainerStates(agents.data || [], rp.name)}
                key={idx}
                resourcePool={rp}
                totalGpuSlots={getTotalGpuSlots(rp.name)} />;
            })}
          </Grid>
        }
        {selectedView === View.List &&
          <ResponsiveTable<ResourcePool>
            columns={columns}
            dataSource={resourcePools}
            loading={{
              indicator: <Indicator />,
              spinning: agents.isLoading, // TODO replace with resource pools
            }}
            pagination={getPaginationConfig(resourcePools.length, 10)} // TODO config page size
            rowClassName={defaultRowClassName({ clickable: true })}
            rowKey="name"
            scroll={{ x: 1000 }}
            showSorterTooltip={false}
            size="small"
            onRow={handleTableRow}
          />
        }
      </Section>
      {!!rpDetail &&
            <ResourcePoolDetails
              finally={hideModal}
              resourcePool={rpDetail}
              visible={!!rpDetail} />
      }
    </Page>
  );
};

export default Cluster;
