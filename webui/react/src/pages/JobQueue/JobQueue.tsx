import { Modal, Tooltip } from 'antd';
import { SorterResult } from 'antd/es/table/interface';
import React, { ReactNode, useCallback, useEffect, useMemo, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Link from 'components/Link';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import {
  defaultRowClassName, getFullPaginationConfig, MINIMUM_PAGE_SIZE,
} from 'components/Table';
import { useStore } from 'contexts/Store';
import { useFetchAgents } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { columns as defaultColumns, JobTypeRenderer } from 'pages/JobQueue/JobQueue.table';
import { getJobQ, getResourcePools } from 'services/api';
import { detApi } from 'services/apiConfig';
import * as decoder from 'services/decoder';
import { ShirtSize } from 'themes';
import {
  Job, Pagination, ResourcePool, RPStats,
} from 'types';
import { truncate } from 'utils/string';

import css from './JobQueue.module.scss';
import ManageJob from './ManageJob';

const STORAGE_PATH = 'job';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const defaultSorter = { descend: false, key: 'name' };

const JobQueue: React.FC = () => {
  const storage = useStorage(STORAGE_PATH);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const { agents, resourcePools } = useStore();
  const [ managingJob, setManagingJob ] = useState<Job>();
  const [ rpStats, setRpStats ] = useState<RPStats[]>([]);
  const [ jobs, setJobs ] = useState<Job[]>([]);
  const [ sorter, setSorter ] = useState(initSorter);
  const [ pagination, setPagination ] = useState<Pagination>({ limit: initLimit, offset: 0 });
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());
  const [ selectedRp, setSelectedRp ] = useState<string>('default');

  const fetchAgents = useFetchAgents(canceler);

  const fetchJobs = useCallback(async () => {
    try {
      const tJobs = await getJobQ({ resourcePool: selectedRp, ...pagination });
      setJobs(tJobs.jobs);
      setPagination(decoder.mapV1Pagination(tJobs.pagination));
      return tJobs;
    } catch (e) {}
  }, [ selectedRp ]);

  // OPTIMIZE
  const fetchAll = useCallback(async () => {
    try {
      // const resourcePools = await getResourcePools({});
      // TODO bring in stats api, set api to return all on no filter
      const stats = await detApi.Jobs.determinedGetJobQueueStats();
      // const stats = resourcePools.map(rp => ({resourcePool: rp.name, stats: {}}))
      setRpStats(stats.results);
      // setTotal(resourcePools.length || 0);
      await fetchJobs();
    } catch (e) {}
  }, []);

  usePolling(fetchAll, { interval: 5000 });

  const handleManageJob = useCallback((job: Job) => {
    return () => setManagingJob(job);
  }, []);

  const hideModal = useCallback(() => setManagingJob(undefined), []);

  const columns = useMemo(() => {
    return defaultColumns.map(col => {
      if (col.key === 'actions') {
        const renderer: JobTypeRenderer = (_, record) => {
          const cell = (
            <ul>
              <li>
                <Link onClick={handleManageJob(record)}>Manage</Link>
              </li>
            </ul>
          );
          return cell;
        };
        col.render = renderer;
      }
      return col;
    });
  }, []);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<Job>;
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

  // const handleTableRow = useCallback((record: Job) => {
  //   const handleClick = (event: React.MouseEvent) => {
  //     window.alert('visiting' + record.type + record.entityId);
  //     // if (isAlternativeAction(event)) return;
  //     // setRpDetail(record);
  //   };
  //   return { onAuxClick: handleClick, onClick: handleClick };
  // }, []);

  useEffect(() => {
    fetchAgents();

    return () => canceler.abort();
  }, [ canceler, fetchAgents ]);

  const rpSwitcher = useCallback((rpName: string) => {
    return () => setSelectedRp(rpName);
  }, []);

  return (
    <Page id="jobs" title="Jobs">
      <Section title="Job Queue By Resource Pool">
        <Grid gap={ShirtSize.medium} minItemWidth={150} mode={GridMode.AutoFill}>
          {rpStats.map((stats, idx) => {
            let onClick = undefined;
            const isTargetRp = stats.resourcePool === selectedRp;
            if (!isTargetRp) {
              onClick = rpSwitcher(stats.resourcePool);
            }
            return <OverviewStats
              focused={isTargetRp}
              key={idx}
              title={stats.resourcePool}
              onClick={onClick}
            >
              <Tooltip title="Scheduled Jobs">
                {stats.stats.scheduledCount}
              </Tooltip>
              /
              <Tooltip title="All Jobs">
                {stats.stats.queuedCount + stats.stats.scheduledCount}
              </Tooltip>
            </OverviewStats>;
          })}
        </Grid>
      </Section>
      <Section hideTitle title={`Queue: ${selectedRp}`}>
        <ResponsiveTable<Job>
          columns={columns}
          dataSource={jobs}
          loading={!agents} // TODO replace with resource pools
          pagination={getFullPaginationConfig(pagination, total)}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="name"
          scroll={{ x: 1000 }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange}
          // onRow={handleTableRow}
        />
      </Section>
      {!!managingJob &&
        <ManageJob job={managingJob} onFinish={hideModal} />
      }

    </Page>
  );
};

export default JobQueue;
