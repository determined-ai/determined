import { Radio } from 'antd';
import { RadioChangeEvent } from 'antd/lib/radio';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Icon from 'components/Icon';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import ResourcePoolCard from 'components/ResourcePoolCard';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { Indicator } from 'components/Spinner';
import { defaultRowClassName, getPaginationConfig, isAlternativeAction } from 'components/Table';
import Agents, { useFetchAgents } from 'contexts/Agents';
import ClusterOverview, { agentsToOverview } from 'contexts/ClusterOverview';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { columns as defaultColumns } from 'pages/Cluster.table';
import { getResourcePools } from 'services/api';
import { ShirtSize } from 'themes';
import { ResourcePool, ResourceState, ResourceType } from 'types';
import { getSlotContainerStates } from 'utils/cluster';

import css from './Cluster.module.scss';

enum View {
  List,
  Grid
}

const STORAGE_PATH = 'cluster';
const VIEW_CHOICE_KEY = 'view-choice';

const Cluster: React.FC = () => {
  const storage = useStorage(STORAGE_PATH);
  const initView = storage.getWithDefault(VIEW_CHOICE_KEY, View.Grid);
  const agents = Agents.useStateContext();
  const overview = ClusterOverview.useStateContext();
  const [ rpDetail, setRpDetail ] = useState<ResourcePool>();
  const [ selectedView, setSelectedView ] = useState<View>(initView);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ canceler ] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);

  const fetchResourcePools = useCallback(async () => {
    const resourcePools = await getResourcePools({});
    setResourcePools(resourcePools);
  }, []);

  const fetchAll = useCallback(() => {
    fetchAgents();
    fetchResourcePools();
  }, [ fetchAgents, fetchResourcePools ]);

  usePolling(fetchAll, { delay: 10000 });

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
    return getSlotContainerStates(agents.data || [], ResourceType.GPU);
  }, [ agents.data ]);

  const getTotalGpuSlots = useCallback((resPoolName: string) => {
    if (!agents.hasLoaded || !agents.data) return 0;
    const resPoolAgents = agents.data.filter(agent => agent.resourcePool === resPoolName);
    const overview = agentsToOverview(resPoolAgents);
    return overview.GPU.total;
  }, [ agents ]);

  const columns = useMemo(() => {

    const descriptionRender = (_: unknown, record: ResourcePool): React.ReactNode =>
      <div className={css.descriptionColumn}>{record.description}</div>;

    const slotsBarRender = (_:unknown, record: ResourcePool): React.ReactNode => {
      const containerStates: ResourceState[] =
          getSlotContainerStates(agents.data || [], ResourceType.GPU, record.name);

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
  }, [ agents.data, getTotalGpuSlots ]);

  const hideModal = useCallback(() => setRpDetail(undefined), []);

  const onChange = useCallback((e: RadioChangeEvent) => {
    const view = e.target.value as View;
    storage.set(VIEW_CHOICE_KEY, view);
    setSelectedView(view);
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
          <OverviewStats title="Connected Agents">
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
          resourceStates={gpuSlotStates}
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
            {resourcePools.map((rp, idx) => {
              return <ResourcePoolCard
                gpuContainerStates={
                  getSlotContainerStates(agents.data || [], ResourceType.GPU, rp.name)
                }
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
    </Page>
  );
};

export default Cluster;
