import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Page from 'components/Page';
import Section from 'components/Section';
import InteractiveTable, { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  checkmarkRenderer,
  defaultRowClassName,
  getFullPaginationConfig,
  userRenderer,
} from 'components/Table/Table';
import { V1SchedulerTypeToLabel } from 'constants/states';
import { useSettings } from 'hooks/useSettings';
import { columns as defaultColumns, SCHEDULING_VAL_KEY } from 'pages/JobQueue/JobQueue.table';
import { paths } from 'routes/utils';
import { cancelExperiment, getJobQ, getJobQStats, killExperiment, killTask } from 'services/api';
import * as Api from 'services/api-ts-sdk';
import ActionDropdown, { Triggers } from 'shared/components/ActionDropdown/ActionDropdown';
import Icon from 'shared/components/Icon/Icon';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { numericSorter } from 'shared/utils/sort';
import { capitalize } from 'shared/utils/string';
import { useClusterStore, useRefetchClusterData } from 'stores/cluster';
import usersStore from 'stores/users';
import { Job, JobAction, JobState, JobType, ResourcePool, RPStats } from 'types';
import handleError from 'utils/error';
import {
  canManageJob,
  jobTypeToCommandType,
  moveJobToTop,
  orderedSchedulers,
  unsupportedQPosSchedulers,
} from 'utils/job';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './JobQueue.module.scss';
import settingsConfig, { Settings } from './JobQueue.settings';
import ManageJob from './ManageJob';

interface Props {
  bodyNoPadding?: boolean;
  jobState: JobState;
  selectedRp: ResourcePool;
}

