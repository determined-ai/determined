import React, { useCallback, useEffect, useMemo, useState } from 'react';

import ActionDropdown, { Triggers } from 'components/ActionDropdown';
import Grid, { GridMode } from 'components/Grid';
import Icon from 'components/Icon';
import Page from 'components/Page';
import ResponsiveTable, { handleTableChange } from 'components/ResponsiveTable';
import Section from 'components/Section';
import {
  defaultRowClassName, getFullPaginationConfig,
} from 'components/Table';
import { V1SchedulerTypeToLabel } from 'constants/states';
import { useStore } from 'contexts/Store';
import { useFetchResourcePools } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { columns as defaultColumns, SCHEDULING_VAL_KEY } from 'pages/JobQueue/JobQueue.table';
import { cancelExperiment, getJobQ, killCommand, killExperiment,
  killJupyterLab, killShell, killTensorBoard } from 'services/api';
import * as Api from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { ShirtSize } from 'themes';
import { Job, JobAction, JobType, ResourcePool, RPStats } from 'types';
import { isEqual } from 'utils/data';
import { numericSorter } from 'utils/sort';
import { capitalize } from 'utils/string';

import css from './JobQueue.module.scss';
import settingsConfig, { Settings } from './JobQueue.settings';
import ManageJob from './ManageJob';
import RPStatsOverview from './RPStats';

const orderdQTypes = [ Api.V1SchedulerType.PRIORITY, Api.V1SchedulerType.KUBERNETES ];

