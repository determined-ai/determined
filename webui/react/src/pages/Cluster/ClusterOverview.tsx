import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
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
import { percent } from 'shared/utils/number';
import { ShirtSize } from 'themes';
import {
  ClusterOverviewResource,
  ClusterOverview as Overview, ResourcePool, ResourceState, ResourceType,
} from 'types';
import { getSlotContainerStates } from 'utils/cluster';

import { Pagination } from '../../shared/types';

import { ClusterOverallBar } from './ClusterOverallBar';
import { ClusterOverallStats } from './ClusterOverallStats';
import css from './ClusterOverview.module.scss';

const STORAGE_PATH = 'cluster';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';
const VIEW_CHOICE_KEY = 'view-choice';

const defaultSorter = { descend: false, key: 'name' };

/**
 * maximum theoretcial capacity of the resource pool in terms of the advertised
 * compute slot type.
 * @param pool resource pool
 */
export const maxPoolSlotCapacity = (pool: ResourcePool): number => {
  if (pool.maxAgents > 0 && pool.slotsPerAgent && pool.slotsPerAgent > 0)
    return pool.maxAgents * pool.slotsPerAgent;
  // on-premise deployments don't have dynamic agents and we don't know how many
  // agents might connect.
  return pool.slotsAvailable;
};

export const clusterStatusText = (
  overview: Overview,
  pools: ResourcePool[],
): string | undefined => {
  if (overview[ResourceType.ALL].allocation === 0) return undefined;
  const totalSlots = pools.reduce((totalSlots, currentPool) => {
    return totalSlots + maxPoolSlotCapacity(currentPool);
  }, 0);
  if (totalSlots === 0) return `${overview[ResourceType.ALL].allocation}%`;
  return `${percent((overview[ResourceType.ALL].total - overview[ResourceType.ALL].available)
        / totalSlots)}%`;
};

const ClusterOverview: React.FC = () => {
  const storage = useStorage(STORAGE_PATH);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initView = storage.get<GridListView>(VIEW_CHOICE_KEY);
  const { agents, resourcePools } = useStore();
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
      if (column.key === 'slotsAvailable') {
        column.render = (_: unknown, rp: ResourcePool): React.ReactNode => {
          return maxPoolSlotCapacity(rp);
        };
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
      <ClusterOverallStats />
      <ClusterOverallBar />
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
                totalComputeSlots={maxPoolSlotCapacity(rp)}
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