const JobQueue: React.FC<Props> = ({ bodyNoPadding, selectedRp, jobState }) => {
  const users = Loadable.map(useObservable(usersStore.getUsers()), ({ users }) => users);
  useRefetchClusterData();
  const resourcePools = useObservable(useClusterStore().resourcePools);
  const [managingJob, setManagingJob] = useState<Job>();
  const [rpStats, setRpStats] = useState<RPStats[]>(() => {
    if (Loadable.isLoading(resourcePools)) return [];

    return resourcePools.data.map(
      (rp) =>
        ({
          resourcePool: rp.name,
          stats: { preemptibleCount: 0, queuedCount: 0, scheduledCount: 0 },
        } as RPStats),
    );
  });
  const [jobs, setJobs] = useState<Job[]>([]);
  const [topJob, setTopJob] = useState<Job>();
  const [total, setTotal] = useState(0);
  const [canceler] = useState(new AbortController());
  const [pageState, setPageState] = useState<{ isLoading: boolean }>({ isLoading: true });
  const pageRef = useRef<HTMLElement>(null);

  const { settings, updateSettings } = useSettings<Settings>(settingsConfig(jobState));
  const settingsColumns = useMemo(() => [...settings.columns], [settings.columns]);

  const isJobOrderAvailable = orderedSchedulers.has(selectedRp.schedulerType);

  const fetchAll = useCallback(async () => {
    if (!settings) return;

    try {
      const orderBy = settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
      const [jobs, stats] = await Promise.all([
        getJobQ(
          {
            limit: settings.tableLimit,
            offset: settings.tableOffset,
            orderBy,
            resourcePool: selectedRp.name,
            states: jobState ? [jobState] : undefined,
          },
          { signal: canceler.signal },
        ),
        getJobQStats({}, { signal: canceler.signal }),
      ]);

      const firstJobResp = await getJobQ(
        {
          limit: 1,
          offset: 0,
          resourcePool: selectedRp.name,
        },
        { signal: canceler.signal },
      );
      const firstJob = firstJobResp.jobs[0];

      // Process jobs response.
      if (firstJob && !isEqual(firstJob, topJob)) setTopJob(firstJob);
      setJobs(jobState ? jobs.jobs.filter((j) => j.summary.state === jobState) : jobs.jobs);
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
      setPageState((cur) => ({ ...cur, isLoading: false }));
    }
  }, [canceler.signal, selectedRp.name, settings, jobState, topJob]);

  useEffect(() => {
    fetchAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const rpTotalJobCount = useCallback(
    (rpName: string) => {
      const stats = rpStats.find((rp) => rp.resourcePool === rpName)?.stats;
      return stats ? stats.queuedCount + stats.scheduledCount : 0;
    },
    [rpStats],
  );

  const dropDownOnTrigger = useCallback(
    (job: Job) => {
      const triggers: Triggers<JobAction> = {};
      const commandType = jobTypeToCommandType(job.type);

      if (commandType) {
        triggers[JobAction.Kill] = () => {
          killTask({ id: job.entityId, type: commandType });
        };
        triggers[JobAction.ViewLog] = () => {
          routeToReactUrl(paths.taskLogs({ id: job.entityId, name: job.name, type: commandType }));
        };
      }

      if (
        selectedRp &&
        isJobOrderAvailable &&
        !!topJob &&
        job.summary.jobsAhead > 0 &&
        canManageJob(job, selectedRp) &&
        !unsupportedQPosSchedulers.has(selectedRp.schedulerType)
      ) {
        triggers[JobAction.MoveToTop] = () => moveJobToTop(topJob, job);
      }

      // if job is an experiment type add action to kill it
      if (job.type === JobType.EXPERIMENT) {
        triggers[JobAction.Cancel] = async () => {
          await cancelExperiment({ experimentId: parseInt(job.entityId, 10) });
        };
        triggers[JobAction.Kill] = async () => {
          await killExperiment({ experimentId: parseInt(job.entityId, 10) });
        };
      }

      if (canManageJob(job, selectedRp)) {
        triggers[JobAction.ManageJob] = () => setManagingJob(job);
      }

      Object.keys(triggers).forEach((key) => {
        const action = key as JobAction;
        const fn = triggers[action];
        if (!fn) return;
        triggers[action] = async () => {
          await fn();
          await fetchAll();
        };
      });
      return triggers;
    },
    [selectedRp, isJobOrderAvailable, topJob, fetchAll],
  );

  const onModalClose = useCallback(() => {
    setManagingJob(undefined);
    fetchAll();
  }, [fetchAll]);

  useEffect(() => {
    if (!managingJob) return;
    const job = jobs.find((j) => j.jobId === managingJob.jobId);
    if (!job) {
      setManagingJob(undefined);
    } else if (!isEqual(job, managingJob)) {
      setManagingJob(job);
    }
  }, [jobs, managingJob]);

  useEffect(() => {
    const col = defaultColumns.find(({ key }) => key === SCHEDULING_VAL_KEY);
    if (col) {
      const replaceIndex = settingsColumns.findIndex((column) =>
        ['priority', 'weight', 'resourcePool'].includes(column),
      );
      const newColumns = [...settingsColumns];
      if (replaceIndex !== -1) newColumns[replaceIndex] = col.dataIndex;
      if (!isEqual(newColumns, settings.columns)) updateSettings({ columns: newColumns });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings.columns, settingsColumns]);

  const columns = useMemo(() => {
    const matchUsers = Loadable.match(users, {
      Loaded: (users) => users,
      NotLoaded: () => [],
    });

    return defaultColumns
      .map((col) => {
        switch (col.key) {
          case 'actions':
            col.render = (_, record) => {
              return (
                <div>
                  <ActionDropdown<JobAction>
                    actionOrder={[
                      JobAction.ManageJob,
                      JobAction.MoveToTop,
                      JobAction.ViewLog,
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
                    onError={handleError}
                    onTrigger={dropDownOnTrigger(record)}
                  />
                </div>
              );
            };
            break;
          case SCHEDULING_VAL_KEY: {
            if (!settingsColumns) break;

            switch (selectedRp.schedulerType) {
              case Api.V1SchedulerType.SLURM:
                col.title = 'Partition';
                col.dataIndex = 'resourcePool';
                break;
              case Api.V1SchedulerType.PBS:
                col.title = 'Queue';
                col.dataIndex = 'resourcePool';
                break;
              case Api.V1SchedulerType.PRIORITY:
              case Api.V1SchedulerType.KUBERNETES:
                col.title = 'Priority';
                col.dataIndex = 'priority';
                break;
              case Api.V1SchedulerType.FAIRSHARE:
                col.title = 'Weight';
                col.dataIndex = 'weight';
                col.align = 'right';
                break;
            }
            break;
          }
          case 'jobsAhead':
            if (!isJobOrderAvailable) {
              col.sorter = undefined;
              col.title = 'Preemptible';
              col.render = (_: unknown, record) => {
                return (
                  <div className={`${css.centerVertically} ${css.centerHorizontally}`}>
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
          case 'user':
            col.render = (_, r) => userRenderer(matchUsers.find((u) => u.id === r.userId));
            break;
        }
        return col;
      })
      .map((column) => {
        column.sortOrder = null;
        if (column.key === settings.sortKey) {
          column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
        }
        return column;
      });

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    isJobOrderAvailable,
    dropDownOnTrigger,
    settingsColumns,
    settings.sortKey,
    settings.sortDesc,
    selectedRp.schedulerType,
  ]);

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
  }, [selectedRp]);

  return (
    <Page
      bodyNoPadding={bodyNoPadding}
      className={css.base}
      containerRef={pageRef}
      headerComponent={<div />}
      id="jobs"
      title="Job Queue by Resource Pool">
      <Section hideTitle={!!selectedRp} title={tableTitle}>
        {settings ? (
          <InteractiveTable
            columns={columns}
            containerRef={pageRef}
            dataSource={jobs}
            loading={pageState.isLoading}
            pagination={getFullPaginationConfig(
              {
                limit: settings.tableLimit,
                offset: settings.tableOffset,
              },
              total,
            )}
            rowClassName={defaultRowClassName({ clickable: false })}
            rowKey="jobId"
            scroll={{ x: 1000 }}
            settings={settings as InteractiveTableSettings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings}
          />
        ) : (
          <SkeletonTable columns={columns.length} />
        )}
      </Section>
      {!!managingJob && (
        <ManageJob
          initialPool={selectedRp.name}
          job={managingJob}
          jobCount={rpTotalJobCount(selectedRp.name)}
          rpStats={rpStats}
          schedulerType={selectedRp.schedulerType}
          onFinish={onModalClose}
        />
      )}
    </Page>
  );
};

export default JobQueue;