const JobQueue: React.FC = () => {
  const { resourcePools } = useStore();
  const [ managingJob, setManagingJob ] = useState<Job>();
  const [ rpStats, setRpStats ] = useState<RPStats[]>(
    resourcePools.map(rp => ({
      resourcePool: rp.name,
      stats: { preemptibleCount: 0, queuedCount: 0, scheduledCount: 0 },
    } as RPStats)),
  );
  const [ jobs, setJobs ] = useState<Job[]>([]);
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());
  const [ selectedRp, setSelectedRp ] = useState<ResourcePool>();
  const [ ps, setPs ] = useState<{isLoading: boolean}>({ isLoading: true }); // house keeping states
  const {
    settings,
    updateSettings,
    resetSettings,
  } = useSettings<Settings>(settingsConfig);

  const fetchResourcePools = useFetchResourcePools(canceler);

  const fetchJobs = useCallback(async () => {
    if (!selectedRp?.name) return;
    try {
      const orderBy = settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
      const tJobs = await getJobQ(
        {
          limit: settings.tableLimit,
          offset: settings.tableOffset,
          orderBy,
          resourcePool: selectedRp.name,
        },
        { signal: canceler.signal },
      );
      setJobs(tJobs.jobs);
      tJobs.pagination.total && setTotal(tJobs.pagination.total);
      return tJobs;
    } catch (e) { } finally {
      setPs(cur => ({ ...cur, isLoading: false }));
    }
  }, [ selectedRp?.name, canceler, settings ]);

  const fetchAll = useCallback(async () => {
    try {
      const promises = [
        detApi.Internal.getJobQueueStats().then(stats => {
          setRpStats(stats.results.sort((a, b) => a.resourcePool.localeCompare(b.resourcePool)));
        }),
        fetchJobs(),
      ] as Promise<unknown>[];
      await Promise.all(promises);
    } catch (e) { }
  }, [ fetchJobs ]);

  usePolling(fetchAll);

  const dropDownOnTrigger = useCallback((job: Job) => {
    const triggers: Triggers<JobAction> = {
      [JobAction.Cancel]: async () => {
        switch (job.type) {
          case JobType.EXPERIMENT:
            await cancelExperiment({ experimentId: parseInt(job.entityId, 10) });
            break;
          case JobType.COMMAND:
            await killCommand({ commandId: job.entityId });
            break;
          case JobType.TENSORBOARD:
            await killTensorBoard({ commandId: job.entityId });
            break;
          case JobType.SHELL:
            await killShell({ commandId: job.entityId });
            break;
          case JobType.NOTEBOOK:
            await killJupyterLab({ commandId: job.entityId });
            break;
          default:
            return Promise.resolve();
        }
      },
      [JobAction.ManageJob]: () => setManagingJob(job),
    };
    const isFirst = job.summary.jobsAhead === 0;
    if (!isFirst) {
      triggers[JobAction.MoveToTop] = () => window.alert('TODO move top');
    }

    // if job is an experiment type add action to kill it
    if (job.type === JobType.EXPERIMENT) {
      triggers[JobAction.Kill] = async () => {
        await killExperiment({ experimentId: parseInt(job.entityId, 10) });
      };
    }

    return triggers;
  }, []);

  const hideModal = useCallback(() => setManagingJob(undefined), []);

  const columns = useMemo(() => {
    return defaultColumns.map(col => {
      switch (col.key) {
        case 'actions':
          col.render = (_, record) => {
            return (
              <div>
                <ActionDropdown<JobAction>
                  actionOrder={[
                    JobAction.ManageJob,
                    JobAction.MoveToTop,
                    JobAction.Cancel,
                    JobAction.Kill,
                  ]}
                  confirmations={{
                    [JobAction.Cancel]: { cancelText: 'Abort' },
                    [JobAction.Kill]: {},
                    [JobAction.MoveToTop]: {},
                  }}
                  id={record.name}
                  kind="job"
                  onTrigger={dropDownOnTrigger(record)}
                />
              </div>
            );
          };
          break;
        case SCHEDULING_VAL_KEY:
          switch (selectedRp?.schedulerType) {
            case Api.V1SchedulerType.PRIORITY:
            case Api.V1SchedulerType.KUBERNETES:
              col.title = 'Priority';
              col.dataIndex = 'priority';
              break;
            case Api.V1SchedulerType.FAIRSHARE:
              col.title = 'Weight';
              col.dataIndex = 'weight';
              break;
          }
          break;
        case 'jobsAhead':
          col.render = (_: unknown, record) => {
            return (
              <div className={css.centerVertically}>
                {record.summary.jobsAhead >= 0 && record.summary.jobsAhead}
                {!record.isPreemptible && <Icon name="lock" title="Not Preemtible" />}
              </div>
            );
          };
          if (selectedRp && !orderdQTypes.includes(selectedRp.schedulerType)) {
            col.sorter = undefined;
            col.title = 'Preemptible';
          } else {
            col.sorter = (a: Job, b: Job): number =>
              numericSorter(a.summary.jobsAhead, b.summary.jobsAhead);
            col.title = '#';
          }
          break;
      }
      return col;
    }).map(column => {
      column.sortOrder = null;
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ dropDownOnTrigger, selectedRp, jobs ]);

  useEffect(() => {
    if (resourcePools.length === 0) {
      setSelectedRp(undefined);
      resetSettings([ 'selectedRp' ]);
    } else if (!selectedRp) {
      let pool: ResourcePool | undefined = undefined;
      if (settings.selectedPool) {
        pool = resourcePools.find(pool => pool.name === settings.selectedPool);
      }
      if (!pool) {
        pool = resourcePools[0];
      }
      updateSettings({ selectedPool: pool.name });
      setSelectedRp(pool);
    }
  }, [ resourcePools, selectedRp, updateSettings, resetSettings, settings.selectedPool ]);

  useEffect(() => {
    fetchResourcePools();
    return () => canceler.abort();
  }, [ canceler, fetchResourcePools ]);

  useEffect(() => {
    setPs(cur => ({ ...cur, isLoading: true }));
    fetchJobs();
    return () => canceler.abort();
  }, [
    fetchJobs,
    canceler,
    settings.sortDesc,
    settings.sortKey,
    settings.tableLimit,
    settings.tableOffset,
  ]);

  useEffect(() => {
    if (!managingJob) return;
    const job = jobs.find(j => j.jobId === managingJob.jobId);
    if (!job) {
      setManagingJob(undefined);
    } else if (!isEqual(job, managingJob)) {
      setManagingJob(job);
    }
  }, [ jobs, managingJob ]);

  const rpSwitcher = useCallback((rpName: string) => {
    return () => {
      const rp = resourcePools.find(rp => rp.name === rpName) as ResourcePool;
      setSelectedRp(rp);
      updateSettings({ selectedPool: rp.name });
    };
  }, [ resourcePools, updateSettings ]);

  // table title using selectedRp and schedulerType from list of resource pools
  const tableTitle = useMemo(() => {
    if (!selectedRp) return '';
    const schedulerType = V1SchedulerTypeToLabel[selectedRp.schedulerType];
    return (
      <div>
        {`${capitalize(selectedRp.name)} (${schedulerType.toLowerCase()}) `}
        <Icon name="info" title={`Job Queue for resource pool "${selectedRp.name}"`} />
      </div>
    );
  }, [ selectedRp ]);

  return (
    <Page className={css.base} id="jobs" title="Job Queues by Resource Pool">
      <Section hideTitle title="Resource Pools">
        <Grid gap={ShirtSize.medium} minItemWidth={150} mode={GridMode.AutoFill}>
          {rpStats.map((stats, idx) => {
            let onClick = undefined;
            const isTargetRp = stats.resourcePool === selectedRp?.name;
            if (!isTargetRp) {
              onClick = rpSwitcher(stats.resourcePool);
            }
            return (
              <RPStatsOverview
                focused={isTargetRp}
                key={idx}
                stats={stats}
                onClick={onClick}
              />
            );
          })}
        </Grid>
      </Section>
      <Section title={tableTitle}>
        <ResponsiveTable<Job>
          columns={columns}
          dataSource={jobs}
          loading={ps.isLoading}
          pagination={getFullPaginationConfig({
            limit: settings.tableLimit,
            offset: settings.tableOffset,
          }, total)}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="jobId"
          scroll={{ x: 1000 }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange(columns, settings, updateSettings)}
        />
      </Section>
      {!!managingJob && !!selectedRp && (
        <ManageJob
          job={managingJob}
          schedulerType={selectedRp.schedulerType}
          selectedRPStats={rpStats.find(rp => rp.resourcePool === selectedRp.name) as RPStats}
          onFinish={hideModal}
        />
      )}

    </Page>
  );
};

export default JobQueue;
