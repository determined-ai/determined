import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import Message, { MessageType } from 'components/Message';
import OverviewStats from 'components/OverviewStats';
import ResourcePoolCard from 'components/ResourcePoolCard';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import SlotAllocationBar from 'components/SlotAllocationBar';
import {
  defaultRowClassName, getFullPaginationConfig, isAlternativeAction, MINIMUM_PAGE_SIZE,
} from 'components/Table';
import { agentsToOverview, initResourceTally, useStore } from 'contexts/Store';
import { useFetchAgents, useFetchResourcePools } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { columns as defaultColumns } from 'pages/Cluster/ClusterOverview.table';
import { ShirtSize } from 'themes';
import {
  ClusterOverviewResource, Pagination, ResourcePool, ResourceState, ResourceType,
} from 'types';
import { getSlotContainerStates } from 'utils/cluster';

import css from './ClusterOverview.module.scss';

const STORAGE_PATH = 'cluster';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';
const VIEW_CHOICE_KEY = 'view-choice';

const defaultSorter = { descend: false, key: 'name' };

const ClusterOverview: React.FC = () => {
  const storage = useStorage(STORAGE_PATH);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initView = storage.get<GridListView>(VIEW_CHOICE_KEY);
  const { agents, cluster: overview, resourcePools } = useStore();
  const [ rpDetail, setRpDetail ] = useState<ResourcePool>();
  const [ selectedView, setSelectedView ] = useState<GridListView>(() => {
    if (initView && Object.values(GridListView).includes(initView as GridListView)) return initView;
    return GridListView.Grid;
  });
  const [ sorter, setSorter ] = useState(initSorter);
  const [ pagination, setPagination ] = useState<Pagination>({ limit: initLimit, offset: 0 });
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);
  const fetchResourcePools = useFetchResourcePools(canceler);

  useEffect(() => {
    setTotal(resourcePools.length || 0);
  }, [ resourcePools ]);

  usePolling(fetchResourcePools, { interval: 10000 });

  const auxContainers = useMemo(() => {
    const tally = {
      running: 0,
      total: 0,
    };
    resourcePools.forEach(rp => {
      tally.total += rp.auxContainerCapacity;
      tally.running += rp.auxContainersRunning;
    });
    return tally;
  }, [ resourcePools ]);

  const cudaSlotStates = useMemo(() => {
    return getSlotContainerStates(agents || [], ResourceType.CUDA);
  }, [ agents ]);

  const rocmSlotStates = useMemo(() => {
    return getSlotContainerStates(agents || [], ResourceType.ROCM);
  }, [ agents ]);

  const cpuSlotStates = useMemo(() => {
    return getSlotContainerStates(agents || [], ResourceType.CPU);
  }, [ agents ]);

  const [ cudaTotalSlots, rocmTotalSlots ] = useMemo(() => {
    return resourcePools.reduce((acc, pool) => {
      let index;
      switch (pool.slotType) {
        case ResourceType.CUDA:
          index = 0;
          break;
        case ResourceType.ROCM:
          index = 1;
          break;
        default:
          index = undefined;
      }
      if (index === undefined) return acc;
      acc[index] += pool.maxAgents * (pool.slotsPerAgent ?? 0);
      return acc;
    }, [ 0, 0 ]);
  }, [ resourcePools ]);

  const getSlotTypeOverview = useCallback((
    resPoolName: string,
    resType: ResourceType,
  ): ClusterOverviewResource => {
    if (!agents || resType === ResourceType.UNSPECIFIED) return initResourceTally;
    const resPoolAgents = agents.filter(agent => agent.resourcePool === resPoolName);
    const overview = agentsToOverview(resPoolAgents);
    return overview[resType];
  }, [ agents ]);

  const columns = useMemo(() => {

    const descriptionRender = (_: unknown, record: ResourcePool): React.ReactNode =>
      <div className={css.descriptionColumn}>{record.description}</div>;

    const slotsBarRender = (_:unknown, rp: ResourcePool): React.ReactNode => {
      const containerStates: ResourceState[] =
        getSlotContainerStates(agents || [], rp.slotType, rp.name);
      const totalSlots = getSlotTypeOverview(rp.name, rp.slotType).total;

      if (totalSlots === 0) return null;
      return (
        <SlotAllocationBar
          className={css.chartColumn}
          hideHeader
          resourceStates={containerStates}
          title={rp.slotType}
          totalSlots={totalSlots}
        />
      );
    };

    const newColumns = [ ...defaultColumns ].map(column => {
      column.sortOrder = null;
      if (column.key === 'description') column.render = descriptionRender;
      if (column.key === 'chart') column.render = slotsBarRender;
      if (column.key === sorter.key) {
        column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      }
      return column;
    });

    return newColumns;
  }, [ agents, getSlotTypeOverview, sorter ]);

  const hideModal = useCallback(() => setRpDetail(undefined), []);

  const handleRadioChange = useCallback((value: GridListView) => {
    storage.set(VIEW_CHOICE_KEY, value);
    setSelectedView(value);
  }, [ storage ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<ResourcePool>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    storage.set(STORAGE_SORTER_KEY, { descend: order === 'descend', key: columnKey as string });
    setSorter({ descend: order === 'descend', key: columnKey as string });

    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPagination(prev => ({
      ...prev,
      limit: tablePagination.pageSize,
      offset: (tablePagination.current - 1) * tablePagination.pageSize,
    }));
  }, [ columns, storage ]);

  const handleTableRow = useCallback((record: ResourcePool) => {
    const handleClick = (event: React.MouseEvent) => {
      if (isAlternativeAction(event)) return;
      setRpDetail(record);
    };
    return { onAuxClick: handleClick, onClick: handleClick };
  }, []);

  useEffect(() => {
    fetchAgents();

    return () => canceler.abort();
  }, [ canceler, fetchAgents ]);

  return (
    <>
      <Section hideTitle title="Overview Stats">
        <Grid gap={ShirtSize.medium} minItemWidth={150} mode={GridMode.AutoFill}>
          <OverviewStats title="Connected Agents">
            {agents ? agents.length : '?'}
          </OverviewStats>
          {cudaTotalSlots ? (
            <OverviewStats title="CUDA Slots Allocated">
              {overview.CUDA.total - overview.CUDA.available} <small>/ {cudaTotalSlots}</small>
            </OverviewStats>
          ) : null}
          {rocmTotalSlots ? (
            <OverviewStats title="ROCm Slots Allocated">
              {overview.ROCM.total - overview.ROCM.available} <small>/ {rocmTotalSlots}</small>
            </OverviewStats>
          ) : null}
          {overview.CPU.total ? (
            <OverviewStats title="CPU Slots Allocated">
              {overview.CPU.total - overview.CPU.available} <small>/ {overview.CPU.total}</small>
            </OverviewStats>
          ) : null}
          {auxContainers.total ? (
            <OverviewStats title="Aux Containers Running">
              {auxContainers.running} <small>/ {auxContainers.total}</small>
            </OverviewStats>
          ) : null}
        </Grid>
      </Section>
      <Section hideTitle title="Overall Allocation">
        {cudaTotalSlots + rocmTotalSlots + overview.CPU.total === 0 ? (
          <Message title="No connected agents." type={MessageType.Empty} />
        ) : null }
        {cudaTotalSlots > 0 && (
          <SlotAllocationBar
            resourceStates={cudaSlotStates}
            showLegends
            size={ShirtSize.enormous}
            title={`Compute (${ResourceType.CUDA})`}
            totalSlots={cudaTotalSlots}
          />
        )}
        {rocmTotalSlots > 0 && (
          <SlotAllocationBar
            resourceStates={rocmSlotStates}
            showLegends
            size={ShirtSize.enormous}
            title={`Compute (${ResourceType.ROCM})`}
            totalSlots={rocmTotalSlots}
          />
        )}
        {overview.CPU.total > 0 && (
          <SlotAllocationBar
            resourceStates={cpuSlotStates}
            showLegends
            size={ShirtSize.enormous}
            title={`Compute (${ResourceType.CPU})`}
            totalSlots={overview.CPU.total}
          />
        )}
      </Section>
      <Section
        options={<GridListRadioGroup value={selectedView} onChange={handleRadioChange} />}
        title={`${resourcePools.length} Resource Pools`}>
        {selectedView === GridListView.Grid && (
          <Grid gap={ShirtSize.medium} minItemWidth={300} mode={GridMode.AutoFill}>
            {resourcePools.map((rp, idx) => (
              <ResourcePoolCard
                computeContainerStates={
                  getSlotContainerStates(agents || [], rp.slotType, rp.name)
                }
                key={idx}
                resourcePool={rp}
                resourceType={rp.slotType}
                totalComputeSlots={rp.maxAgents * (rp.slotsPerAgent ?? 0)}
              />
            ))}
          </Grid>
        )}
        {selectedView === GridListView.List && (
          <ResponsiveTable<ResourcePool>
            columns={columns}
            dataSource={resourcePools}
            loading={!agents} // TODO replace with resource pools
            pagination={getFullPaginationConfig(pagination, total)}
            rowClassName={defaultRowClassName({ clickable: true })}
            rowKey="name"
            scroll={{ x: 1000 }}
            showSorterTooltip={false}
            size="small"
            onChange={handleTableChange}
            onRow={handleTableRow}
          />
        )}
      </Section>
      {!!rpDetail && (
        <ResourcePoolDetails
          finally={hideModal}
          resourcePool={rpDetail}
          visible={!!rpDetail}
        />
      )}
    </>
  );
};

export default ClusterOverview;
