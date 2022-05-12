import React, { useCallback, useEffect, useMemo, useState } from 'react';

import ActionDropdown, { Triggers } from 'components/ActionDropdown';
import Grid, { GridMode } from 'components/Grid';
import Icon from 'components/Icon';
import Page from 'components/Page';
import ResponsiveTable, { handleTableChange } from 'components/ResponsiveTable';
import Section from 'components/Section';
import { checkmarkRenderer, defaultRowClassName, getFullPaginationConfig } from 'components/Table';
import { V1SchedulerTypeToLabel } from 'constants/states';
import { useStore } from 'contexts/Store';
import { useFetchResourcePools } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { columns as defaultColumns, SCHEDULING_VAL_KEY } from 'pages/JobQueue/JobQueue.table';
import { cancelExperiment, getJobQ, getJobQStats, killCommand, killExperiment,
  killJupyterLab, killShell, killTensorBoard } from 'services/api';
import * as Api from 'services/api-ts-sdk';
import { GetJobsResponse } from 'services/types';
import { isEqual } from 'shared/utils/data';
import { capitalize } from 'shared/utils/string';
import { ShirtSize } from 'themes';
import { Job, JobAction, JobType, ResourcePool, RPStats } from 'types';
import handleError from 'utils/error';
import { canManageJob, moveJobToPosition, orderedSchedulers,
  unsupportedQPosSchedulers } from 'utils/job';
import { numericSorter } from 'utils/sort';

import { ErrorLevel, ErrorType } from '../../shared/utils/error';

import css from './JobQueue.module.scss';
import settingsConfig, { Settings } from './JobQueue.settings';
import ManageJob from './ManageJob';
import RPStatsOverview from './RPStats';

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
  const [ pageState, setPageState ] = useState<{isLoading: boolean}>({ isLoading: true });
  const {
    settings,
    updateSettings,
    resetSettings,
  } = useSettings<Settings>(settingsConfig);

  const fetchResourcePools = useFetchResourcePools(canceler);
  const isJobOrderAvailable = !!selectedRp && orderedSchedulers.has(selectedRp.schedulerType);

  const fetchAll = useCallback(async () => {
    if (!selectedRp?.name) return;

    try {
      const orderBy = settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
      const promises = [
        getJobQ(
          {
            limit: settings.tableLimit,
            offset: settings.tableOffset,
            orderBy,
            resourcePool: selectedRp.name,
          },
          { signal: canceler.signal },
        ),
        getJobQStats({}, { signal: canceler.signal }),
      ] as [ Promise<GetJobsResponse>, Promise<Api.V1GetJobQueueStatsResponse> ];

      const [ jobs, stats ] = await Promise.all(promises);

      // Process jobs response.
      setJobs(jobs.jobs);
      if (jobs.pagination.total) setTotal(jobs.pagination.total);

      // Process job stats response.
      setRpStats(stats.results.sort((a, b) => a.resourcePool.localeCompare(b.resourcePool)));
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicSubject: 'Unable to fetch job queue and stats.',
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      setPageState(cur => ({ ...cur, isLoading: false }));
    }
  }, [ canceler.signal, selectedRp?.name, settings ]);

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
    };
    if (selectedRp && isJobOrderAvailable &&
        job.summary.jobsAhead > 0 && canManageJob(job, selectedRp) &&
        !unsupportedQPosSchedulers.has(selectedRp.schedulerType)) {
      triggers[JobAction.MoveToTop] = () => moveJobToPosition(jobs, job.jobId, 1);
    }

    // if job is an experiment type add action to kill it
    if (job.type === JobType.EXPERIMENT) {
      triggers[JobAction.Kill] = async () => {
        await killExperiment({ experimentId: parseInt(job.entityId, 10) });
      };
    }

    if (canManageJob(job, selectedRp)) {
      triggers[JobAction.ManageJob] = () => setManagingJob(job);
    }

    Object.keys(triggers).forEach(key => {
      const action = key as JobAction;
      const fn = triggers[action];
      if (!fn) return;
      triggers[action] = async () => {
        await fn();
        await fetchAll();
      };
    });
    return triggers;
  }, [ isJobOrderAvailable, jobs, selectedRp, fetchAll ]);

  const onModalClose = useCallback(() => {
    setManagingJob(undefined);
    fetchAll();
  }, [ fetchAll ]);

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
          if (!isJobOrderAvailable) {
            col.sorter = undefined;
            col.title = 'Preemptible';
            col.render = (_: unknown, record) => {
              return (
                <div className={css.centerVertically}>
                  {checkmarkRenderer(record.isPreemptible)}
                </div>
              );
            };
          } else {
            col.sorter = (a: Job, b: Job): number =>
              numericSorter(a.summary.jobsAhead, b.summary.jobsAhead);
            col.title = '#';
            col.render = (_: unknown, record) => {
              return (
                <div className={css.centerVertically}>
                  {record.summary.jobsAhead}
                  {!record.isPreemptible && <Icon name="lock" title="Not Preemtible" />}
                </div>
              );
            };
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
  }, [ dropDownOnTrigger, selectedRp, jobs, isJobOrderAvailable ]);

  useEffect(() => {
    if (resourcePools.length === 0) {
      if (selectedRp) {
        resetSettings([ 'selectedRp' ]);
        setSelectedRp(undefined);
      }
      return;
    } else if (selectedRp) return;

    let pool: ResourcePool | undefined = undefined;
    if (settings.selectedPool) {
      pool = resourcePools.find(pool => pool.name === settings.selectedPool);
    }
    if (!pool) {
      pool = resourcePools[0];
    }
    updateSettings({ selectedPool: pool.name });
    setSelectedRp(pool);

  }, [ resourcePools, selectedRp, updateSettings, resetSettings, settings.selectedPool ]);

  useEffect(() => {
    fetchResourcePools();
    return () => canceler.abort();
  }, [ canceler, fetchResourcePools ]);

  useEffect(() => {
    setPageState(cur => ({ ...cur, isLoading: true }));
    fetchAll();
    return () => canceler.abort();
  }, [
    fetchAll,
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
      const rp = resourcePools.find(rp => rp.name === rpName);
      if (!rp) return;
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
    <Page className={css.base} id="jobs" title="Job Queue by Resource Pool">
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
          loading={pageState.isLoading}
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
          initialPool={selectedRp.name}
          job={managingJob}
          jobs={jobs}
          rpStats={rpStats}
          schedulerType={selectedRp.schedulerType}
          onFinish={onModalClose}
        />
      )}

    </Page>
  );
};

export default JobQueue;
