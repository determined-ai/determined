import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import OverviewStats from 'components/OverviewStats';
import ResourcePoolCard from 'components/ResourcePoolCard';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { defaultRowClassName, getPaginationConfig, isAlternativeAction } from 'components/Table';
import Agents from 'contexts/Agents';
import ClusterOverviewContext, { agentsToOverview } from 'contexts/ClusterOverview';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { columns as defaultColumns } from 'pages/Cluster/ClusterOverview.table';
import { getResourcePools } from 'services/api';
import { ShirtSize } from 'themes';
import { ResourcePool, ResourceState, ResourceType } from 'types';
import { getSlotContainerStates } from 'utils/cluster';

import css from './ClusterOverview.module.scss';

const STORAGE_PATH = 'cluster';
const VIEW_CHOICE_KEY = 'view-choice';

const ClusterOverview: React.FC = () => {
  const storage = useStorage(STORAGE_PATH);
  const initView = storage.getWithDefault(VIEW_CHOICE_KEY, GridListView.Grid);
  const agents = Agents.useStateContext();
  const overview = ClusterOverviewContext.useStateContext();
  const [ rpDetail, setRpDetail ] = useState<ResourcePool>();
  const [ selectedView, setSelectedView ] = useState<GridListView>(initView);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ canceler ] = useState(new AbortController());

  const fetchResourcePools = useCallback(async () => {
    const resourcePools = await getResourcePools({});
    setResourcePools(resourcePools);
  }, []);

  usePolling(fetchResourcePools, { interval: 10000 });

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

  const gpuSlotStates = useMemo(() => {
    return getSlotContainerStates(agents || [], ResourceType.GPU);
  }, [ agents ]);

  const getTotalGpuSlots = useCallback((resPoolName: string) => {
    if (!agents) return 0;
    const resPoolAgents = agents.filter(agent => agent.resourcePool === resPoolName);
    const overview = agentsToOverview(resPoolAgents);
    return overview.GPU.total;
  }, [ agents ]);

  const columns = useMemo(() => {

    const descriptionRender = (_: unknown, record: ResourcePool): React.ReactNode =>
      <div className={css.descriptionColumn}>{record.description}</div>;

    const slotsBarRender = (_:unknown, record: ResourcePool): React.ReactNode => {
      const containerStates: ResourceState[] =
        getSlotContainerStates(agents || [], ResourceType.GPU, record.name);

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
      if (column.key === 'chart') column.render = slotsBarRender;
      return column;
    });

    return newColumns;
  }, [ agents, getTotalGpuSlots ]);

  const hideModal = useCallback(() => setRpDetail(undefined), []);

  const handleRadioChange = useCallback((value: GridListView) => {
    storage.set(VIEW_CHOICE_KEY, value);
    setSelectedView(value);
  }, [ storage ]);

  const handleTableRow = useCallback((record: ResourcePool) => {
    const handleClick = (event: React.MouseEvent) => {
      if (isAlternativeAction(event)) return;
      setRpDetail(record);
    };
    return { onAuxClick: handleClick, onClick: handleClick };
  }, []);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <>
      <Section hideTitle title="Overview Stats">
        <Grid gap={ShirtSize.medium} minItemWidth={15} mode={GridMode.AutoFill}>
          <OverviewStats title="Connected Agents">
            {agents ? agents.length : '?'}
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
          resourceStates={gpuSlotStates}
          showLegends
          size={ShirtSize.enormous}
          totalSlots={overview.GPU.total} />
      </Section>
      <Section
        options={<GridListRadioGroup value={selectedView} onChange={handleRadioChange} />}
        title={`${resourcePools.length} Resource Pools`}
      >
        {selectedView === GridListView.Grid &&
        <Grid gap={ShirtSize.medium} minItemWidth={30} mode={GridMode.AutoFill}>
          {resourcePools.map((rp, idx) => {
            return <ResourcePoolCard
              gpuContainerStates={
                getSlotContainerStates(agents || [], ResourceType.GPU, rp.name)
              }
              key={idx}
              resourcePool={rp}
              totalGpuSlots={getTotalGpuSlots(rp.name)} />;
          })}
        </Grid>
        }
        {selectedView === GridListView.List &&
        <ResponsiveTable<ResourcePool>
          columns={columns}
          dataSource={resourcePools}
          loading={!agents} // TODO replace with resource pools
          pagination={getPaginationConfig(resourcePools.length, 10)}
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
    </>
  );
};

export default ClusterOverview;
