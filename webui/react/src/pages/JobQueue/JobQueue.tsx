import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import {
  defaultRowClassName, getFullPaginationConfig, isAlternativeAction, MINIMUM_PAGE_SIZE,
} from 'components/Table';
import { useStore } from 'contexts/Store';
import { useFetchAgents } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { columns as defaultColumns } from 'pages/JobQueue/JobQueue.table';
import { getResourcePools } from 'services/api';
import { ShirtSize } from 'themes';
import {
  Pagination, ResourcePool
} from 'types';

import css from './JobQueue.module.scss';

const STORAGE_PATH = 'job';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const defaultSorter = { descend: false, key: 'name' };

const JobQueue: React.FC = () => {
  const storage = useStorage(STORAGE_PATH);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const { agents } = useStore();
  const [ rpDetail, setRpDetail ] = useState<ResourcePool>();
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ sorter, setSorter ] = useState(initSorter);
  const [ pagination, setPagination ] = useState<Pagination>({ limit: initLimit, offset: 0 });
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);

  const fetchResourcePools = useCallback(async () => {
    try {
      const resourcePools = await getResourcePools({});
      setResourcePools(resourcePools);
      setTotal(resourcePools.length || 0);
    } catch (e) {}
  }, []);

  usePolling(fetchResourcePools, { interval: 10000 });

  const columns = useMemo(() => {
    sorter;
    return defaultColumns
  }, []);

  const hideModal = useCallback(() => setRpDetail(undefined), []);

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
          <div>rp</div>
          <div>rp</div>
          <div>rp</div>
        </Grid>
      </Section>
      <Section hideTitle title="Overall Allocation">
        <div>my table</div>
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

export default JobQueue;
